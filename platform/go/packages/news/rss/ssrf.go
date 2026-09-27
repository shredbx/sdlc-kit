package rss

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

// SSRF hardening for the outbound feed fetcher (and the P3 og:image fetch).
//
// The aggregator fetches attacker-influenceable URLs: a registered source URL,
// an article `link` (for og:image), and an image src parsed out of feed HTML.
// Without a guard, a malicious feed could point those at an internal address
// (cloud metadata 169.254.169.254, localhost, RFC-1918) and turn the fetcher
// into a Server-Side Request Forgery pivot. Two complementary controls:
//
//  1. validatePublicURL — pre-flight scheme + host/IP check before the request.
//  2. guardedCheckRedirect — re-validates every redirect hop (a 30x to an
//     internal host is the classic bypass of a one-time pre-flight check).
//
// The guard is deliberately conservative: https only, and any literal IP (or a
// hostname that is itself an IP) that lands in a private / loopback / link-local
// / unique-local range is refused. Beyond the URL/redirect checks, a dial-time
// Control hook (safeControl) screens the RESOLVED peer IP of every connection —
// closing DNS-rebinding (a public hostname that resolves to an internal address)
// and the TOCTOU window the URL-level check alone cannot cover.

// errBlockedHost is the shared SSRF rejection.
type errBlockedHost struct {
	host   string
	reason string
}

func (e *errBlockedHost) Error() string {
	return fmt.Sprintf("feed: blocked request to %q: %s", e.host, e.reason)
}

// validatePublicURL rejects a URL whose scheme is not https or whose host is an
// internal / loopback / link-local / private address (or the literal localhost).
// It is the pre-flight check applied before the source fetch and the og:image
// fetch.
func validatePublicURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("feed: parse url %q: %w", raw, err)
	}
	if u.Scheme != "https" {
		return &errBlockedHost{host: raw, reason: "scheme must be https"}
	}
	host := u.Hostname()
	if host == "" {
		return &errBlockedHost{host: raw, reason: "empty host"}
	}
	if isBlockedHostname(host) {
		return &errBlockedHost{host: host, reason: "internal hostname"}
	}
	// When the host is an IP literal, screen it directly.
	if ip := net.ParseIP(host); ip != nil && isBlockedIP(ip) {
		return &errBlockedHost{host: host, reason: "internal IP range"}
	}
	return nil
}

// isBlockedHostname blocks the obvious loopback aliases by name. IP-literal
// hosts are screened by isBlockedIP via validatePublicURL.
func isBlockedHostname(host string) bool {
	switch host {
	case "localhost", "localhost.localdomain", "ip6-localhost", "ip6-loopback":
		return true
	}
	return false
}

// isBlockedIP reports whether ip falls in a range that must never be fetched:
// loopback (127/8, ::1), private (10/8, 172.16/12, 192.168/16, fc00::/7),
// link-local (169.254/16, fe80::/10), the unspecified address, or other
// non-global-unicast ranges (multicast, etc.).
func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsInterfaceLocalMulticast() {
		return true
	}
	return false
}

// guardedCheckRedirect is the *http.Client.CheckRedirect that re-validates every
// redirect target with validatePublicURL — closing the redirect-bypass hole
// (a public URL 30x-ing to an internal host). It also caps the redirect chain.
func guardedCheckRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxRedirects {
		return fmt.Errorf("feed: stopped after %d redirects", maxRedirects)
	}
	return validatePublicURL(req.URL.String())
}

// maxRedirects caps a redirect chain (Go's default is 10; we keep parity).
const maxRedirects = 10

// safeControl is a net.Dialer Control hook that screens the RESOLVED peer
// address of every connection attempt. Because it runs AFTER DNS resolution and
// immediately before connect, it is the only place that defends against
// DNS-rebinding (a public hostname resolving to an internal IP) and the
// resolve-then-connect TOCTOU window. Wired into safeTransport's dialer.
func safeControl(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return &errBlockedHost{host: address, reason: "unresolvable dial address"}
	}
	if isBlockedIP(ip) {
		return &errBlockedHost{host: ip.String(), reason: "resolved to internal IP range"}
	}
	return nil
}

// safeTransport returns an *http.Transport whose dialer enforces safeControl on
// the resolved peer IP. Shared by DefaultHTTPClient so every outbound feed /
// og:image / image fetch is dial-guarded.
func safeTransport() *http.Transport {
	d := &net.Dialer{Timeout: DefaultFetchTimeout, Control: safeControl}
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return d.DialContext(ctx, network, addr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   DefaultFetchTimeout,
		ExpectContinueTimeout: time.Second,
	}
}
