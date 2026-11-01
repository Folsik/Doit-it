package veriff

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyWebhookSignature(t *testing.T) {
	c := &Client{sharedSecret: []byte("test-secret")}
	body := []byte(`{"verification":{"id":"session","status":"approved"}}`)
	mac := hmac.New(sha256.New, c.sharedSecret)
	_, _ = mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))
	if !c.VerifyWebhookSignature(body, signature) {
		t.Fatal("expected valid signature")
	}
	if c.VerifyWebhookSignature(append(body, ' '), signature) {
		t.Fatal("signature must bind exact raw body")
	}
	if c.VerifyWebhookSignature(body, "not-hex") {
		t.Fatal("malformed signature must fail")
	}
}

func TestParseDecisionWebhook(t *testing.T) {
	body := []byte(`{"verification":{"id":"session-123","status":"approved","vendorData":"user-123"}}`)
	event, err := ParseDecisionWebhook(body)
	if err != nil {
		t.Fatal(err)
	}
	if event.Verification.ID != "session-123" || event.Verification.Status != "approved" {
		t.Fatalf("unexpected event: %+v", event)
	}
}
