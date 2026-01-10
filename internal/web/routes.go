package web

import (
	"github.com/gin-gonic/gin"
	"github.com/macjediwizard/calbridge/internal/auth"
)

// SetupRoutes configures all application routes.
func SetupRoutes(r *gin.Engine, h *Handlers, sm *auth.SessionManager) {
	// Health endpoints (no auth, no rate limit)
	r.GET("/health", h.HealthCheck)
	r.GET("/healthz", h.Liveness)
	r.GET("/ready", h.Readiness)

	// Auth endpoints (no auth required)
	authGroup := r.Group("/auth")
	{
		authGroup.GET("/login", h.LoginPage)
		authGroup.POST("/login", h.Login)
		authGroup.GET("/callback", h.Callback)
		authGroup.POST("/logout", h.Logout)
	}

	// Protected routes
	protected := r.Group("/")
	protected.Use(auth.RequireAuth(sm))
	{
		protected.GET("/", h.Dashboard)
		protected.GET("/sources", h.ListSources)
		protected.GET("/sources/add", h.AddSourcePage)
		protected.POST("/sources/add", h.AddSource)
		protected.GET("/sources/:id/edit", h.EditSourcePage)
		protected.POST("/sources/:id", h.UpdateSource)
		protected.DELETE("/sources/:id", h.DeleteSource)
		protected.POST("/sources/:id/sync", h.TriggerSync)
		protected.POST("/sources/:id/toggle", h.ToggleSource)
		protected.GET("/sources/:id/logs", h.ViewLogs)
	}
}
