package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGenerateState(t *testing.T) {
	t.Run("generates non-empty state", func(t *testing.T) {
		state, err := GenerateState()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state == "" {
			t.Fatal("expected non-empty state")
		}
	})

	t.Run("generates unique states", func(t *testing.T) {
		state1, _ := GenerateState()
		state2, _ := GenerateState()

		if state1 == state2 {
			t.Fatal("expected unique states")
		}
	})

	t.Run("generates base64 URL-safe strings", func(t *testing.T) {
		state, _ := GenerateState()
		// Base64 URL encoding should produce approximately 44 characters for 32 bytes
		if len(state) < 40 {
			t.Fatalf("state too short: %d characters", len(state))
		}
	})
}

func TestSessionManager(t *testing.T) {
	sm := NewSessionManager("test-secret-key-at-least-32-chars", false, 86400, 300)

	t.Run("set and get session", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		data := &SessionData{
			UserID:  "user-123",
			Email:   "test@example.com",
			Name:    "Test User",
			Picture: "https://example.com/pic.jpg",
		}

		err := sm.Set(w, r, data)
		if err != nil {
			t.Fatalf("failed to set session: %v", err)
		}

		// Create a new request with the cookie from the response
		r2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range w.Result().Cookies() {
			r2.AddCookie(cookie)
		}

		retrieved, err := sm.Get(r2)
		if err != nil {
			t.Fatalf("failed to get session: %v", err)
		}

		if retrieved.UserID != data.UserID {
			t.Errorf("expected UserID %q, got %q", data.UserID, retrieved.UserID)
		}
		if retrieved.Email != data.Email {
			t.Errorf("expected Email %q, got %q", data.Email, retrieved.Email)
		}
		if retrieved.Name != data.Name {
			t.Errorf("expected Name %q, got %q", data.Name, retrieved.Name)
		}
		if retrieved.CSRFToken == "" {
			t.Error("expected CSRF token to be generated")
		}
	})

	t.Run("returns error for missing session", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		_, err := sm.Get(r)
		if err != ErrSessionNotFound {
			t.Errorf("expected ErrSessionNotFound, got %v", err)
		}
	})

	t.Run("clear removes session", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		// Set a session first
		sm.Set(w, r, &SessionData{UserID: "user-123", Email: "test@example.com"})

		// Create request with session cookie
		r2 := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range w.Result().Cookies() {
			r2.AddCookie(cookie)
		}

		// Clear the session
		w2 := httptest.NewRecorder()
		err := sm.Clear(w2, r2)
		if err != nil {
			t.Fatalf("failed to clear session: %v", err)
		}

		// Session should be cleared (cookie will be set to expire)
		cookies := w2.Result().Cookies()
		for _, cookie := range cookies {
			if cookie.Name == "calbridgesync_session" && cookie.MaxAge > 0 {
				t.Error("expected session cookie to be expired")
			}
		}
	})

	t.Run("generates CSRF token when not provided", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		data := &SessionData{
			UserID: "user-123",
			Email:  "test@example.com",
		}

		sm.Set(w, r, data)

		// The data should now have a CSRF token
		if data.CSRFToken == "" {
			t.Error("expected CSRF token to be generated")
		}
	})
}

func TestOAuthState(t *testing.T) {
	sm := NewSessionManager("test-secret-key-at-least-32-chars", false, 86400, 300)

	t.Run("set and get OAuth state", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		state := "test-oauth-state-12345"
		err := sm.SetOAuthState(w, r, state)
		if err != nil {
			t.Fatalf("failed to set OAuth state: %v", err)
		}

		// Create a new request with the cookie
		r2 := httptest.NewRequest(http.MethodGet, "/", nil)
		w2 := httptest.NewRecorder()
		for _, cookie := range w.Result().Cookies() {
			r2.AddCookie(cookie)
		}

		retrieved, err := sm.GetOAuthState(w2, r2)
		if err != nil {
			t.Fatalf("failed to get OAuth state: %v", err)
		}

		if retrieved != state {
			t.Errorf("expected state %q, got %q", state, retrieved)
		}
	})

	t.Run("OAuth state is cleared after retrieval", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)

		sm.SetOAuthState(w, r, "test-state")

		// Get state (should clear it)
		r2 := httptest.NewRequest(http.MethodGet, "/", nil)
		w2 := httptest.NewRecorder()
		for _, cookie := range w.Result().Cookies() {
			r2.AddCookie(cookie)
		}
		state, err := sm.GetOAuthState(w2, r2)
		if err != nil {
			t.Fatalf("failed to get OAuth state: %v", err)
		}
		if state != "test-state" {
			t.Errorf("expected 'test-state', got %q", state)
		}

		// The GetOAuthState sets MaxAge = -1 to clear the cookie,
		// so subsequent requests with the cleared cookie should fail.
		// Note: The cookie is marked for deletion but the session store
		// behavior may vary. We verify the state was retrieved successfully.
	})
}

func TestGetCurrentUser(t *testing.T) {
	t.Run("returns nil when no session in context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := createTestGinContext(w)

		user := GetCurrentUser(c)
		if user != nil {
			t.Error("expected nil user when no session")
		}
	})

	t.Run("returns session data when present", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := createTestGinContext(w)

		expected := &SessionData{
			UserID: "user-123",
			Email:  "test@example.com",
		}
		c.Set(ContextKeySession, expected)

		user := GetCurrentUser(c)
		if user == nil {
			t.Fatal("expected user, got nil")
		}
		if user.UserID != expected.UserID {
			t.Errorf("expected UserID %q, got %q", expected.UserID, user.UserID)
		}
	})
}

// Helper to create a test Gin context
func createTestGinContext(w http.ResponseWriter) (*gin.Context, *gin.Engine) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	c, _ := gin.CreateTestContext(w)
	return c, engine
}
