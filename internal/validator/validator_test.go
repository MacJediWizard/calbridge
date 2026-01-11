package validator

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestValidateURL(t *testing.T) {
	v := New()

	testCases := []struct {
		name         string
		url          string
		requireHTTPS bool
		wantErr      error
	}{
		{
			name:         "valid HTTPS URL",
			url:          "https://example.com/path",
			requireHTTPS: true,
			wantErr:      nil,
		},
		{
			name:         "valid HTTP URL when not required",
			url:          "http://example.com/path",
			requireHTTPS: false,
			wantErr:      nil,
		},
		{
			name:         "HTTP URL when HTTPS required",
			url:          "http://example.com/path",
			requireHTTPS: true,
			wantErr:      ErrHTTPSRequired,
		},
		{
			name:         "empty URL",
			url:          "",
			requireHTTPS: false,
			wantErr:      ErrInvalidURL,
		},
		{
			name:         "missing host",
			url:          "https:///path",
			requireHTTPS: true,
			wantErr:      ErrInvalidURL,
		},
		{
			name:         "invalid scheme",
			url:          "ftp://example.com/path",
			requireHTTPS: false,
			wantErr:      ErrInvalidURL,
		},
		{
			name:         "URL with port",
			url:          "https://example.com:8080/path",
			requireHTTPS: true,
			wantErr:      nil,
		},
		{
			name:         "URL with query string",
			url:          "https://example.com/path?foo=bar",
			requireHTTPS: true,
			wantErr:      nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := v.ValidateURL(tc.url, tc.requireHTTPS)

			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			} else {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("expected error %v, got %v", tc.wantErr, err)
				}
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Private IPs
		{"loopback IPv4", "127.0.0.1", true},
		{"loopback IPv6", "::1", true},
		{"private 10.x.x.x", "10.0.0.1", true},
		{"private 172.16.x.x", "172.16.0.1", true},
		{"private 192.168.x.x", "192.168.1.1", true},
		{"link-local", "169.254.1.1", true},
		{"unspecified IPv4", "0.0.0.0", true},
		{"unspecified IPv6", "::", true},

		// Public IPs
		{"public IP 8.8.8.8", "8.8.8.8", false},
		{"public IP 1.1.1.1", "1.1.1.1", false},
		{"public IP 93.184.216.34", "93.184.216.34", false},

		// Edge cases
		{"nil IP", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var ip net.IP
			if tc.ip != "" {
				ip = net.ParseIP(tc.ip)
			}

			result := isPrivateIP(ip)
			if result != tc.expected {
				t.Errorf("isPrivateIP(%s) = %v, expected %v", tc.ip, result, tc.expected)
			}
		})
	}
}

func TestWithAllowPrivateIPs(t *testing.T) {
	t.Run("default does not allow private IPs", func(t *testing.T) {
		v := New()
		if v.allowPrivateIPs {
			t.Error("expected allowPrivateIPs to be false by default")
		}
	})

	t.Run("WithAllowPrivateIPs enables private IPs", func(t *testing.T) {
		v := New(WithAllowPrivateIPs())
		if !v.allowPrivateIPs {
			t.Error("expected allowPrivateIPs to be true")
		}
	})
}

func TestNewValidator(t *testing.T) {
	t.Run("creates validator with default settings", func(t *testing.T) {
		v := New()
		if v == nil {
			t.Fatal("expected non-nil validator")
		}
		if v.client == nil {
			t.Error("expected HTTP client to be created")
		}
	})

	t.Run("creates validator with options", func(t *testing.T) {
		v := New(WithAllowPrivateIPs())
		if v == nil {
			t.Fatal("expected non-nil validator")
		}
		if !v.allowPrivateIPs {
			t.Error("expected option to be applied")
		}
	})
}

func TestErrorConstants(t *testing.T) {
	t.Run("error constants are not nil", func(t *testing.T) {
		errs := []error{
			ErrInvalidURL,
			ErrHTTPSRequired,
			ErrPrivateIP,
			ErrTooManyRedirects,
			ErrConnectionFailed,
			ErrInvalidOIDCIssuer,
			ErrInvalidCalDAV,
		}

		for _, err := range errs {
			if err == nil {
				t.Error("expected error constant to be non-nil")
			}
		}
	})

	t.Run("error messages are descriptive", func(t *testing.T) {
		if ErrInvalidURL.Error() == "" {
			t.Error("ErrInvalidURL should have a message")
		}
		if ErrHTTPSRequired.Error() == "" {
			t.Error("ErrHTTPSRequired should have a message")
		}
		if ErrPrivateIP.Error() == "" {
			t.Error("ErrPrivateIP should have a message")
		}
		if ErrTooManyRedirects.Error() == "" {
			t.Error("ErrTooManyRedirects should have a message")
		}
		if ErrConnectionFailed.Error() == "" {
			t.Error("ErrConnectionFailed should have a message")
		}
		if ErrInvalidOIDCIssuer.Error() == "" {
			t.Error("ErrInvalidOIDCIssuer should have a message")
		}
		if ErrInvalidCalDAV.Error() == "" {
			t.Error("ErrInvalidCalDAV should have a message")
		}
	})
}

