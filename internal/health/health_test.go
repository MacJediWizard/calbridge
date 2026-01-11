package health

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLiveness(t *testing.T) {
	checker := &Checker{}

	t.Run("returns healthy status", func(t *testing.T) {
		report := checker.Liveness()

		if report == nil {
			t.Fatal("expected non-nil report")
		}

		if report.Status != StatusHealthy {
			t.Errorf("expected status %q, got %q", StatusHealthy, report.Status)
		}

		if report.Timestamp.IsZero() {
			t.Error("expected non-zero timestamp")
		}

		aliveCheck, ok := report.Checks["alive"]
		if !ok {
			t.Fatal("expected 'alive' check in report")
		}

		if aliveCheck.Status != StatusHealthy {
			t.Errorf("expected alive check to be healthy, got %q", aliveCheck.Status)
		}

		if aliveCheck.Name != "alive" {
			t.Errorf("expected check name 'alive', got %q", aliveCheck.Name)
		}
	})
}

func TestDetermineOverallStatus(t *testing.T) {
	checker := &Checker{}

	testCases := []struct {
		name     string
		checks   map[string]Check
		expected Status
	}{
		{
			name: "all healthy",
			checks: map[string]Check{
				"database": {Status: StatusHealthy},
				"oidc":     {Status: StatusHealthy},
				"caldav":   {Status: StatusHealthy},
			},
			expected: StatusHealthy,
		},
		{
			name: "database unhealthy makes overall unhealthy",
			checks: map[string]Check{
				"database": {Status: StatusUnhealthy},
				"oidc":     {Status: StatusHealthy},
			},
			expected: StatusUnhealthy,
		},
		{
			name: "oidc unhealthy makes overall unhealthy",
			checks: map[string]Check{
				"database": {Status: StatusHealthy},
				"oidc":     {Status: StatusUnhealthy},
			},
			expected: StatusUnhealthy,
		},
		{
			name: "caldav degraded makes overall degraded",
			checks: map[string]Check{
				"database": {Status: StatusHealthy},
				"oidc":     {Status: StatusHealthy},
				"caldav":   {Status: StatusDegraded},
			},
			expected: StatusDegraded,
		},
		{
			name: "any unhealthy makes overall unhealthy",
			checks: map[string]Check{
				"database": {Status: StatusHealthy},
				"oidc":     {Status: StatusHealthy},
				"other":    {Status: StatusUnhealthy},
			},
			expected: StatusUnhealthy,
		},
		{
			name:     "empty checks is healthy",
			checks:   map[string]Check{},
			expected: StatusHealthy,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := checker.determineOverallStatus(tc.checks)
			if result != tc.expected {
				t.Errorf("expected status %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	t.Run("status constants are correct", func(t *testing.T) {
		if StatusHealthy != "healthy" {
			t.Errorf("expected StatusHealthy to be 'healthy', got %q", StatusHealthy)
		}
		if StatusUnhealthy != "unhealthy" {
			t.Errorf("expected StatusUnhealthy to be 'unhealthy', got %q", StatusUnhealthy)
		}
		if StatusDegraded != "degraded" {
			t.Errorf("expected StatusDegraded to be 'degraded', got %q", StatusDegraded)
		}
	})
}

func TestCheckMarshalJSON(t *testing.T) {
	t.Run("marshals latency as milliseconds", func(t *testing.T) {
		check := Check{
			Name:    "test",
			Status:  StatusHealthy,
			Message: "test message",
			Latency: 150 * time.Millisecond,
		}

		data, err := json.Marshal(check)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		latency, ok := result["latency_ms"].(float64)
		if !ok {
			t.Fatal("expected latency_ms field")
		}

		if int(latency) != 150 {
			t.Errorf("expected latency_ms 150, got %v", latency)
		}
	})

	t.Run("includes all fields in JSON", func(t *testing.T) {
		check := Check{
			Name:    "database",
			Status:  StatusHealthy,
			Message: "connected",
			Latency: 50 * time.Millisecond,
		}

		data, err := json.Marshal(check)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var result map[string]interface{}
		json.Unmarshal(data, &result)

		if result["name"] != "database" {
			t.Errorf("expected name 'database', got %v", result["name"])
		}
		if result["status"] != "healthy" {
			t.Errorf("expected status 'healthy', got %v", result["status"])
		}
		if result["message"] != "connected" {
			t.Errorf("expected message 'connected', got %v", result["message"])
		}
	})
}

func TestLastReport(t *testing.T) {
	t.Run("returns nil when no check performed", func(t *testing.T) {
		checker := &Checker{}

		report := checker.LastReport()
		if report != nil {
			t.Error("expected nil report when no check performed")
		}
	})
}

func TestNewChecker(t *testing.T) {
	t.Run("creates checker with provided values", func(t *testing.T) {
		// Note: We can't easily test with a real DB here, so we just verify creation
		checker := NewChecker(nil, "https://auth.example.com", "https://caldav.example.com")

		if checker == nil {
			t.Fatal("expected non-nil checker")
		}

		if checker.oidcIssuer != "https://auth.example.com" {
			t.Errorf("expected oidcIssuer to be set")
		}

		if checker.caldavURL != "https://caldav.example.com" {
			t.Errorf("expected caldavURL to be set")
		}

		if checker.httpClient == nil {
			t.Error("expected httpClient to be created")
		}
	})
}
