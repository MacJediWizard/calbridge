package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/macjediwizard/calbridgesync/internal/auth"
	"github.com/macjediwizard/calbridgesync/internal/db"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// testHandlers holds test dependencies.
type testHandlers struct {
	db       *db.DB
	handlers *Handlers
	cleanup  func()
}

// setupTestHandlers creates handlers with a test database.
func setupTestHandlers(t *testing.T) *testHandlers {
	t.Helper()

	// Create a temp directory for the test database
	tempDir, err := os.MkdirTemp("", "calbridge-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.New(dbPath)
	if err != nil {
		os.RemoveAll(tempDir)
		t.Fatalf("failed to create test database: %v", err)
	}

	handlers := &Handlers{
		db: database,
	}

	cleanup := func() {
		database.Close()
		os.RemoveAll(tempDir)
	}

	return &testHandlers{
		db:       database,
		handlers: handlers,
		cleanup:  cleanup,
	}
}

// setAuthContext sets the authenticated user context for testing.
func setAuthContext(c *gin.Context, userID, email string) {
	session := &auth.SessionData{
		UserID:    userID,
		Email:     email,
		Name:      "Test User",
		CSRFToken: "test-csrf-token",
	}
	c.Set(auth.ContextKeySession, session)
}

// createTestUserAndSource creates a user and source for testing.
func createTestUserAndSource(t *testing.T, database *db.DB, email, sourceName string) (string, *db.Source) {
	t.Helper()

	user, err := database.GetOrCreateUser(email, "Test User")
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	source := &db.Source{
		UserID:           user.ID,
		Name:             sourceName,
		SourceType:       db.SourceTypeCustom,
		SourceURL:        "https://example.com/caldav",
		SourceUsername:   "user",
		SourcePassword:   "encrypted-password",
		DestURL:          "https://dest.com/caldav",
		DestUsername:     "destuser",
		DestPassword:     "encrypted-dest-password",
		SyncInterval:     300,
		SyncDirection:    db.SyncDirectionOneWay,
		ConflictStrategy: db.ConflictSourceWins,
		Enabled:          true,
	}

	err = database.CreateSource(source)
	if err != nil {
		t.Fatalf("failed to create source: %v", err)
	}

	return user.ID, source
}

func TestAPIDeleteAllMalformedEvents(t *testing.T) {
	t.Run("deletes all malformed events and returns count", func(t *testing.T) {
		th := setupTestHandlers(t)
		defer th.cleanup()

		// Create test data
		userID, source := createTestUserAndSource(t, th.db, "test@example.com", "Test Source")

		// Add malformed events
		th.db.SaveMalformedEvent(source.ID, "/event1.ics", "Error 1")
		th.db.SaveMalformedEvent(source.ID, "/event2.ics", "Error 2")
		th.db.SaveMalformedEvent(source.ID, "/event3.ics", "Error 3")

		// Create request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/malformed-events", nil)

		// Set auth context
		setAuthContext(c, userID, "test@example.com")

		// Call handler
		th.handlers.APIDeleteAllMalformedEvents(c)

		// Check response
		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("failed to parse response: %v", err)
		}

		if response["message"] != "All malformed events deleted" {
			t.Errorf("unexpected message: %v", response["message"])
		}

		deleted, ok := response["deleted"].(float64)
		if !ok {
			t.Fatalf("deleted field not found or wrong type")
		}
		if int(deleted) != 3 {
			t.Errorf("expected 3 deleted, got %v", deleted)
		}

		// Verify events are actually gone
		events, _ := th.db.GetMalformedEvents(userID)
		if len(events) != 0 {
			t.Errorf("expected 0 events remaining, got %d", len(events))
		}
	})

	t.Run("returns unauthorized when not authenticated", func(t *testing.T) {
		th := setupTestHandlers(t)
		defer th.cleanup()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/malformed-events", nil)

		// Don't set auth context

		th.handlers.APIDeleteAllMalformedEvents(c)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", w.Code)
		}
	})

	t.Run("returns zero when no events exist", func(t *testing.T) {
		th := setupTestHandlers(t)
		defer th.cleanup()

		// Create user without any malformed events
		user, _ := th.db.GetOrCreateUser("empty@example.com", "Empty User")

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/malformed-events", nil)

		setAuthContext(c, user.ID, "empty@example.com")

		th.handlers.APIDeleteAllMalformedEvents(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		deleted := response["deleted"].(float64)
		if int(deleted) != 0 {
			t.Errorf("expected 0 deleted, got %v", deleted)
		}
	})

	t.Run("does not delete other user's events", func(t *testing.T) {
		th := setupTestHandlers(t)
		defer th.cleanup()

		// Create two users with malformed events
		user1ID, source1 := createTestUserAndSource(t, th.db, "user1@example.com", "Source 1")
		user2ID, source2 := createTestUserAndSource(t, th.db, "user2@example.com", "Source 2")

		th.db.SaveMalformedEvent(source1.ID, "/user1/event.ics", "User1 Error")
		th.db.SaveMalformedEvent(source2.ID, "/user2/event1.ics", "User2 Error 1")
		th.db.SaveMalformedEvent(source2.ID, "/user2/event2.ics", "User2 Error 2")

		// Delete user1's events
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodDelete, "/api/malformed-events", nil)
		setAuthContext(c, user1ID, "user1@example.com")

		th.handlers.APIDeleteAllMalformedEvents(c)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", w.Code)
		}

		// User1's events should be gone
		events1, _ := th.db.GetMalformedEvents(user1ID)
		if len(events1) != 0 {
			t.Errorf("expected 0 events for user1, got %d", len(events1))
		}

		// User2's events should still exist
		events2, _ := th.db.GetMalformedEvents(user2ID)
		if len(events2) != 2 {
			t.Errorf("expected 2 events for user2, got %d", len(events2))
		}
	})
}