func TestValidateOIDCIssuer(t *testing.T) {
	v := New(WithAllowPrivateIPs())

	t.Run("returns error for empty URL", func(t *testing.T) {
		ctx := context.Background()
		err := v.ValidateOIDCIssuer(ctx, "")
		if !errors.Is(err, ErrInvalidOIDCIssuer) {
			t.Errorf("expected ErrInvalidOIDCIssuer, got %v", err)
		}
	})

	t.Run("returns error for HTTP URL", func(t *testing.T) {
		ctx := context.Background()
		err := v.ValidateOIDCIssuer(ctx, "http://example.com")
		if !errors.Is(err, ErrInvalidOIDCIssuer) {
			t.Errorf("expected ErrInvalidOIDCIssuer, got %v", err)
		}
	})

	t.Run("returns error for invalid URL format", func(t *testing.T) {
		ctx := context.Background()
		err := v.ValidateOIDCIssuer(ctx, "not-a-valid-url")
		if !errors.Is(err, ErrInvalidOIDCIssuer) {
			t.Errorf("expected ErrInvalidOIDCIssuer, got %v", err)
		}
	})

	t.Run("succeeds with valid OIDC discovery endpoint", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid-configuration" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"issuer": "https://example.com"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		// Create a validator with the test server's client
		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.ValidateOIDCIssuer(ctx, server.URL)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("returns error when discovery endpoint returns non-200", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.ValidateOIDCIssuer(ctx, server.URL)
		if !errors.Is(err, ErrInvalidOIDCIssuer) {
			t.Errorf("expected ErrInvalidOIDCIssuer, got %v", err)
		}
	})

	t.Run("normalizes URL with trailing slash", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid-configuration" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"issuer": "https://example.com"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.ValidateOIDCIssuer(ctx, server.URL+"/")
		if err != nil {
			t.Errorf("expected no error with trailing slash, got %v", err)
		}
	})
}

func TestValidateCalDAVEndpoint(t *testing.T) {
	t.Run("returns error for empty URL", func(t *testing.T) {
		v := New(WithAllowPrivateIPs())
		ctx := context.Background()
		err := v.ValidateCalDAVEndpoint(ctx, "")
		if !errors.Is(err, ErrInvalidCalDAV) {
			t.Errorf("expected ErrInvalidCalDAV, got %v", err)
		}
	})

	t.Run("returns error for HTTP URL", func(t *testing.T) {
		v := New(WithAllowPrivateIPs())
		ctx := context.Background()
		err := v.ValidateCalDAVEndpoint(ctx, "http://example.com")
		if !errors.Is(err, ErrInvalidCalDAV) {
			t.Errorf("expected ErrInvalidCalDAV, got %v", err)
		}
	})

	t.Run("succeeds with valid CalDAV endpoint returning 200", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("DAV", "1, 2, calendar-access")
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}))
		defer server.Close()

		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.ValidateCalDAVEndpoint(ctx, server.URL)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("succeeds with valid CalDAV endpoint returning 204", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("DAV", "1, 2, calendar-access")
				w.WriteHeader(http.StatusNoContent)
			} else {
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		}))
		defer server.Close()

		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.ValidateCalDAVEndpoint(ctx, server.URL)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("returns error when OPTIONS returns non-200/204", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.ValidateCalDAVEndpoint(ctx, server.URL)
		if !errors.Is(err, ErrInvalidCalDAV) {
			t.Errorf("expected ErrInvalidCalDAV, got %v", err)
		}
	})

	t.Run("returns error when DAV header is missing", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return 200 but no DAV header
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.ValidateCalDAVEndpoint(ctx, server.URL)
		if !errors.Is(err, ErrInvalidCalDAV) {
			t.Errorf("expected ErrInvalidCalDAV for missing DAV header, got %v", err)
		}
	})
}

