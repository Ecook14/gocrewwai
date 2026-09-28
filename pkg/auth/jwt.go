// Package auth provides optional JWT bearer validation (stdlib-only, no new
// dependencies). It is an alternative to static API tokens: when JWT_SECRET
// is set, pkg/server and pkg/api accept `Authorization: Bearer <jwt>` signed
// with HS256. API keys keep working alongside.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// MaxTokenBytes bounds JWT input (DoS guard).
const MaxTokenBytes = 8192

// clockSkew allows for reasonable host clock drift when checking exp/nbf.
const clockSkew = 30 * time.Second

// Claims is the verified JWT payload subset we enforce.
type Claims struct {
	Subject   string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// Validator checks HS256 JWTs against a shared secret.
type Validator struct {
	secret []byte
}

// NewValidator creates a validator; empty secret disables validation
// (Validate always errors — fail closed, never open).
func NewValidator(secret string) *Validator {
	return &Validator{secret: []byte(secret)}
}

// ValidatorFromEnv builds a validator from JWT_SECRET (nil when unset).
func ValidatorFromEnv() *Validator {
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return NewValidator(s)
	}
	return nil
}

// b64 returns the RawURL-decoded bytes or a generic error.
func b64(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// Validate verifies signature (constant-time), structure, and time bounds.
// Only HS256 is accepted (`none` and other algorithms are rejected).
// Returns the subject claim. Unknown claims are ignored.
func (v *Validator) Validate(token string) (*Claims, error) {
	if v == nil || len(v.secret) == 0 {
		return nil, fmt.Errorf("auth: JWT validation not configured")
	}
	if len(token) == 0 || len(token) > MaxTokenBytes {
		return nil, fmt.Errorf("auth: malformed token")
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("auth: malformed token")
	}

	var hdr struct {
		Alg string `json:"alg"`
	}
	headerJSON, err := b64(parts[0])
	if err != nil || json.Unmarshal(headerJSON, &hdr) != nil || hdr.Alg != "HS256" {
		return nil, fmt.Errorf("auth: unsupported algorithm (HS256 only)")
	}

	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	sig, err := b64(parts[2])
	if err != nil {
		return nil, fmt.Errorf("auth: malformed signature")
	}
	if subtle.ConstantTimeCompare(sig, mac.Sum(nil)) != 1 {
		return nil, fmt.Errorf("auth: invalid signature")
	}

	var raw struct {
		Sub string `json:"sub"`
		Exp int64  `json:"exp"`
		Nbf int64  `json:"nbf"`
		Iat int64  `json:"iat"`
	}
	payload, err := b64(parts[1])
	if err != nil {
		return nil, fmt.Errorf("auth: malformed payload")
	}
	if err := json.Unmarshal(payload, &raw); err != nil || strings.TrimSpace(raw.Sub) == "" {
		return nil, fmt.Errorf("auth: malformed payload")
	}
	// Expiry is mandatory: a signed token without exp validates forever,
	// giving a stolen token indefinite access.
	if raw.Exp == 0 {
		return nil, fmt.Errorf("auth: missing expiry")
	}
	now := time.Now()
	if now.After(time.Unix(raw.Exp, 0).Add(clockSkew)) {
		return nil, fmt.Errorf("auth: token expired")
	}
	if raw.Nbf != 0 && now.Add(clockSkew).Before(time.Unix(raw.Nbf, 0)) {
		return nil, fmt.Errorf("auth: token not yet valid")
	}
	claims := &Claims{Subject: raw.Sub, ExpiresAt: time.Unix(raw.Exp, 0)}
	if raw.Iat != 0 {
		claims.IssuedAt = time.Unix(raw.Iat, 0)
	}
	return claims, nil
}

// SubjectFingerprint derives a stable, non-reversible owner ID from a subject,
// mirroring the tokenFingerprint pattern used for API keys.
func SubjectFingerprint(subject string) string {
	sum := sha256.Sum256([]byte("jwt-sub:" + subject))
	return fmt.Sprintf("%x", sum)[:16]
}
