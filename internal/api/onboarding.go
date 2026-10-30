package api

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

type supportedCountry struct {
	Code string `json:"code"`
	Name string `json:"name"`
	CallingCode string `json:"calling_code"`
}

// This is a product-availability allowlist, not a sanctions determination.
// Final eligibility must also consider residence, identity, ownership and transactions.
var onboardingCountries = []supportedCountry{
	{"AT","Austria","+43"},{"BE","Belgium","+32"},{"BG","Bulgaria","+359"},
	{"HR","Croatia","+385"},{"CY","Cyprus","+357"},{"CZ","Czechia","+420"},
	{"DK","Denmark","+45"},{"EE","Estonia","+372"},{"FI","Finland","+358"},
	{"FR","France","+33"},{"DE","Germany","+49"},{"GR","Greece","+30"},
	{"HU","Hungary","+36"},{"IE","Ireland","+353"},{"IT","Italy","+39"},
	{"LV","Latvia","+371"},{"LT","Lithuania","+370"},{"LU","Luxembourg","+352"},
	{"MT","Malta","+356"},{"NL","Netherlands","+31"},{"PL","Poland","+48"},
	{"PT","Portugal","+351"},{"RO","Romania","+40"},{"SK","Slovakia","+421"},
	{"SI","Slovenia","+386"},{"ES","Spain","+34"},{"SE","Sweden","+46"},
}

var e164Pattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

func (s *Server) onboardingCountries(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": onboardingCountries, "policy": "Supported phone regions only. This is not a sanctions or KYC decision."})
}

type sendOTPRequest struct {
	CountryCode string `json:"country_code"`
	PhoneE164 string `json:"phone_e164"`
}

