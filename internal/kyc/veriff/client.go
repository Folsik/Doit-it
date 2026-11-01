package veriff

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrInvalidResponse = errors.New("veriff: invalid provider response")

type Config struct {
	BaseURL       string
	APIKey        string
	SharedSecret  string
	CallbackURL   string
	HTTPClient    *http.Client
}

type Client struct {
	baseURL      string
	apiKey       string
	sharedSecret []byte
	callbackURL  string
	httpClient   *http.Client
}

type CreateSessionRequest struct {
	UserID string
}

type Session struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	Token     string `json:"sessionToken,omitempty"`
	Status    string `json:"status,omitempty"`
	VendorData string `json:"vendorData,omitempty"`
}

type createSessionResponse struct {
	Status       string `json:"status"`
	Verification struct {
		ID          string `json:"id"`
		URL         string `json:"url"`
		SessionToken string `json:"sessionToken"`
		Status      string `json:"status"`
	} `json:"verification"`
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.APIKey) == "" || strings.TrimSpace(cfg.SharedSecret) == "" {
		return nil, errors.New("veriff: API key and shared secret are required")
	}
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return nil, errors.New("veriff: base URL is required")
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return nil, errors.New("veriff: base URL must be an absolute HTTPS URL")
	}
	if strings.TrimSpace(cfg.CallbackURL) == "" {
		return nil, errors.New("veriff: callback URL is required")
	}
	cb, err := url.Parse(cfg.CallbackURL)
	if err != nil || cb.Scheme != "https" || cb.Host == "" {
		return nil, errors.New("veriff: callback URL must be an absolute HTTPS URL")
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: base, apiKey: cfg.APIKey, sharedSecret: []byte(cfg.SharedSecret), callbackURL: cfg.CallbackURL, httpClient: hc}, nil
}

func (c *Client) CreateSession(ctx context.Context, req CreateSessionRequest) (Session, error) {
	if strings.TrimSpace(req.UserID) == "" {
		return Session{}, errors.New("veriff: user ID is required")
	}
	payload := struct {
		Verification struct {
			Callback  string `json:"callback"`
			VendorData string `json:"vendorData"`
		} `json:"verification"`
	}{}
	payload.Verification.Callback = c.callbackURL
	payload.Verification.VendorData = req.UserID
	body, err := json.Marshal(payload)
	if err != nil {
		return Session{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/sessions", bytes.NewReader(body))
	if err != nil {
		return Session{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-AUTH-CLIENT", c.apiKey)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return Session{}, fmt.Errorf("veriff: create session request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Session{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Session{}, fmt.Errorf("veriff: create session returned HTTP %d", resp.StatusCode)
	}
	var decoded createSessionResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return Session{}, ErrInvalidResponse
	}
	v := decoded.Verification
	if decoded.Status != "success" || v.ID == "" || v.URL == "" {
		return Session{}, ErrInvalidResponse
	}
	parsed, err := url.Parse(v.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return Session{}, ErrInvalidResponse
	}
	return Session{ID: v.ID, URL: v.URL, Token: v.SessionToken, Status: v.Status, VendorData: req.UserID}, nil
}

// VerifyWebhookSignature validates Veriff's HMAC-SHA256 hex signature over the exact raw body.
func (c *Client) VerifyWebhookSignature(raw []byte, signature string) bool {
	sig, err := hex.DecodeString(strings.TrimSpace(signature))
	if err != nil || len(sig) != sha256.Size {
		return false
	}
	mac := hmac.New(sha256.New, c.sharedSecret)
	_, _ = mac.Write(raw)
	return hmac.Equal(sig, mac.Sum(nil))
}
