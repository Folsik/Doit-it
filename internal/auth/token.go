package auth

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const maxTokenLength = 8192

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

// NewManager generates an ephemeral key and is intended for local development only.
// Production deployments must use NewManagerFromPrivateKey with a persistent secret.
func NewManager(issuer string, ttl time.Duration) (*Manager, error) {
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return newManager(issuer, ttl, private)
}

// NewManagerFromPrivateKey loads a base64url-encoded 64-byte Ed25519 private key.
// Keep the key in a secret manager/environment variable; never commit it to source control.
func NewManagerFromPrivateKey(issuer string, ttl time.Duration, encoded string) (*Manager, error) {
	key, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, fmt.Errorf("decode JWT_PRIVATE_KEY: %w", err)
	}
	if len(key) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("JWT_PRIVATE_KEY must decode to %d bytes", ed25519.PrivateKeySize)
	}
	derived := ed25519.NewKeyFromSeed(key[:ed25519.SeedSize])
	if !bytes.Equal(derived, key) {
		return nil, errors.New("JWT_PRIVATE_KEY is not a valid Ed25519 private key")
	}
	return newManager(issuer, ttl, ed25519.PrivateKey(key))
}

func newManager(issuer string, ttl time.Duration, private ed25519.PrivateKey) (*Manager, error) {
	if strings.TrimSpace(issuer) == "" {
		return nil, errors.New("JWT issuer must not be empty")
	}
	if ttl < time.Minute || ttl > time.Hour {
		return nil, errors.New("JWT TTL must be at least one minute and no more than one hour")
	}
	if len(private) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid Ed25519 private key")
	}
	privateCopy := append(ed25519.PrivateKey(nil), private...)
	public := append(ed25519.PublicKey(nil), privateCopy.Public().(ed25519.PublicKey)...)
	return &Manager{private: privateCopy, public: public, issuer: issuer, ttl: ttl}, nil
}

func (m *Manager) Issue(subject string) (string, time.Time, error) {
	if strings.TrimSpace(subject) == "" {
		return "", time.Time{}, errors.New("token subject must not be empty")
	}
	now := time.Now().UTC()
	exp := now.Add(m.ttl)
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return "", time.Time{}, err
	}
	claims := Claims{
		Subject: subject, Issuer: m.issuer, IssuedAt: now.Unix(),
		ExpiresAt: exp.Unix(), TokenID: base64.RawURLEncoding.EncodeToString(idBytes),
	}
	header, err := json.Marshal(map[string]string{"alg": "EdDSA", "typ": "JWT"})
	if err != nil {
		return "", time.Time{}, err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", time.Time{}, err
	}
	input := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	sig := ed25519.Sign(m.private, []byte(input))
	return input + "." + base64.RawURLEncoding.EncodeToString(sig), exp, nil
}

func (m *Manager) Verify(token string) (Claims, error) {
	var claims Claims
	if token == "" || len(token) > maxTokenLength {
		return claims, errors.New("invalid token length")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return claims, errors.New("invalid token format")
	}
	headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return claims, errors.New("invalid token header")
	}
	var header struct {
		Algorithm string `json:"alg"`
		Type      string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil || header.Algorithm != "EdDSA" || header.Type != "JWT" {
		return claims, errors.New("unsupported token header")
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(signature) != ed25519.SignatureSize {
		return claims, errors.New("invalid token signature")
	}
	input := parts[0] + "." + parts[1]
	if !ed25519.Verify(m.public, []byte(input), signature) {
		return claims, errors.New("invalid token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return claims, errors.New("invalid token payload")
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return claims, errors.New("invalid token claims")
	}
	now := time.Now().Unix()
	if claims.Issuer != m.issuer || claims.Subject == "" || claims.TokenID == "" ||
		claims.IssuedAt <= 0 || claims.IssuedAt > now+30 ||
		claims.ExpiresAt <= now || claims.ExpiresAt <= claims.IssuedAt ||
		time.Duration(claims.ExpiresAt-claims.IssuedAt)*time.Second > m.ttl+time.Second {
		return Claims{}, errors.New("expired or invalid token claims")
	}
	return claims, nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
