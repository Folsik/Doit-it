package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/folsik/doit-it/internal/auth"
)

type Server struct {
	db     *pgxpool.Pool
	tokens *auth.Manager
	logger *slog.Logger
}

func New(db *pgxpool.Pool, tokens *auth.Manager, logger *slog.Logger) *Server {
	return &Server{db: db, tokens: tokens, logger: logger}
}
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("POST /v1/auth/register", s.register)
	mux.HandleFunc("POST /v1/auth/login", s.login)
	mux.Handle("POST /v1/auth/logout", s.protect(http.HandlerFunc(s.logout)))
	mux.Handle("GET /v1/me", s.protect(http.HandlerFunc(s.me)))
	mux.Handle("PATCH /v1/me", s.protect(http.HandlerFunc(s.updateProfile)))
	mux.Handle("GET /v1/dashboard", s.protect(http.HandlerFunc(s.dashboard)))
	mux.Handle("GET /v1/accounts", s.protect(http.HandlerFunc(s.accounts)))
	mux.Handle("GET /v1/transactions", s.protect(http.HandlerFunc(s.transactions)))
	mux.Handle("GET /v1/transactions/{id}", s.protect(http.HandlerFunc(s.transactionDetail)))
	mux.Handle("POST /v1/transfers", s.protect(http.HandlerFunc(s.transfer)))
	mux.Handle("GET /v1/security/sessions", s.protect(http.HandlerFunc(s.sessions)))
	mux.Handle("DELETE /v1/security/sessions/{id}", s.protect(http.HandlerFunc(s.revokeSession)))
	return requestID(recoverer(s.logger)(cors(mux)))
}

type contextKey string

const (
	userIDKey contextKey = "userID"
	tokenClaimsKey contextKey = "tokenClaims"
)

