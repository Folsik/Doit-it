package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/your-org/doit-backend/internal/auth"
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

const userIDKey contextKey = "userID"

func (s *Server) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		if h == "" {
			writeError(w, 401, "unauthorized", "Bearer token required")
			return
		}
		c, err := s.tokens.Verify(h)
		if err != nil {
			writeError(w, 401, "unauthorized", "Invalid or expired access token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDKey, c.Subject)))
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
	if !strings.Contains(in.Email, "@") || len(in.Password) < 12 || len(in.FullName) > 120 {
		writeError(w, 400, "validation_error", "Valid email, full_name and password of at least 12 characters are required")
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
	err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,country_code) VALUES($1,$2,$3,NULLIF($4,'')) RETURNING id`, in.Email, string(hash), strings.TrimSpace(in.FullName), strings.ToUpper(in.CountryCode)).Scan(&id)
	if err != nil {
		writeError(w, 409, "email_exists", "Email is already registered")
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
	if err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		writeError(w, 401, "invalid_credentials", "Email or password is incorrect")
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
	writeJSON(w, 200, map[string]string{"status": "ok", "message": "Access token expires automatically; server-side revocation is not implemented in this prototype"})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	var u userDTO
	err := s.db.QueryRow(r.Context(), `SELECT id,email,full_name,COALESCE(country_code,''),preferred_currency,language_code,two_factor_enabled,created_at FROM users WHERE id=$1`, uid(r)).Scan(&u.ID, &u.Email, &u.FullName, &u.CountryCode, &u.PreferredCurrency, &u.LanguageCode, &u.TwoFactorEnabled, &u.CreatedAt)
	if err != nil {
		writeError(w, 404, "not_found", "User not found")
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
	if len(in.FullName) > 120 || (in.PreferredCurrency != "" && len(in.PreferredCurrency) != 3) || len(in.LanguageCode) > 10 || len(in.CountryCode) > 2 {
		writeError(w, 400, "validation_error", "Invalid profile fields")
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
		if rows.Scan(&id, &name, &currency, &balance, &status, &created) == nil {
			out = append(out, map[string]any{"id": id, "name": name, "currency": strings.TrimSpace(currency), "balance_minor": balance, "balance": float64(balance) / 100, "status": status, "created_at": created})
		}
	}
	writeJSON(w, 200, map[string]any{"items": out})
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
		if rows.Scan(&id, &name, &currency, &b, &status) == nil {
			accounts = append(accounts, map[string]any{"id": id, "name": name, "currency": strings.TrimSpace(currency), "balance_minor": b, "status": status})
			if currency == "EUR" {
				total += b
			}
		}
	}
	writeJSON(w, 200, map[string]any{"total_balance_minor": total, "currency": "EUR", "accounts": accounts, "spending_this_month_minor": 0, "recent_transactions": []any{}, "note": "Dashboard spending analytics will populate as transaction categories are added."})
}
func (s *Server) transactions(w http.ResponseWriter, r *http.Request) {
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
		if rows.Scan(&id, &ref, &amount, &currency, &note, &status, &created, &fromUser, &toUser, &fromName, &toName) == nil {
			signed := amount
			title := toName
			if fromUser == uid(r) {
				signed = -amount
				title = fromName
			}
			items = append(items, map[string]any{"id": id, "reference": ref, "amount_minor": signed, "currency": strings.TrimSpace(currency), "amount": float64(signed) / 100, "title": title, "note": note, "status": status, "created_at": created, "type": map[bool]string{true: "outgoing", false: "incoming"}[fromUser == uid(r)]})
		}
	}
	writeJSON(w, 200, map[string]any{"items": items, "limit": limit, "offset": offset})
}
func (s *Server) transactionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var ref, currency, note, status, fromName, toName string
	var amount int64
	var created time.Time
	var fromUser, toUser string
	err := s.db.QueryRow(r.Context(), `SELECT t.reference,t.amount_minor,t.currency,t.note,t.status,t.created_at,a1.user_id,a2.user_id,a1.name,a2.name FROM transfers t JOIN accounts a1 ON a1.id=t.from_account_id JOIN accounts a2 ON a2.id=t.to_account_id WHERE t.id=$1 AND (a1.user_id=$2 OR a2.user_id=$2)`, id, uid(r)).Scan(&ref, &amount, &currency, &note, &status, &created, &fromUser, &toUser, &fromName, &toName)
	if err != nil {
		writeError(w, 404, "not_found", "Transaction not found")
		return
	}
	signed := amount
	if fromUser == uid(r) {
		signed = -amount
	}
	writeJSON(w, 200, map[string]any{"id": id, "reference": ref, "amount_minor": signed, "currency": strings.TrimSpace(currency), "amount": float64(signed) / 100, "note": note, "status": status, "created_at": created, "counterparty_account_name": map[bool]string{true: toName, false: fromName}[fromUser == uid(r)]})
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
	if in.FromAccountID == "" || in.ToAccountID == "" || in.FromAccountID == in.ToAccountID || in.AmountMinor <= 0 || in.Currency != "EUR" || len(in.Note) > 140 || len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 100 {
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
		if rows.Scan(&a.id, &a.user, &a.currency, &a.balance, &a.status) == nil {
			found[a.id] = a
		}
	}
	rows.Close()
	from, ok1 := found[in.FromAccountID]
	to, ok2 := found[in.ToAccountID]
	if !ok1 || !ok2 || from.user != uid(r) || to.status != "active" || from.status != "active" {
		writeError(w, 404, "account_not_found", "Source or destination account is unavailable")
		return
	}
	if from.currency != in.Currency || to.currency != in.Currency {
		writeError(w, 400, "currency_mismatch", "Both accounts must use the transfer currency")
		return
	}
	if from.balance < in.AmountMinor {
		writeError(w, 409, "insufficient_funds", "Insufficient funds")
		return
	}
	// Idempotency is recorded as a unique reference. In a real API use a dedicated idempotency table scoped to the user and endpoint.
	ref := in.IdempotencyKey
	var transferID string
	err = tx.QueryRow(ctx, `INSERT INTO transfers(reference,from_account_id,to_account_id,amount_minor,currency,note,status) VALUES($1,$2,$3,$4,$5,$6,'completed') RETURNING id`, ref, in.FromAccountID, in.ToAccountID, in.AmountMinor, in.Currency, in.Note).Scan(&transferID)
	if err != nil {
		writeError(w, 409, "duplicate_or_invalid_transfer", "This idempotency key may already have been used")
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
	writeJSON(w, 201, map[string]any{"id": transferID, "reference": ref, "status": "completed", "amount_minor": in.AmountMinor, "currency": in.Currency, "note": in.Note, "created_at": time.Now().UTC(), "message": "Internal prototype transfer completed; no external payment network was contacted."})
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
		if rows.Scan(&id, &device, &ip, &ua, &exp, &created) == nil {
			items = append(items, map[string]any{"id": id, "device_name": device, "ip_address": ip, "user_agent": ua, "expires_at": exp, "created_at": created})
		}
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (s *Server) revokeSession(w http.ResponseWriter, r *http.Request) {
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
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeError(w, 400, "invalid_json", "Request body is invalid or contains unknown fields")
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { next.ServeHTTP(w, r) })
}
func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var _ = errors.Is
var _ = pgx.ErrNoRows
