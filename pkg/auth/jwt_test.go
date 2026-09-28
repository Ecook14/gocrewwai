package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func tamperPayload(t *testing.T, tok string) string {
	t.Helper()
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("bad token shape")
	}
	// Flip the last char of the payload segment (keeps base64 valid).
	last := parts[1][len(parts[1])-1]
	rep := byte('A')
	if last == 'A' {
		rep = 'B'
	}
	parts[1] = parts[1][:len(parts[1])-1] + string(rep)
	return strings.Join(parts, ".")
}

func mint(t *testing.T, secret string, header, payload map[string]any) string {
	t.Helper()
	hb, _ := json.Marshal(header)
	pb, _ := json.Marshal(payload)
	h, p := base64.RawURLEncoding.EncodeToString(hb), base64.RawURLEncoding.EncodeToString(pb)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(h + "." + p))
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func TestValidate_RoundTrip(t *testing.T) {
	v := NewValidator("test-secret-12345")
	tok := mint(t, "test-secret-12345",
		map[string]any{"alg": "HS256", "typ": "JWT"},
		map[string]any{"sub": "alice", "exp": time.Now().Add(time.Hour).Unix()})
	c, err := v.Validate(tok)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if c.Subject != "alice" {
		t.Errorf("subject = %q", c.Subject)
	}
}

func TestValidate_Rejects(t *testing.T) {
	v := NewValidator("test-secret-12345")
	cases := map[string]string{
		"empty":      "",
		"structure":  "a.b",
		"none-alg":   mint(t, "test-secret-12345", map[string]any{"alg": "none"}, map[string]any{"sub": "x"}),
		"wrong-key":  mint(t, "other-secret", map[string]any{"alg": "HS256"}, map[string]any{"sub": "x"}),
		"tampered":   tamperPayload(t, mint(t, "test-secret-12345", map[string]any{"alg": "HS256"}, map[string]any{"sub": "x"})),
		"expired":    mint(t, "test-secret-12345", map[string]any{"alg": "HS256"}, map[string]any{"sub": "x", "exp": time.Now().Add(-time.Hour).Unix()}),
		"no-subject": mint(t, "test-secret-12345", map[string]any{"alg": "HS256"}, map[string]any{"exp": time.Now().Add(time.Hour).Unix()}),
		"no-expiry":  mint(t, "test-secret-12345", map[string]any{"alg": "HS256"}, map[string]any{"sub": "x"}),
	}
	for name, tok := range cases {
		if _, err := v.Validate(tok); err == nil {
			t.Errorf("%s: expected rejection", name)
		}
	}
	if _, err := NewValidator("").Validate("a.b.c"); err == nil {
		t.Error("unconfigured validator must fail closed")
	}
	t.Setenv("JWT_SECRET", "env-secret")
	if ValidatorFromEnv() == nil {
		t.Error("expected validator from JWT_SECRET")
	}
	t.Setenv("JWT_SECRET", "")
	if ValidatorFromEnv() != nil {
		t.Error("expected nil validator without JWT_SECRET")
	}
}
