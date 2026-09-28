// Package webhook delivers signed outbound notifications on crew lifecycle
// events (kickoff completion/failure). Receivers verify authenticity with
// HMAC-SHA256 over the raw body:
//
//	X-Gocrew-Signature: sha256=<hex(hmac(secret, body))>
//	X-Gocrew-Timestamp: <unix seconds> (reject if skewed > 5min)
//	X-Gocrew-Event:     kickoff.completed | kickoff.failed
package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Ecook14/gocrewwai/pkg/utils"
)

// DefaultTimeout bounds delivery; MaxBodyBytes bounds response reads.
// MaxRedirects caps redirect hops (each re-validated against the SSRF gate).
const (
	DefaultTimeout = 10 * time.Second
	DefaultDial    = 5 * time.Second
	MaxBodyBytes   = 1 << 20 // 1MB
	MaxRedirects   = 3
	diagLimit      = 1024
)

// Notifier posts signed JSON payloads to a single endpoint.
type Notifier struct {
	URL    string
	Secret string
	client *http.Client
}

// transport builds the shared HTTP transport with timeouts and a redirect
// hook that re-validates each hop against the SSRF gate.
func transport() *http.Client {
	return &http.Client{
		Timeout: DefaultTimeout,
		Transport: &http.Transport{
			DialContext:         (&net.Dialer{Timeout: DefaultDial}).DialContext,
			TLSHandshakeTimeout: DefaultDial,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= MaxRedirects {
				return fmt.Errorf("webhook: too many redirects (max %d)", MaxRedirects)
			}
			if _, err := utils.ValidateURL(req.URL.String()); err != nil {
				return fmt.Errorf("webhook: blocked redirect target: %w", err)
			}
			return nil
		},
	}
}

// NewNotifier validates the URL (SSRF-gated) and returns a notifier.
// An empty secret disables signing (unsigned delivery, log-visible).
func NewNotifier(url, secret string) (*Notifier, error) {
	if _, err := utils.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("webhook: blocked URL: %w", err)
	}
	return &Notifier{URL: url, Secret: secret, client: transport()}, nil
}

// Sign returns the hex HMAC-SHA256 of body under secret.
func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

// Payload is the crew lifecycle event envelope.
type Payload struct {
	Event     string      `json:"event"`
	SessionID string      `json:"session_id,omitempty"`
	Status    string      `json:"status"`
	Result    interface{} `json:"result,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
}

// Notify POSTs payload with signature headers. A 2xx response is success;
// other statuses are an error (body truncated for diagnostics).
func (n *Notifier) Notify(ctx context.Context, p Payload) error {
	if p.Timestamp == 0 {
		p.Timestamp = time.Now().Unix()
	}
	body, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("webhook: encode: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("webhook: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Gocrew-Timestamp", strconv.FormatInt(p.Timestamp, 10))
	req.Header.Set("X-Gocrew-Event", p.Event)
	if n.Secret != "" {
		req.Header.Set("X-Gocrew-Signature", "sha256="+Sign(n.Secret, body))
	}
	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook: delivery: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		diag, _ := io.ReadAll(io.LimitReader(resp.Body, diagLimit))
		return fmt.Errorf("webhook: endpoint returned %d: %s", resp.StatusCode, string(diag))
	}
	return nil
}

// VerifySignature checks a received signature header against the body.
// skew bounds the timestamp header (0 disables time check — tests only).
func VerifySignature(secret string, body []byte, signature, timestamp string, skew time.Duration) error {
	want := "sha256=" + Sign(secret, body)
	if len(signature) != len(want) || subtle.ConstantTimeCompare([]byte(signature), []byte(want)) != 1 {
		return fmt.Errorf("webhook: bad signature")
	}
	if skew > 0 && timestamp != "" {
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return fmt.Errorf("webhook: bad timestamp")
		}
		if age := time.Since(time.Unix(ts, 0)); age < 0 || age > skew {
			return fmt.Errorf("webhook: stale timestamp")
		}
	}
	return nil
}
