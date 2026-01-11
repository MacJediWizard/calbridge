package validator

import (
	"errors"
	"net"
	"testing"
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