func (s *Server) sendPhoneOTP(w http.ResponseWriter, r *http.Request) {
	var in sendOTPRequest
	if !decode(w, r, &in) { return }
	in.CountryCode = strings.ToUpper(strings.TrimSpace(in.CountryCode))
	in.PhoneE164 = strings.TrimSpace(in.PhoneE164)
	country, ok := findOnboardingCountry(in.CountryCode)
	if !ok || !e164Pattern.MatchString(in.PhoneE164) || !strings.HasPrefix(in.PhoneE164, country.CallingCode) {
		writeError(w, http.StatusBadRequest, "unsupported_phone_region", "Choose a supported country and enter a phone number in international format")
		return
	}

	// A real SMS provider must be configured before production onboarding is enabled.
	if strings.EqualFold(os.Getenv("APP_ENV"), "production") {
		writeError(w, http.StatusServiceUnavailable, "sms_provider_unconfigured", "Phone verification is temporarily unavailable")
		return
	}

	ctx := r.Context()
	var recent int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM phone_otp_challenges WHERE phone_e164=$1 AND created_at>now()-interval '1 hour'`, in.PhoneE164).Scan(&recent); err != nil {
		writeError(w, 500, "internal_error", "Could not start phone verification"); return
	}
	if recent >= 5 {
		writeError(w, http.StatusTooManyRequests, "otp_rate_limited", "Too many verification codes requested. Try again later."); return
	}
	var last time.Time
	err := s.db.QueryRow(ctx, `SELECT created_at FROM phone_otp_challenges WHERE phone_e164=$1 ORDER BY created_at DESC LIMIT 1`, in.PhoneE164).Scan(&last)
	if err == nil && time.Since(last) < time.Minute {
		writeError(w, http.StatusTooManyRequests, "otp_cooldown", "Wait before requesting another code"); return
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "internal_error", "Could not start phone verification"); return
	}

	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil { writeError(w, 500, "internal_error", "Could not generate verification code"); return }
	code := fmt.Sprintf("%06d", n.Int64())
	var challengeID string
	hash := otpHash("pending", code)
	err = s.db.QueryRow(ctx, `INSERT INTO phone_otp_challenges(phone_e164,code_hash,expires_at) VALUES($1,$2,now()+interval '5 minutes') RETURNING id`, in.PhoneE164, hash).Scan(&challengeID)
	if err != nil { writeError(w, 500, "internal_error", "Could not start phone verification"); return }
	// Bind the hash to the generated challenge ID after insertion.
	hash = otpHash(challengeID, code)
	if _, err := s.db.Exec(ctx, `UPDATE phone_otp_challenges SET code_hash=$2 WHERE id=$1`, challengeID, hash); err != nil {
		writeError(w, 500, "internal_error", "Could not start phone verification"); return
	}
	s.logger.Warn("development OTP generated; use only in local development", "challenge_id", challengeID, "phone_suffix", phoneSuffix(in.PhoneE164), "code", code)
	response := map[string]any{"challenge_id": challengeID, "expires_in_seconds": 300, "resend_after_seconds": 60}
	if !strings.EqualFold(os.Getenv("APP_ENV"), "production") { response["development_code"] = code }
	writeJSON(w, http.StatusCreated, response)
}

type verifyOTPRequest struct {
	ChallengeID string `json:"challenge_id"`
	Code string `json:"code"`
}

func (s *Server) verifyPhoneOTP(w http.ResponseWriter, r *http.Request) {
	var in verifyOTPRequest
	if !decode(w, r, &in) { return }
	if !validUUID(in.ChallengeID) || len(in.Code) != 6 || !digitsOnly(in.Code) {
		writeError(w, 400, "validation_error", "A challenge ID and six-digit code are required"); return
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil { writeError(w, 500, "internal_error", "Could not verify code"); return }
	defer tx.Rollback(ctx)
	var phone, storedHash string
	var attempts int
	var expires time.Time
	var consumed sql.NullTime
	err = tx.QueryRow(ctx, `SELECT phone_e164,code_hash,attempts,expires_at,consumed_at FROM phone_otp_challenges WHERE id=$1 FOR UPDATE`, in.ChallengeID).Scan(&phone,&storedHash,&attempts,&expires,&consumed)
	if errors.Is(err, pgx.ErrNoRows) { writeError(w, 400, "invalid_code", "Verification code is invalid or expired"); return }
	if err != nil { writeError(w, 500, "internal_error", "Could not verify code"); return }
	if consumed.Valid || time.Now().After(expires) || attempts >= 5 || storedHash != otpHash(in.ChallengeID,in.Code) {
		if !consumed.Valid && attempts < 5 {
			_, _ = tx.Exec(ctx, `UPDATE phone_otp_challenges SET attempts=LEAST(attempts+1,5) WHERE id=$1`, in.ChallengeID)
			_ = tx.Commit(ctx)
		}
		writeError(w, 400, "invalid_code", "Verification code is invalid or expired"); return
	}
	if _, err := tx.Exec(ctx, `UPDATE phone_otp_challenges SET consumed_at=now() WHERE id=$1`, in.ChallengeID); err != nil {
		writeError(w, 500, "internal_error", "Could not verify code"); return
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil { writeError(w,500,"internal_error","Could not complete verification"); return }
	setupToken := base64.RawURLEncoding.EncodeToString(tokenBytes)
	sum := sha256.Sum256([]byte(setupToken))
	if _, err := tx.Exec(ctx, `INSERT INTO onboarding_tokens(token_hash,phone_e164,expires_at) VALUES($1,$2,now()+interval '10 minutes')`, hex.EncodeToString(sum[:]), phone); err != nil {
		writeError(w,500,"internal_error","Could not complete verification"); return
	}
	if err := tx.Commit(ctx); err != nil { writeError(w,500,"internal_error","Could not complete verification"); return }
	writeJSON(w,200,map[string]any{"verification_token":setupToken,"expires_in_seconds":600})
}

type setPINRequest struct { VerificationToken string `json:"verification_token"`; PIN string `json:"pin"` }

func (s *Server) setAppPIN(w http.ResponseWriter, r *http.Request) {
	var in setPINRequest
	if !decode(w,r,&in) { return }
	if !validPIN(in.PIN) { writeError(w,400,"weak_pin","PIN must contain six digits and must not be repeated or sequential"); return }
	if len(in.VerificationToken)<32 || len(in.VerificationToken)>128 { writeError(w,401,"invalid_verification","Phone verification is required"); return }
	sum:=sha256.Sum256([]byte(in.VerificationToken))
	hashToken:=hex.EncodeToString(sum[:])
	ctx:=r.Context()
	tx,err:=s.db.Begin(ctx)
	if err!=nil { writeError(w,500,"internal_error","Could not create account"); return }
	defer tx.Rollback(ctx)
	var phone string
	err=tx.QueryRow(ctx,`SELECT phone_e164 FROM onboarding_tokens WHERE token_hash=$1 AND consumed_at IS NULL AND expires_at>now() FOR UPDATE`,hashToken).Scan(&phone)
	if errors.Is(err,pgx.ErrNoRows) { writeError(w,401,"invalid_verification","Phone verification is missing or expired"); return }
	if err!=nil { writeError(w,500,"internal_error","Could not create account"); return }
	pinHash,err:=bcrypt.GenerateFromPassword([]byte(in.PIN),bcrypt.DefaultCost)
	if err!=nil { writeError(w,500,"internal_error","Could not secure PIN"); return }
	var userID string
	err=tx.QueryRow(ctx,`INSERT INTO users(phone_e164,app_pin_hash,full_name,country_code,preferred_currency,language_code) VALUES($1,$2,'',$3,'EUR','en') RETURNING id`,phone,string(pinHash),countryForPhone(phone)).Scan(&userID)
	if err!=nil {
		var pgErr *pgconn.PgError
		if errors.As(err,&pgErr) && pgErr.Code=="23505" { writeError(w,409,"phone_exists","This phone number is already registered") } else { writeError(w,500,"internal_error","Could not create account") }
		return
	}
	if _,err=tx.Exec(ctx,`UPDATE onboarding_tokens SET consumed_at=now() WHERE token_hash=$1`,hashToken); err!=nil { writeError(w,500,"internal_error","Could not complete registration"); return }
	if err=tx.Commit(ctx); err!=nil { writeError(w,500,"internal_error","Could not complete registration"); return }
	accessToken,expires,err:=s.tokens.Issue(userID)
	if err!=nil { writeError(w,500,"internal_error","Could not start session"); return }
	writeJSON(w,201,map[string]any{"user":map[string]any{"id":userID,"phone_e164":phone,"email":nil,"country_code":countryForPhone(phone)},"access_token":accessToken,"token_type":"Bearer","expires_at":expires,"next_step":"choose_card"})
}

type phoneLoginRequest struct { PhoneE164 string `json:"phone_e164"`; PIN string `json:"pin"` }
func (s *Server) phoneLogin(w http.ResponseWriter,r *http.Request) {
	var in phoneLoginRequest
	if !decode(w,r,&in) { return }
	if !e164Pattern.MatchString(in.PhoneE164) || len(in.PIN)!=6 || !digitsOnly(in.PIN) { writeError(w,400,"validation_error","Phone and six-digit PIN are required"); return }
	ctx:=r.Context()
	var lockedUntil sql.NullTime
	if err:=s.db.QueryRow(ctx,`SELECT locked_until FROM pin_login_attempts WHERE phone_e164=$1`,in.PhoneE164).Scan(&lockedUntil); err!=nil && !errors.Is(err,pgx.ErrNoRows) {
		writeError(w,500,"internal_error","Could not log in"); return
	}
	if lockedUntil.Valid && time.Now().Before(lockedUntil.Time) {
		writeError(w,http.StatusTooManyRequests,"login_temporarily_locked","Too many incorrect PIN attempts. Try again later."); return
	}
	var id,hash string
	err:=s.db.QueryRow(ctx,`SELECT id,app_pin_hash FROM users WHERE phone_e164=$1`,in.PhoneE164).Scan(&id,&hash)
	if errors.Is(err,pgx.ErrNoRows) || err==nil && bcrypt.CompareHashAndPassword([]byte(hash),[]byte(in.PIN))!=nil {
		_,_=s.db.Exec(ctx,`INSERT INTO pin_login_attempts(phone_e164,failed_attempts,updated_at) VALUES($1,1,now())
			ON CONFLICT(phone_e164) DO UPDATE SET failed_attempts=pin_login_attempts.failed_attempts+1,
			locked_until=CASE WHEN pin_login_attempts.failed_attempts+1>=5 THEN now()+interval '15 minutes' ELSE pin_login_attempts.locked_until END,
			updated_at=now()`,in.PhoneE164)
		writeError(w,401,"invalid_credentials","Phone number or PIN is incorrect"); return
	}
	if err!=nil { writeError(w,500,"internal_error","Could not log in"); return }
	_,_=s.db.Exec(ctx,`DELETE FROM pin_login_attempts WHERE phone_e164=$1`,in.PhoneE164)
	token,exp,err:=s.tokens.Issue(id)
	if err!=nil { writeError(w,500,"internal_error","Could not start session"); return }
	writeJSON(w,200,map[string]any{"access_token":token,"token_type":"Bearer","expires_at":exp})
}

type optionalEmailRequest struct { Email string `json:"email"` }
func (s *Server) addOptionalEmail(w http.ResponseWriter,r *http.Request) {
	var in optionalEmailRequest
	if !decode(w,r,&in) { return }
	in.Email=strings.ToLower(strings.TrimSpace(in.Email))
	if _,err:=mail.ParseAddress(in.Email); err!=nil || len(in.Email)>254 { writeError(w,400,"validation_error","Enter a valid email address"); return }
	tag,err:=s.db.Exec(r.Context(),`UPDATE users SET email=$2,updated_at=now() WHERE id=$1`,uid(r),in.Email)
	if err!=nil {
		var pgErr *pgconn.PgError
		if errors.As(err,&pgErr)&&pgErr.Code=="23505" { writeError(w,409,"email_exists","Email is already in use") } else { writeError(w,500,"internal_error","Could not save email") }
		return
	}
	if tag.RowsAffected()==0 { writeError(w,404,"not_found","User not found"); return }
	writeJSON(w,200,map[string]any{"email":in.Email,"status":"saved"})
}

type chooseCardRequest struct { Scheme string `json:"scheme"` }
func (s *Server) chooseCard(w http.ResponseWriter,r *http.Request) {
	var in chooseCardRequest
	if !decode(w,r,&in) { return }
	in.Scheme=strings.ToLower(strings.TrimSpace(in.Scheme))
	if in.Scheme!="visa" && in.Scheme!="mastercard" { writeError(w,400,"validation_error","scheme must be visa or mastercard"); return }
	ctx:=r.Context()
	tx,err:=s.db.Begin(ctx)
	if err!=nil { writeError(w,500,"internal_error","Could not open account"); return }
	defer tx.Rollback(ctx)
	var accountID string
	err=tx.QueryRow(ctx,`SELECT account_id FROM issued_cards WHERE user_id=$1 AND scheme=$2`,uid(r),in.Scheme).Scan(&accountID)
	if err==nil { writeJSON(w,200,map[string]any{"account_id":accountID,"scheme":in.Scheme,"status":"requested","replayed":true}); return }
	if !errors.Is(err,pgx.ErrNoRows) { writeError(w,500,"internal_error","Could not open account"); return }
	err=tx.QueryRow(ctx,`INSERT INTO accounts(user_id,name,currency,balance_minor) VALUES($1,'Main account','EUR',0) RETURNING id`,uid(r)).Scan(&accountID)
	if err!=nil { writeError(w,500,"internal_error","Could not open account"); return }
	if _,err=tx.Exec(ctx,`INSERT INTO issued_cards(user_id,account_id,scheme,status) VALUES($1,$2,$3,'requested')`,uid(r),accountID,in.Scheme); err!=nil { writeError(w,500,"internal_error","Could not save card selection"); return }
	if err=tx.Commit(ctx); err!=nil { writeError(w,500,"internal_error","Could not open account"); return }
	writeJSON(w,201,map[string]any{"account_id":accountID,"scheme":in.Scheme,"card_status":"requested","account_status":"active","notice":"A prototype account was opened. Actual Visa/Mastercard issuance requires an approved issuer and card programme."})
}

func findOnboardingCountry(code string)(supportedCountry,bool){ for _,c:=range onboardingCountries { if c.Code==code{return c,true} }; return supportedCountry{},false }
func countryForPhone(phone string) string { best:=""; code:=""; for _,c:=range onboardingCountries { if strings.HasPrefix(phone,c.CallingCode)&&len(c.CallingCode)>len(best){best=c.CallingCode;code=c.Code} }; return code }
func otpHash(challenge,code string) string { sum:=sha256.Sum256([]byte(challenge+":"+code)); return hex.EncodeToString(sum[:]) }
func phoneSuffix(phone string) string { if len(phone)<4{return "****"}; return phone[len(phone)-4:] }
func digitsOnly(v string) bool { for _,r:=range v { if r<'0'||r>'9'{return false} }; return true }
func validPIN(pin string) bool {
	if len(pin)!=6 || !digitsOnly(pin) { return false }
	allSame:=true; ascending:=true; descending:=true
	for i:=1;i<len(pin);i++ { if pin[i]!=pin[0]{allSame=false}; if pin[i]!=pin[i-1]+1{ascending=false}; if pin[i]+1!=pin[i-1]{descending=false} }
	return !allSame&&!ascending&&!descending
}