func TestTestConnection(t *testing.T) {
	t.Run("returns error for empty URL", func(t *testing.T) {
		v := New(WithAllowPrivateIPs())
		ctx := context.Background()
		err := v.TestConnection(ctx, "")
		if !errors.Is(err, ErrInvalidURL) {
			t.Errorf("expected ErrInvalidURL, got %v", err)
		}
	})

	t.Run("returns error for invalid URL", func(t *testing.T) {
		v := New(WithAllowPrivateIPs())
		ctx := context.Background()
		err := v.TestConnection(ctx, "not-a-valid-url")
		if !errors.Is(err, ErrInvalidURL) {
			t.Errorf("expected ErrInvalidURL, got %v", err)
		}
	})

	t.Run("succeeds with reachable URL", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.TestConnection(ctx, server.URL)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("uses HEAD method", func(t *testing.T) {
		var methodUsed string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			methodUsed = r.Method
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		testV := &Validator{
			client:          server.Client(),
			allowPrivateIPs: true,
		}

		ctx := context.Background()
		err := testV.TestConnection(ctx, server.URL)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if methodUsed != http.MethodHead {
			t.Errorf("expected HEAD method, got %s", methodUsed)
		}
	})
}

func TestValidatorConstants(t *testing.T) {
	t.Run("constants have expected values", func(t *testing.T) {
		if maxRedirects != 3 {
			t.Errorf("expected maxRedirects to be 3, got %d", maxRedirects)
		}
		if defaultTimeout != 10*time.Second {
			t.Errorf("expected defaultTimeout to be 10s, got %v", defaultTimeout)
		}
		if minTLSVersion != tls.VersionTLS12 {
			t.Errorf("expected minTLSVersion to be TLS 1.2, got %v", minTLSVersion)
		}
	})
}

func TestValidateURLEdgeCases(t *testing.T) {
	v := New()

	t.Run("URL with fragment", func(t *testing.T) {
		err := v.ValidateURL("https://example.com/path#section", true)
		if err != nil {
			t.Errorf("expected no error for URL with fragment, got %v", err)
		}
	})

	t.Run("URL with username and password", func(t *testing.T) {
		err := v.ValidateURL("https://user:pass@example.com/path", true)
		if err != nil {
			t.Errorf("expected no error for URL with credentials, got %v", err)
		}
	})

	t.Run("URL with IPv4 address", func(t *testing.T) {
		err := v.ValidateURL("https://192.168.1.1/path", true)
		if err != nil {
			t.Errorf("expected no error for URL with IP, got %v", err)
		}
	})

	t.Run("URL with IPv6 address", func(t *testing.T) {
		err := v.ValidateURL("https://[::1]/path", true)
		if err != nil {
			t.Errorf("expected no error for URL with IPv6, got %v", err)
		}
	})

	t.Run("URL with encoded characters", func(t *testing.T) {
		err := v.ValidateURL("https://example.com/path%20with%20spaces", true)
		if err != nil {
			t.Errorf("expected no error for URL with encoded chars, got %v", err)
		}
	})

	t.Run("relative URL fails", func(t *testing.T) {
		err := v.ValidateURL("/relative/path", false)
		if !errors.Is(err, ErrInvalidURL) {
			t.Errorf("expected ErrInvalidURL for relative URL, got %v", err)
		}
	})

	t.Run("data URL fails", func(t *testing.T) {
		err := v.ValidateURL("data:text/html,<h1>Hello</h1>", false)
		if !errors.Is(err, ErrInvalidURL) {
			t.Errorf("expected ErrInvalidURL for data URL, got %v", err)
		}
	})

	t.Run("javascript URL fails", func(t *testing.T) {
		err := v.ValidateURL("javascript:alert(1)", false)
		if !errors.Is(err, ErrInvalidURL) {
			t.Errorf("expected ErrInvalidURL for javascript URL, got %v", err)
		}
	})

	t.Run("file URL fails", func(t *testing.T) {
		err := v.ValidateURL("file:///etc/passwd", false)
		if !errors.Is(err, ErrInvalidURL) {
			t.Errorf("expected ErrInvalidURL for file URL, got %v", err)
		}
	})
}

func TestIsPrivateIPEdgeCases(t *testing.T) {
	t.Run("link-local multicast addresses are blocked", func(t *testing.T) {
		// 224.0.0.1 is a link-local multicast address and should be blocked
		ip := net.ParseIP("224.0.0.1")
		result := isPrivateIP(ip)
		if !result {
			t.Error("link-local multicast address should be considered private")
		}
	})

	t.Run("non-link-local multicast addresses", func(t *testing.T) {
		// 239.1.1.1 is a multicast address but not link-local
		ip := net.ParseIP("239.1.1.1")
		result := isPrivateIP(ip)
		// Non-link-local multicast is not blocked by current implementation
		if result {
			t.Error("non-link-local multicast address should not be considered private")
		}
	})

	t.Run("broadcast address", func(t *testing.T) {
		ip := net.ParseIP("255.255.255.255")
		result := isPrivateIP(ip)
		// Broadcast is not explicitly checked
		if result {
			t.Error("broadcast address is not in private ranges")
		}
	})

	t.Run("IPv6 private addresses", func(t *testing.T) {
		// Unique Local Address (fc00::/7)
		ip := net.ParseIP("fd00::1")
		result := isPrivateIP(ip)
		if !result {
			t.Error("IPv6 ULA should be considered private")
		}
	})

	t.Run("IPv6 link-local", func(t *testing.T) {
		ip := net.ParseIP("fe80::1")
		result := isPrivateIP(ip)
		if !result {
			t.Error("IPv6 link-local should be considered private")
		}
	})
}