func (s *Server) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Fields(r.Header.Get("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Bearer token required")
			return
		}
		h := parts[1]
		claims, err := s.tokens.Verify(h)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired access token")
			return
		}
		var revoked bool
		if err := s.db.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM revoked_tokens WHERE jti=$1 AND expires_at>now())`, claims.TokenID).Scan(&revoked); err != nil {
			s.logger.Error("token revocation check failed", "error", err)
			writeError(w, http.StatusServiceUnavailable, "unavailable", "Authentication service temporarily unavailable")
			return
		}
		if revoked {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Access token has been revoked")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
		ctx = context.WithValue(ctx, tokenClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func uid(r *http.Request) string { v, _ := r.Context().Value(userIDKey).(string); return v }
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.db.Ping(ctx); err != nil {
		writeError(w, 503, "unavailable", "Database unavailable")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}

type registerRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	FullName    string `json:"full_name"`
	CountryCode string `json:"country_code"`
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in registerRequest
	if !decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.FullName = strings.TrimSpace(in.FullName)
	parsedEmail, emailErr := mail.ParseAddress(in.Email)
	if emailErr != nil || parsedEmail.Address != in.Email || len(in.Email) > 254 ||
		len(in.Password) < 12 || len(in.Password) > 72 ||
		in.FullName == "" || len(in.FullName) > 120 ||
		(in.CountryCode != "" && (len(in.CountryCode) != 2 || !asciiLetters(in.CountryCode))) {
		writeError(w, http.StatusBadRequest, "validation_error", "Provide a valid email, a password of 12-72 bytes, a full name (max 120 characters), and an optional two-letter country code")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, 500, "internal_error", "Could not create user")
		return
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		writeError(w, 500, "internal_error", "Could not create user")
		return
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,country_code) VALUES($1,$2,$3,NULLIF($4,'')) RETURNING id`, in.Email, string(hash), in.FullName, strings.ToUpper(in.CountryCode)).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			writeError(w, http.StatusConflict, "email_exists", "Email is already registered")
		} else {
			s.logger.Error("register user failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not create user")
		}
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO accounts(user_id,name,currency,balance_minor) VALUES($1,'Main account','EUR',0)`, id); err != nil {
		writeError(w, 500, "internal_error", "Could not create account")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(w, 500, "internal_error", "Could not create user")
		return
	}
	token, exp, err := s.tokens.Issue(id)
	if err != nil {
		writeError(w, 500, "internal_error", "Could not issue access token")
		return
	}
	writeJSON(w, 201, map[string]any{"user": map[string]any{"id": id, "email": in.Email, "full_name": in.FullName, "country_code": strings.ToUpper(in.CountryCode), "preferred_currency": "EUR", "language_code": "en"}, "access_token": token, "token_type": "Bearer", "expires_at": exp, "notice": "Prototype account only; no real funds or regulated banking services are connected."})
}

const dummyBcryptHash = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in loginRequest
	if !decode(w, r, &in) {
		return
	}
	var id, hash string
	err := s.db.QueryRow(r.Context(), `SELECT id,password_hash FROM users WHERE email=$1`, strings.ToLower(strings.TrimSpace(in.Email))).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(in.Password))
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect")
		return
	}
	if err != nil {
		s.logger.Error("login lookup failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not log in")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Email or password is incorrect")
		return
	}
	token, exp, err := s.tokens.Issue(id)
	if err != nil {
		writeError(w, 500, "internal_error", "Could not issue access token")
		return
	}
	writeJSON(w, 200, map[string]any{"access_token": token, "token_type": "Bearer", "expires_at": exp})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(tokenClaimsKey).(auth.Claims)
	if !ok || claims.TokenID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Token claims are missing")
		return
	}
	_, err := s.db.Exec(r.Context(), `INSERT INTO revoked_tokens(jti,expires_at) VALUES($1,to_timestamp($2)) ON CONFLICT(jti) DO NOTHING`, claims.TokenID, claims.ExpiresAt)
	if err != nil {
		s.logger.Error("revoke access token failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not log out")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "logged_out"})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	var u userDTO
	err := s.db.QueryRow(r.Context(), `SELECT id,email,full_name,COALESCE(country_code,''),preferred_currency,language_code,two_factor_enabled,created_at FROM users WHERE id=$1`, uid(r)).Scan(&u.ID, &u.Email, &u.FullName, &u.CountryCode, &u.PreferredCurrency, &u.LanguageCode, &u.TwoFactorEnabled, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "User not found")
		} else {
			s.logger.Error("load profile failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load profile")
		}
		return
	}
	writeJSON(w, 200, u)
}

type userDTO struct {
	ID                string    `json:"id"`
	Email             string    `json:"email"`
	FullName          string    `json:"full_name"`
	CountryCode       string    `json:"country_code"`
	PreferredCurrency string    `json:"preferred_currency"`
	LanguageCode      string    `json:"language_code"`
	TwoFactorEnabled  bool      `json:"two_factor_enabled"`
	CreatedAt         time.Time `json:"created_at"`
}
type profileRequest struct {
	FullName          string `json:"full_name"`
	CountryCode       string `json:"country_code"`
	PreferredCurrency string `json:"preferred_currency"`
	LanguageCode      string `json:"language_code"`
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var in profileRequest
	if !decode(w, r, &in) {
		return
	}
	in.FullName = strings.TrimSpace(in.FullName)
	if len(in.FullName) > 120 || (in.PreferredCurrency != "" && (len(in.PreferredCurrency) != 3 || !asciiLetters(in.PreferredCurrency))) ||
		len(in.LanguageCode) > 10 || (in.CountryCode != "" && (len(in.CountryCode) != 2 || !asciiLetters(in.CountryCode))) {
		writeError(w, http.StatusBadRequest, "validation_error", "Invalid profile fields")
		return
	}
	_, err := s.db.Exec(r.Context(), `UPDATE users SET full_name=COALESCE(NULLIF($2,''),full_name),country_code=COALESCE(NULLIF($3,''),country_code),preferred_currency=COALESCE(NULLIF($4,''),preferred_currency),language_code=COALESCE(NULLIF($5,''),language_code),updated_at=now() WHERE id=$1`, uid(r), in.FullName, strings.ToUpper(in.CountryCode), strings.ToUpper(in.PreferredCurrency), in.LanguageCode)
	if err != nil {
		writeError(w, 500, "internal_error", "Could not update profile")
		return
	}
	s.me(w, r)
}
func (s *Server) accounts(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id,name,currency,balance_minor,status,created_at FROM accounts WHERE user_id=$1 ORDER BY created_at`, uid(r))
	if err != nil {
		writeError(w, 500, "internal_error", "Could not load accounts")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id, name, currency, status string
		var balance int64
		var created time.Time
		if err := rows.Scan(&id, &name, &currency, &balance, &status, &created); err != nil {
			s.logger.Error("scan account failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load accounts")
			return
		}
		out = append(out, map[string]any{"id": id, "name": name, "currency": strings.TrimSpace(currency), "balance_minor": balance, "status": status, "created_at": created})
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("iterate accounts failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load accounts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id,name,currency,balance_minor,status FROM accounts WHERE user_id=$1 ORDER BY created_at`, uid(r))
	if err != nil {
		writeError(w, 500, "internal_error", "Could not load dashboard")
		return
	}
	defer rows.Close()
	accounts := []map[string]any{}
	var total int64
	for rows.Next() {
		var id, name, currency, status string
		var b int64
		if err := rows.Scan(&id, &name, &currency, &b, &status); err != nil {
			s.logger.Error("scan dashboard account failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load dashboard")
			return
		}
		accounts = append(accounts, map[string]any{"id": id, "name": name, "currency": strings.TrimSpace(currency), "balance_minor": b, "status": status})
		if currency == "EUR" {
			if b > 0 && total > math.MaxInt64-b {
				writeError(w, http.StatusInternalServerError, "balance_overflow", "Could not calculate total balance")
				return
			}
			total += b
		}
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("iterate dashboard accounts failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load dashboard")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"total_balance_minor": total, "currency": "EUR", "accounts": accounts, "spending_this_month_minor": 0, "recent_transactions": []any{}, "note": "Dashboard spending analytics will populate as transaction categories are added."})
}
func (s *Server) transactions(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	if statusFilter != "" && statusFilter != "completed" && statusFilter != "rejected" {
		writeError(w, http.StatusBadRequest, "validation_error", "status must be completed or rejected")
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 || n > 100 {
			writeError(w, 400, "validation_error", "limit must be between 1 and 100")
			return
		}
		limit = n
	}
	offset := 0
	if v := r.URL.Query().Get("offset"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 0 {
			writeError(w, 400, "validation_error", "offset must be >= 0")
			return
		}
		offset = n
	}
	rows, err := s.db.Query(r.Context(), `SELECT t.id,t.reference,t.amount_minor,t.currency,t.note,t.status,t.created_at, a1.user_id, a2.user_id, a1.name, a2.name FROM transfers t JOIN accounts a1 ON a1.id=t.from_account_id JOIN accounts a2 ON a2.id=t.to_account_id WHERE (a1.user_id=$1 OR a2.user_id=$1) AND ($2='' OR t.status=$2) ORDER BY t.created_at DESC LIMIT $3 OFFSET $4`, uid(r), r.URL.Query().Get("status"), limit, offset)
	if err != nil {
		writeError(w, 500, "internal_error", "Could not load transactions")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, ref, currency, note, status, fromUser, toUser, fromName, toName string
		var amount int64
		var created time.Time
		if err := rows.Scan(&id, &ref, &amount, &currency, &note, &status, &created, &fromUser, &toUser, &fromName, &toName); err != nil {
			s.logger.Error("scan transaction failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load transactions")
			return
		}
		signed := amount
		title := toName
		if fromUser == uid(r) {
			signed = -amount
			title = fromName
		}
		items = append(items, map[string]any{"id": id, "reference": ref, "amount_minor": signed, "currency": strings.TrimSpace(currency), "title": title, "note": note, "status": status, "created_at": created, "type": map[bool]string{true: "outgoing", false: "incoming"}[fromUser == uid(r)]})
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("iterate transactions failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load transactions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "limit": limit, "offset": offset})
}
func (s *Server) transactionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !validUUID(id) {
		writeError(w, http.StatusNotFound, "not_found", "Transaction not found")
		return
	}
	var ref, currency, note, status, fromName, toName string
	var amount int64
	var created time.Time
	var fromUser, toUser string
	err := s.db.QueryRow(r.Context(), `SELECT t.reference,t.amount_minor,t.currency,t.note,t.status,t.created_at,a1.user_id,a2.user_id,a1.name,a2.name FROM transfers t JOIN accounts a1 ON a1.id=t.from_account_id JOIN accounts a2 ON a2.id=t.to_account_id WHERE t.id=$1 AND (a1.user_id=$2 OR a2.user_id=$2)`, id, uid(r)).Scan(&ref, &amount, &currency, &note, &status, &created, &fromUser, &toUser, &fromName, &toName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "not_found", "Transaction not found")
		} else {
			s.logger.Error("load transaction detail failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load transaction")
		}
		return
	}
	signed := amount
	if fromUser == uid(r) {
		signed = -amount
	}
	writeJSON(w, 200, map[string]any{"id": id, "reference": ref, "amount_minor": signed, "currency": strings.TrimSpace(currency), "note": note, "status": status, "created_at": created, "counterparty_account_name": map[bool]string{true: toName, false: fromName}[fromUser == uid(r)]})
}

