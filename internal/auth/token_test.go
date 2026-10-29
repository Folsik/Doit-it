package auth

import (
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
