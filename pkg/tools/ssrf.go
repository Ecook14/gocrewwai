package tools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ssrfBlockedIP reports whether an IP must never be dialed by agent fetch tools.
func ssrfBlockedIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}
	return false
}

// ssrfDialContext resolves, validates, and dials an approved literal address.
// It closes the DNS-change TOCTOU between preflight URL validation and connect:
// the address policy is enforced on the addresses actually dialed, while the
// original hostname is preserved for HTTP Host and TLS ServerName by the Transport.
func ssrfDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid dial address: %w", err)
	}
	if ip := net.ParseIP(host); ip != nil {
		if ssrfBlockedIP(ip) {
			return nil, fmt.Errorf("dial to blocked IP %s refused", ip.String())
		}
		return (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, addr)
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %w", host, err)
	}
	var lastErr error
	for _, ip := range ips {
		if ssrfBlockedIP(ip) {
			lastErr = fmt.Errorf("dial to blocked IP %s refused", ip.String())
			continue
		}
		conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = errors.New("no allowed addresses resolved")
	}
	return nil, lastErr
}

// ssrfCheckRedirect re-applies the destination policy on every redirect hop.
// Initial-URL approval must never authorize a later server-chosen destination.
func ssrfCheckRedirect(validate func(string) error) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if err := validate(req.URL.String()); err != nil {
			return err
		}
		return nil
	}
}

// newSSRFProtectedClient builds an HTTP client with redirect re-validation
// and connect-time address enforcement.
func newSSRFProtectedClient(validate func(string) error, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:           ssrfDialContext,
			MaxIdleConns:          5,
			MaxIdleConnsPerHost:   2,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 15 * time.Second,
		},
		CheckRedirect: ssrfCheckRedirect(validate),
	}
}
