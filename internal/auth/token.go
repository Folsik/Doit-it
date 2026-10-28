package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Claims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	TokenID   string `json:"jti"`
}
type Manager struct {
	private ed25519.PrivateKey
	public  ed25519.PublicKey
	issuer  string
	ttl     time.Duration
}

func NewManager(issuer string, ttl time.Duration) (*Manager, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	// Development convenience only: keys rotate on restart. Configure persistent keys before deployment.
	return &Manager{private: priv, public: pub, issuer: issuer, ttl: ttl}, nil
}
func (m *Manager) Issue(subject string) (string, time.Time, error) {
	now := time.Now().UTC()
	exp := now.Add(m.ttl)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", time.Time{}, err
	}
	claims := Claims{Subject: subject, Issuer: m.issuer, IssuedAt: now.Unix(), ExpiresAt: exp.Unix(), TokenID: base64.RawURLEncoding.EncodeToString(idBytes)}
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT"})
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	sig := ed25519.Sign(m.private, []byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(sig), exp, nil
}
func (m *Manager) Verify(token string) (Claims, error) {
	var c Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return c, errors.New("invalid token")
	}
	input := parts[0] + "." + parts[1]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !ed25519.Verify(m.public, []byte(input), sig) {
		return c, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return c, errors.New("invalid token payload")
	}
	if err := json.Unmarshal(payload, &c); err != nil {
		return c, errors.New("invalid token claims")
	}
	if c.Issuer != m.issuer || c.Subject == "" || c.ExpiresAt <= time.Now().Unix() {
		return c, errors.New("expired or invalid token")
	}
	return c, nil
}
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
