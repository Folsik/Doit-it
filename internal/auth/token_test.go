package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"
)

func TestIssueAndVerify(t *testing.T) {
	manager, err := NewManager("doit-test", 1*time.Minute)
	if err != nil { t.Fatal(err) }
	token, _, err := manager.Issue("user-123")
	if err != nil { t.Fatal(err) }
	claims, err := manager.Verify(token)
	if err != nil { t.Fatalf("Verify() error = %v", err) }
	if claims.Subject != "user-123" || claims.Issuer != "doit-test" || claims.TokenID == "" {
		t.Fatalf("unexpected claims: %+v", claims)
	}
}

func TestVerifyRejectsMalformedTokens(t *testing.T) {
	manager, err := NewManager("doit-test", 1*time.Minute)
	if err != nil { t.Fatal(err) }
	for _, token := range []string{"", "a.b.c", "eyJhbGciOiJub25lIn0.eyJzdWIiOiJ1c2VyIn0."} {
		if _, err := manager.Verify(token); err == nil {
			t.Errorf("Verify(%q) unexpectedly succeeded", token)
		}
	}
}

func TestPrivateKeyMustHaveCorrectLength(t *testing.T) {
	if _, err := NewManagerFromPrivateKey("doit-test", time.Minute, "aW52YWxpZA"); err == nil {
		t.Fatal("expected invalid private key to be rejected")
	}
}

func TestPersistentPrivateKeyWorksAcrossManagers(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(privateKey)
	issuer := "doit-test"
	first, err := NewManagerFromPrivateKey(issuer, time.Minute, encoded)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewManagerFromPrivateKey(issuer, time.Minute, encoded)
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := first.Issue("user-123")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := second.Verify(token); err != nil {
		t.Fatalf("token signed by first manager should verify with the same key: %v", err)
	}
}
