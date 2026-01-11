package caldav

import (
	"testing"
)

func TestEventDedupeKey(t *testing.T) {
	testCases := []struct {
		name      string
		event     Event
		expected  string
	}{
		{
			name: "combines summary and start time",
			event: Event{
				Summary:   "Team Meeting",
				StartTime: "20240115T140000Z",
			},
			expected: "Team Meeting|20240115T140000Z",
		},
		{
			name: "handles empty summary",
			event: Event{
				Summary:   "",
				StartTime: "20240115T140000Z",
			},
			expected: "|20240115T140000Z",
		},
		{
			name: "handles empty start time",
			event: Event{
				Summary:   "Team Meeting",
				StartTime: "",
			},
			expected: "Team Meeting|",
		},
		{
			name: "handles both empty",
			event: Event{
				Summary:   "",
				StartTime: "",
			},
			expected: "|",
		},
		{
			name: "handles special characters in summary",
			event: Event{
				Summary:   "Meeting: Q1 Review & Planning",
				StartTime: "20240115T140000Z",
			},
			expected: "Meeting: Q1 Review & Planning|20240115T140000Z",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.event.DedupeKey()
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestMalformedEventCollector(t *testing.T) {
	t.Run("creates empty collector", func(t *testing.T) {
		collector := NewMalformedEventCollector()

		if collector == nil {
			t.Fatal("expected non-nil collector")
		}

		if collector.Count() != 0 {
			t.Errorf("expected count 0, got %d", collector.Count())
		}

		events := collector.GetEvents()
		if len(events) != 0 {
			t.Errorf("expected empty events, got %d", len(events))
		}
	})

	t.Run("adds and retrieves events", func(t *testing.T) {
		collector := NewMalformedEventCollector()

		collector.Add("/calendar/event1.ics", "Invalid VCALENDAR format")
		collector.Add("/calendar/event2.ics", "Missing DTSTART")

		if collector.Count() != 2 {
			t.Errorf("expected count 2, got %d", collector.Count())
		}

		events := collector.GetEvents()
		if len(events) != 2 {
			t.Fatalf("expected 2 events, got %d", len(events))
		}

		if events[0].Path != "/calendar/event1.ics" {
			t.Errorf("expected path '/calendar/event1.ics', got %q", events[0].Path)
		}
		if events[0].ErrorMessage != "Invalid VCALENDAR format" {
			t.Errorf("unexpected error message: %q", events[0].ErrorMessage)
		}

		if events[1].Path != "/calendar/event2.ics" {
			t.Errorf("expected path '/calendar/event2.ics', got %q", events[1].Path)
		}
	})

	t.Run("preserves order of events", func(t *testing.T) {
		collector := NewMalformedEventCollector()

		collector.Add("/first.ics", "error 1")
		collector.Add("/second.ics", "error 2")
		collector.Add("/third.ics", "error 3")

		events := collector.GetEvents()

		expectedPaths := []string{"/first.ics", "/second.ics", "/third.ics"}
		for i, path := range expectedPaths {
			if events[i].Path != path {
				t.Errorf("expected path %q at index %d, got %q", path, i, events[i].Path)
			}
		}
	})
}

func TestCalendarStruct(t *testing.T) {
	t.Run("calendar struct has expected fields", func(t *testing.T) {
		cal := Calendar{
			Path:        "/dav/calendars/user/default/",
			Name:        "Default Calendar",
			Description: "My default calendar",
			Color:       "#FF5733",
			SyncToken:   "sync-token-123",
			CTag:        "ctag-456",
		}

		if cal.Path != "/dav/calendars/user/default/" {
			t.Error("Path not set correctly")
		}
		if cal.Name != "Default Calendar" {
			t.Error("Name not set correctly")
		}
		if cal.Description != "My default calendar" {
			t.Error("Description not set correctly")
		}
		if cal.Color != "#FF5733" {
			t.Error("Color not set correctly")
		}
		if cal.SyncToken != "sync-token-123" {
			t.Error("SyncToken not set correctly")
		}
		if cal.CTag != "ctag-456" {
			t.Error("CTag not set correctly")
		}
	})
}

func TestEventStruct(t *testing.T) {
	t.Run("event struct has expected fields", func(t *testing.T) {
		event := Event{
			Path:      "/calendar/event.ics",
			ETag:      "etag-123",
			Data:      "BEGIN:VCALENDAR\nEND:VCALENDAR",
			UID:       "unique-id@example.com",
			Summary:   "Test Event",
			StartTime: "20240115T140000Z",
		}

		if event.Path != "/calendar/event.ics" {
			t.Error("Path not set correctly")
		}
		if event.ETag != "etag-123" {
			t.Error("ETag not set correctly")
		}
		if event.UID != "unique-id@example.com" {
			t.Error("UID not set correctly")
		}
		if event.Summary != "Test Event" {
			t.Error("Summary not set correctly")
		}
	})
}

func TestErrorConstants(t *testing.T) {
	t.Run("error constants are not nil", func(t *testing.T) {
		errors := []error{
			ErrConnectionFailed,
			ErrAuthFailed,
			ErrNotFound,
			ErrInvalidResponse,
			ErrMalformedContent,
		}

		for _, err := range errors {
			if err == nil {
				t.Error("expected error constant to be non-nil")
			}
		}
	})

	t.Run("error messages are descriptive", func(t *testing.T) {
		if ErrConnectionFailed.Error() != "connection failed" {
			t.Errorf("unexpected error message: %q", ErrConnectionFailed.Error())
		}
		if ErrAuthFailed.Error() != "authentication failed" {
			t.Errorf("unexpected error message: %q", ErrAuthFailed.Error())
		}
		if ErrNotFound.Error() != "resource not found" {
			t.Errorf("unexpected error message: %q", ErrNotFound.Error())
		}
	})
}

func TestMalformedEventInfo(t *testing.T) {
	t.Run("struct holds path and error message", func(t *testing.T) {
		info := MalformedEventInfo{
			Path:         "/calendar/broken.ics",
			ErrorMessage: "invalid iCalendar format",
		}

		if info.Path != "/calendar/broken.ics" {
			t.Errorf("expected path '/calendar/broken.ics', got %q", info.Path)
		}
		if info.ErrorMessage != "invalid iCalendar format" {
			t.Errorf("expected error message, got %q", info.ErrorMessage)
		}
	})
}