type transferRequest struct {
	FromAccountID  string `json:"from_account_id"`
	ToAccountID    string `json:"to_account_id"`
	AmountMinor    int64  `json:"amount_minor"`
	Currency       string `json:"currency"`
	Note           string `json:"note"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (s *Server) transfer(w http.ResponseWriter, r *http.Request) {
	var in transferRequest
	if !decode(w, r, &in) {
		return
	}
	in.Currency = strings.ToUpper(in.Currency)
	if !validUUID(in.FromAccountID) || !validUUID(in.ToAccountID) || in.FromAccountID == in.ToAccountID || in.AmountMinor <= 0 || in.Currency != "EUR" || len(in.Note) > 140 || len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 100 {
		writeError(w, 400, "validation_error", "Provide distinct account IDs, positive amount_minor, EUR currency, note <= 140 chars and idempotency_key (8-100 chars)")
		return
	}
	ctx := r.Context()
	tx, err := s.db.Begin(ctx)
	if err != nil {
		writeError(w, 500, "internal_error", "Could not initiate transfer")
		return
	}
	defer tx.Rollback(ctx)
	// Lock accounts in a deterministic order to reduce deadlock risk.
	ids := []string{in.FromAccountID, in.ToAccountID}
	if ids[0] > ids[1] {
		ids[0], ids[1] = ids[1], ids[0]
	}
	rows, err := tx.Query(ctx, `SELECT id,user_id,currency,balance_minor,status FROM accounts WHERE id IN ($1,$2) ORDER BY id FOR UPDATE`, ids[0], ids[1])
	if err != nil {
		writeError(w, 500, "internal_error", "Could not lock accounts")
		return
	}
	type acct struct {
		id, user, currency, status string
		balance                    int64
	}
	found := map[string]acct{}
	for rows.Next() {
		var a acct
		if err := rows.Scan(&a.id, &a.user, &a.currency, &a.balance, &a.status); err != nil {
			rows.Close()
			s.logger.Error("scan transfer account failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not read accounts")
			return
		}
		found[a.id] = a
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		s.logger.Error("iterate transfer accounts failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not read accounts")
		return
	}
	rows.Close()
	from, ok1 := found[in.FromAccountID]
	to, ok2 := found[in.ToAccountID]
	if !ok1 || !ok2 || from.user != uid(r) {
		writeError(w, http.StatusNotFound, "account_not_found", "Source or destination account is unavailable")
		return
	}
	if from.currency != in.Currency || to.currency != in.Currency {
		writeError(w, 400, "currency_mismatch", "Both accounts must use the transfer currency")
		return
	}
	fingerprint := struct {
		FromAccountID string `json:"from_account_id"`
		ToAccountID   string `json:"to_account_id"`
		AmountMinor   int64  `json:"amount_minor"`
		Currency      string `json:"currency"`
		Note          string `json:"note"`
	}{in.FromAccountID, in.ToAccountID, in.AmountMinor, in.Currency, in.Note}
	fingerprintJSON, _ := json.Marshal(fingerprint)
	fingerprintSum := sha256.Sum256(fingerprintJSON)
	requestHash := hex.EncodeToString(fingerprintSum[:])
	var transferID string
	idempotencyInsert, err := tx.Exec(ctx, `INSERT INTO transfer_idempotency(user_id,idempotency_key,request_hash) VALUES($1,$2,$3) ON CONFLICT(user_id,idempotency_key) DO NOTHING`, uid(r), in.IdempotencyKey, requestHash)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not record transfer request")
		return
	}
	if idempotencyInsert.RowsAffected() == 0 {
		var existingHash string
		var existingID string
		err := tx.QueryRow(ctx, `SELECT request_hash,transfer_id::text FROM transfer_idempotency WHERE user_id=$1 AND idempotency_key=$2`, uid(r), in.IdempotencyKey).Scan(&existingHash, &existingID)
		if err != nil || existingHash != requestHash || existingID == "" {
			writeError(w, http.StatusConflict, "idempotency_conflict", "This idempotency key was already used with a different request")
			return
		}
		var ref, status, currency, note string
		var amount int64
		var created time.Time
		if err := tx.QueryRow(ctx, `SELECT reference,status,currency,note,amount_minor,created_at FROM transfers WHERE id=$1`, existingID).Scan(&ref, &status, &currency, &note, &amount, &created); err != nil {
			writeError(w, http.StatusConflict, "idempotency_conflict", "Original transfer is unavailable")
			return
		}
		if err := tx.Commit(ctx); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not confirm transfer result")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"id": existingID, "reference": ref, "status": status, "amount_minor": amount, "currency": strings.TrimSpace(currency), "note": note, "created_at": created, "replayed": true})
		return
	}
	if from.status != "active" || to.status != "active" {
		writeError(w, http.StatusConflict, "account_unavailable", "Source or destination account is not active")
		return
	}
	if from.balance < in.AmountMinor {
		writeError(w, http.StatusConflict, "insufficient_funds", "Insufficient funds")
		return
	}
	if to.balance > math.MaxInt64-in.AmountMinor {
		writeError(w, http.StatusConflict, "balance_limit_exceeded", "Destination balance limit exceeded")
		return
	}
	refBytes := make([]byte, 16)
	if _, err := rand.Read(refBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not create transfer reference")
		return
	}
	ref := "TRX-" + strings.ToUpper(hex.EncodeToString(refBytes))
	var createdAt time.Time
	err = tx.QueryRow(ctx, `INSERT INTO transfers(reference,from_account_id,to_account_id,amount_minor,currency,note,status) VALUES($1,$2,$3,$4,$5,$6,'completed') RETURNING id,created_at`, ref, in.FromAccountID, in.ToAccountID, in.AmountMinor, in.Currency, in.Note).Scan(&transferID, &createdAt)
	if err != nil {
		s.logger.Error("insert transfer failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not create transfer")
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE transfer_idempotency SET transfer_id=$3 WHERE user_id=$1 AND idempotency_key=$2`, uid(r), in.IdempotencyKey, transferID); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not record transfer result")
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE accounts SET balance_minor=balance_minor-$1 WHERE id=$2`, in.AmountMinor, in.FromAccountID); err != nil {
		writeError(w, 500, "internal_error", "Could not debit account")
		return
	}
	if _, err = tx.Exec(ctx, `UPDATE accounts SET balance_minor=balance_minor+$1 WHERE id=$2`, in.AmountMinor, in.ToAccountID); err != nil {
		writeError(w, 500, "internal_error", "Could not credit account")
		return
	}
	if _, err = tx.Exec(ctx, `INSERT INTO ledger_entries(transfer_id,account_id,amount_minor,currency) VALUES($1,$2,$3,$4),($1,$5,$6,$4)`, transferID, in.FromAccountID, -in.AmountMinor, in.Currency, in.ToAccountID, in.AmountMinor); err != nil {
		writeError(w, 500, "internal_error", "Could not record ledger entries")
		return
	}
	if err = tx.Commit(ctx); err != nil {
		writeError(w, 500, "internal_error", "Could not commit transfer")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": transferID, "reference": ref, "status": "completed", "amount_minor": in.AmountMinor, "currency": in.Currency, "note": in.Note, "created_at": createdAt, "message": "Internal prototype transfer completed; no external payment network was contacted."})
}
func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(r.Context(), `SELECT id,device_name,COALESCE(host(ip_address),''),user_agent,expires_at,created_at FROM sessions WHERE user_id=$1 AND revoked_at IS NULL AND expires_at>now() ORDER BY created_at DESC`, uid(r))
	if err != nil {
		writeError(w, 500, "internal_error", "Could not load sessions")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id, device, ip, ua string
		var exp, created time.Time
		if err := rows.Scan(&id, &device, &ip, &ua, &exp, &created); err != nil {
			s.logger.Error("scan session failed", "error", err)
			writeError(w, http.StatusInternalServerError, "internal_error", "Could not load sessions")
			return
		}
		items = append(items, map[string]any{"id": id, "device_name": device, "ip_address": ip, "user_agent": ua, "expires_at": exp, "created_at": created})
	}
	if err := rows.Err(); err != nil {
		s.logger.Error("iterate sessions failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Could not load sessions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
	if !validUUID(r.PathValue("id")) {
		writeError(w, http.StatusNotFound, "not_found", "Session not found")
		return
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE sessions SET revoked_at=now() WHERE id=$1 AND user_id=$2 AND revoked_at IS NULL`, r.PathValue("id"), uid(r))
	if err != nil {
		writeError(w, 500, "internal_error", "Could not revoke session")
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, 404, "not_found", "Session not found")
		return
	}
	writeJSON(w, 200, map[string]string{"status": "revoked"})
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if contentType := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])); contentType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type must be application/json")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body exceeds 1 MiB")
		} else {
			writeError(w, http.StatusBadRequest, "invalid_json", "Request body is invalid or contains unknown fields")
		}
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must contain exactly one JSON value")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}
func recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					log.Error("panic recovered", "panic", v)
					writeError(w, 500, "internal_error", "Unexpected server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := make([]byte, 16)
		if _, err := rand.Read(id); err != nil {
			id = []byte(fmt.Sprintf("%d", time.Now().UnixNano()))
		}
		requestID := hex.EncodeToString(id)
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(&requestWriterWithID{ResponseWriter: w, requestID: requestID}, r)
	})
}

type requestWriterWithID struct {
	http.ResponseWriter
	requestID string
}
func (w *requestWriterWithID) WriteHeader(status int) {
	w.Header().Set("X-Request-ID", w.requestID)
	w.ResponseWriter.WriteHeader(status)
}
func (w *requestWriterWithID) Write(p []byte) (int, error) {
	w.Header().Set("X-Request-ID", w.requestID)
	return w.ResponseWriter.Write(p)
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodOptions {
			w.Header().Set("Allow", "GET, POST, PATCH, DELETE")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for i, ch := range value {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')) {
			return false
		}
	}
	return true
}

func asciiLetters(value string) bool {
	if value == "" { return false }
	for _, ch := range value {
		if (ch < 'A' || ch > 'Z') && (ch < 'a' || ch > 'z') { return false }
	}
	return true
}
