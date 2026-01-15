package notify

import (
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid webhook config",
			cfg: &Config{
				WebhookEnabled: true,
				WebhookURL:     "https://hooks.slack.com/services/xxx",
				CooldownPeriod: time.Hour,
			},
			wantErr: false,
		},
		{
			name: "webhook with HTTP (not HTTPS) fails",
			cfg: &Config{
				WebhookEnabled: true,
				WebhookURL:     "http://hooks.slack.com/services/xxx",
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "HTTPS",
		},
		{
			name: "webhook pointing to localhost fails",
			cfg: &Config{
				WebhookEnabled: true,
				WebhookURL:     "https://localhost:8080/webhook",
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "localhost",
		},
		{
			name: "webhook pointing to private IP fails",
			cfg: &Config{
				WebhookEnabled: true,
				WebhookURL:     "https://192.168.1.1/webhook",
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "private IP",
		},
		{
			name: "webhook pointing to 10.x.x.x fails",
			cfg: &Config{
				WebhookEnabled: true,
				WebhookURL:     "https://10.0.0.1/webhook",
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "private IP",
		},
		{
			name: "webhook with empty URL fails",
			cfg: &Config{
				WebhookEnabled: true,
				WebhookURL:     "",
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "required",
		},
		{
			name: "valid email config",
			cfg: &Config{
				EmailEnabled:   true,
				SMTPHost:       "smtp.gmail.com",
				SMTPPort:       587,
				SMTPFrom:       "alerts@example.com",
				SMTPTo:         []string{"admin@example.com"},
				CooldownPeriod: time.Hour,
			},
			wantErr: false,
		},
		{
			name: "email with invalid port fails",
			cfg: &Config{
				EmailEnabled:   true,
				SMTPHost:       "smtp.gmail.com",
				SMTPPort:       0,
				SMTPFrom:       "alerts@example.com",
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "port",
		},
		{
			name: "email with missing host fails",
			cfg: &Config{
				EmailEnabled:   true,
				SMTPPort:       587,
				SMTPFrom:       "alerts@example.com",
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "host",
		},
		{
			name: "email with invalid from address fails",
			cfg: &Config{
				EmailEnabled:   true,
				SMTPHost:       "smtp.gmail.com",
				SMTPPort:       587,
				SMTPFrom:       "not-an-email",
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "from address",
		},
		{
			name: "email with invalid recipient fails",
			cfg: &Config{
				EmailEnabled:   true,
				SMTPHost:       "smtp.gmail.com",
				SMTPPort:       587,
				SMTPFrom:       "alerts@example.com",
				SMTPTo:         []string{"invalid-email"},
				CooldownPeriod: time.Hour,
			},
			wantErr: true,
			errMsg:  "recipient",
		},
		{
			name: "cooldown too short fails",
			cfg: &Config{
				WebhookEnabled: true,
				WebhookURL:     "https://hooks.slack.com/services/xxx",
				CooldownPeriod: 30 * time.Second,
			},
			wantErr: true,
			errMsg:  "minute",
		},
		{
			name: "disabled notifications need no validation",
			cfg: &Config{
				WebhookEnabled: false,
				EmailEnabled:   false,
				CooldownPeriod: time.Hour,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if !contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateConfig() error = %v, expected to contain %q", err, tt.errMsg)
				}
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		email string
		valid bool
	}{
		{"user@example.com", true},
		{"user.name@example.com", true},
		{"user+tag@example.com", true},
		{"user@sub.example.com", true},
		{"not-an-email", false},
		{"@example.com", false},
		{"user@", false},
		{"user@.com", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			if got := isValidEmail(tt.email); got != tt.valid {
				t.Errorf("isValidEmail(%q) = %v, want %v", tt.email, got, tt.valid)
			}
		})
	}
}

func TestSanitizeForEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "normal text unchanged",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "CR removed",
			input: "Line1\rLine2",
			want:  "Line1Line2",
		},
		{
			name:  "LF replaced with space",
			input: "Line1\nLine2",
			want:  "Line1 Line2",
		},
		{
			name:  "CRLF header injection attempt blocked",
			input: "Subject\r\nBcc: attacker@evil.com",
			want:  "Subject Bcc: attacker@evil.com",
		},
		{
			name:  "long string truncated",
			input: string(make([]byte, 300)),
			want:  string(make([]byte, 200)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeForEmail(tt.input); got != tt.want {
				t.Errorf("sanitizeForEmail() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"valid HTTPS URL", "https://hooks.slack.com/services/xxx", false},
		{"HTTP not allowed", "http://example.com/webhook", true},
		{"localhost not allowed", "https://localhost/webhook", true},
		{"127.0.0.1 not allowed", "https://127.0.0.1/webhook", true},
		{"192.168.x.x not allowed", "https://192.168.0.1/webhook", true},
		{"10.x.x.x not allowed", "https://10.0.0.1/webhook", true},
		{"172.16.x.x not allowed", "https://172.16.0.1/webhook", true},
		{".local domain not allowed", "https://server.local/webhook", true},
		{".internal domain not allowed", "https://api.internal/webhook", true},
		{"invalid URL", "not-a-url", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWebhookURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateWebhookURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
			}
		})
	}
}

func TestNotifierCooldown(t *testing.T) {
	cfg := &Config{
		WebhookEnabled: false,
		EmailEnabled:   false,
		CooldownPeriod: time.Hour,
	}
	n := New(cfg)

	// First alert should succeed
	sent := n.SendStaleAlert(nil, "source1", "Test Source", "user@example.com", 2*time.Hour, time.Hour)
	if !sent {
		t.Error("First stale alert should be sent")
	}

	// Second alert within cooldown should be blocked
	sent = n.SendStaleAlert(nil, "source1", "Test Source", "user@example.com", 2*time.Hour, time.Hour)
	if sent {
		t.Error("Second stale alert within cooldown should be blocked")
	}

	// Different source should still work
	sent = n.SendStaleAlert(nil, "source2", "Test Source 2", "user@example.com", 2*time.Hour, time.Hour)
	if !sent {
		t.Error("Alert for different source should be sent")
	}
}

func TestNotifierRecovery(t *testing.T) {
	cfg := &Config{
		WebhookEnabled: false,
		EmailEnabled:   false,
		CooldownPeriod: time.Hour,
	}
	n := New(cfg)

	// Recovery without prior stale should return false
	recovered := n.SendRecoveryAlert(nil, "source1", "Test Source", "user@example.com")
	if recovered {
		t.Error("Recovery without prior stale should return false")
	}

	// Send stale alert first
	n.SendStaleAlert(nil, "source1", "Test Source", "user@example.com", 2*time.Hour, time.Hour)

	// Now recovery should work
	recovered = n.SendRecoveryAlert(nil, "source1", "Test Source", "user@example.com")
	if !recovered {
		t.Error("Recovery after stale should return true")
	}

	// Second recovery should fail (already recovered)
	recovered = n.SendRecoveryAlert(nil, "source1", "Test Source", "user@example.com")
	if recovered {
		t.Error("Second recovery should return false")
	}
}

func TestClearStaleState(t *testing.T) {
	cfg := &Config{
		WebhookEnabled: false,
		EmailEnabled:   false,
		CooldownPeriod: time.Hour,
	}
	n := New(cfg)

	// Send stale alert
	n.SendStaleAlert(nil, "source1", "Test Source", "user@example.com", 2*time.Hour, time.Hour)

	// Clear stale state
	n.ClearStaleState("source1")

	// Should be able to send stale alert again (not in cooldown)
	sent := n.SendStaleAlert(nil, "source1", "Test Source", "user@example.com", 2*time.Hour, time.Hour)
	if !sent {
		t.Error("Should be able to send alert after clearing stale state")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
