package rss

import (
	"net"
	"net/http"
	"testing"
)

func TestValidatePublicURL_AllowsPublicHTTPS(t *testing.T) {
	for _, u := range []string{
		"https://example.com/feed.xml",
		"https://news.bestierealestate.com/article",
		"https://1.1.1.1/x", // public IP literal
	} {
		if err := validatePublicURL(u); err != nil {
			t.Errorf("validatePublicURL(%q) = %v, want nil", u, err)
		}
	}
}

func TestValidatePublicURL_RejectsNonHTTPS(t *testing.T) {
	for _, u := range []string{
		"http://example.com/feed.xml", // plain http
		"ftp://example.com/x",
		"file:///etc/passwd",
		"gopher://example.com",
	} {
		if err := validatePublicURL(u); err == nil {
			t.Errorf("validatePublicURL(%q) = nil, want error (non-https scheme)", u)
		}
	}
}

func TestValidatePublicURL_RejectsInternalHosts(t *testing.T) {
	for _, u := range []string{
		"https://localhost/x",
		"https://127.0.0.1/x",
		"https://10.0.0.5/x",
		"https://172.16.0.1/x",
		"https://192.168.1.1/x",
		"https://169.254.169.254/latest/meta-data", // cloud metadata
		"https://[::1]/x",
		"https://[fc00::1]/x",
	} {
		if err := validatePublicURL(u); err == nil {
			t.Errorf("validatePublicURL(%q) = nil, want error (internal host)", u)
		}
	}
}

func TestIsBlockedIP(t *testing.T) {
	blocked := []string{
		"127.0.0.1", "10.1.2.3", "172.16.5.5", "172.31.255.255",
		"192.168.0.1", "169.254.0.1", "::1", "fc00::1", "fe80::1",
		"0.0.0.0",
	}
	for _, s := range blocked {
		ip := net.ParseIP(s)
		if ip == nil {
			t.Fatalf("bad test IP %q", s)
		}
		if !isBlockedIP(ip) {
			t.Errorf("isBlockedIP(%s) = false, want true", s)
		}
	}
	allowed := []string{"1.1.1.1", "8.8.8.8", "203.0.113.10", "2606:4700:4700::1111"}
	for _, s := range allowed {
		ip := net.ParseIP(s)
		if !isBlockedIP(ip) {
			continue
		}
		t.Errorf("isBlockedIP(%s) = true, want false (public)", s)
	}
}

func TestGuardedRedirect_BlocksInternalAndNonHTTPS(t *testing.T) {
	client := DefaultHTTPClient()
	if client.CheckRedirect == nil {
		t.Fatal("DefaultHTTPClient must set CheckRedirect (SSRF guard)")
	}

	mkReq := func(rawURL string) *http.Request {
		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			t.Fatalf("build req %q: %v", rawURL, err)
		}
		return req
	}

	// A redirect to an internal host must be blocked.
	if err := client.CheckRedirect(mkReq("https://169.254.169.254/x"), nil); err == nil {
		t.Error("CheckRedirect to link-local metadata IP should error")
	}
	// A redirect to plain http must be blocked (downgrade).
	if err := client.CheckRedirect(mkReq("http://example.com/x"), nil); err == nil {
		t.Error("CheckRedirect to http (downgrade) should error")
	}
	// A redirect to a public https target is allowed.
	if err := client.CheckRedirect(mkReq("https://example.com/elsewhere"), nil); err != nil {
		t.Errorf("CheckRedirect to public https should be nil, got %v", err)
	}
}

// TestSafeControl_RejectsResolvedInternalIP proves the dial-time guard screens
// the RESOLVED peer IP (the DNS-rebinding defense the URL check alone can't give).
func TestSafeControl_RejectsResolvedInternalIP(t *testing.T) {
	blocked := []string{"169.254.169.254:443", "10.0.0.1:443", "127.0.0.1:443", "[::1]:443", "192.168.1.1:80"}
	for _, addr := range blocked {
		if err := safeControl("tcp", addr, nil); err == nil {
			t.Errorf("safeControl(%q) = nil, want blocked", addr)
		}
	}
	if err := safeControl("tcp", "1.1.1.1:443", nil); err != nil {
		t.Errorf("safeControl(1.1.1.1:443) = %v, want allowed (public)", err)
	}
}
