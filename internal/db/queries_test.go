package db

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestDB creates a temporary test database.
func setupTestDB(t *testing.T) (*DB, func()) {
	t.Helper()

	// Create a temp directory for the test database
	tempDir, err := os.MkdirTemp("", "calbridge-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := New(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tempDir)
	}

	return db, cleanup
}

// createTestUser creates a test user and returns the user ID.
func createTestUser(t *testing.T, db *DB, email string) string {
	t.Helper()

	user, err := db.GetOrCreateUser(email, "Test User")
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return user.ID
}

// createTestSource creates a test source for a user.
func createTestSource(t *testing.T, db *DB, userID, name string) *Source {
	t.Helper()

	source := &Source{
		UserID:           userID,
		Name:             name,
		SourceType:       SourceTypeCustom,
		SourceURL:        "https://example.com/caldav",
		SourceUsername:   "user",
		SourcePassword:   "encrypted-password",
		DestURL:          "https://dest.com/caldav",
		DestUsername:     "destuser",
		DestPassword:     "encrypted-dest-password",
		SyncInterval:     300,
		SyncDirection:    SyncDirectionOneWay,
		ConflictStrategy: ConflictSourceWins,
		Enabled:          true,
	}

	err := db.CreateSource(source)
	if err != nil {
		t.Fatalf("failed to create test source: %v", err)
	}
	return source
}

func TestDeleteAllMalformedEventsForUser(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("deletes all malformed events for user", func(t *testing.T) {
		// Create test user and source
		userID := createTestUser(t, db, "test@example.com")
		source := createTestSource(t, db, userID, "Test Source")

		// Create some malformed events
		err := db.SaveMalformedEvent(source.ID, "/calendar/event1.ics", "Error 1")
		if err != nil {
			t.Fatalf("failed to save malformed event: %v", err)
		}
		err = db.SaveMalformedEvent(source.ID, "/calendar/event2.ics", "Error 2")
		if err != nil {
			t.Fatalf("failed to save malformed event: %v", err)
		}
		err = db.SaveMalformedEvent(source.ID, "/calendar/event3.ics", "Error 3")
		if err != nil {
			t.Fatalf("failed to save malformed event: %v", err)
		}

		// Verify events exist
		events, err := db.GetMalformedEvents(userID)
		if err != nil {
			t.Fatalf("failed to get malformed events: %v", err)
		}
		if len(events) != 3 {
			t.Fatalf("expected 3 events, got %d", len(events))
		}

		// Delete all malformed events
		deleted, err := db.DeleteAllMalformedEventsForUser(userID)
		if err != nil {
			t.Fatalf("failed to delete all malformed events: %v", err)
		}
		if deleted != 3 {
			t.Fatalf("expected 3 deleted, got %d", deleted)
		}

		// Verify events are gone
		events, err = db.GetMalformedEvents(userID)
		if err != nil {
			t.Fatalf("failed to get malformed events: %v", err)
		}
		if len(events) != 0 {
			t.Fatalf("expected 0 events, got %d", len(events))
		}
	})

	t.Run("does not delete other user's events", func(t *testing.T) {
		// Create two test users
		user1ID := createTestUser(t, db, "user1@example.com")
		user2ID := createTestUser(t, db, "user2@example.com")

		source1 := createTestSource(t, db, user1ID, "User1 Source")
		source2 := createTestSource(t, db, user2ID, "User2 Source")

		// Create events for both users
		db.SaveMalformedEvent(source1.ID, "/user1/event1.ics", "User1 Error")
		db.SaveMalformedEvent(source2.ID, "/user2/event1.ics", "User2 Error")
		db.SaveMalformedEvent(source2.ID, "/user2/event2.ics", "User2 Error 2")

		// Delete user1's events
		deleted, err := db.DeleteAllMalformedEventsForUser(user1ID)
		if err != nil {
			t.Fatalf("failed to delete: %v", err)
		}
		if deleted != 1 {
			t.Fatalf("expected 1 deleted for user1, got %d", deleted)
		}

		// User2's events should still exist
		events, err := db.GetMalformedEvents(user2ID)
		if err != nil {
			t.Fatalf("failed to get user2 events: %v", err)
		}
		if len(events) != 2 {
			t.Fatalf("expected 2 events for user2, got %d", len(events))
		}
	})

	t.Run("returns zero when no events exist", func(t *testing.T) {
		userID := createTestUser(t, db, "empty@example.com")

		deleted, err := db.DeleteAllMalformedEventsForUser(userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if deleted != 0 {
			t.Fatalf("expected 0 deleted, got %d", deleted)
		}
	})

	t.Run("handles multiple sources for same user", func(t *testing.T) {
		userID := createTestUser(t, db, "multi@example.com")
		source1 := createTestSource(t, db, userID, "Source 1")
		source2 := createTestSource(t, db, userID, "Source 2")

		// Add events to both sources
		db.SaveMalformedEvent(source1.ID, "/source1/event.ics", "Error")
		db.SaveMalformedEvent(source2.ID, "/source2/event.ics", "Error")

		// Delete all
		deleted, err := db.DeleteAllMalformedEventsForUser(userID)
		if err != nil {
			t.Fatalf("failed to delete: %v", err)
		}
		if deleted != 2 {
			t.Fatalf("expected 2 deleted, got %d", deleted)
		}

		// Verify all gone
		events, err := db.GetMalformedEvents(userID)
		if err != nil {
			t.Fatalf("failed to get events: %v", err)
		}
		if len(events) != 0 {
			t.Fatalf("expected 0 events, got %d", len(events))
		}
	})
}
