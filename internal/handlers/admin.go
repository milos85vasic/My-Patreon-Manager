package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/audit"
)

// AdminHandler exposes admin endpoints alongside an audit sink. The
// constructor always installs a non-nil ring store; SetAuditStore swaps the
// backend for callers wanting persistence.
type AdminHandler struct {
	logger *slog.Logger
	audit  audit.Store
}

func NewAdminHandler(logger *slog.Logger) *AdminHandler {
	return &AdminHandler{logger: logger, audit: audit.NewRingStore(1024)}
}

// SetAuditStore replaces the handler's audit sink. Passing nil resets it to a
// bounded in-memory ring store.
func (h *AdminHandler) SetAuditStore(s audit.Store) {
	if s == nil {
		s = audit.NewRingStore(1024)
	}
	h.audit = s
}

// AuditStore returns the handler's current audit sink. Test-only accessor.
func (h *AdminHandler) AuditStore() audit.Store { return h.audit }

func (h *AdminHandler) emitAudit(ctx context.Context, e audit.Entry) {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	_ = h.audit.Write(ctx, e)
}

// Reload handles POST /admin/reload. Emits an audit entry on success.
func (h *AdminHandler) Reload(c *gin.Context) {
	h.emitAudit(c.Request.Context(), audit.Entry{
		Actor:   "admin",
		Action:  "admin.reload",
		Target:  "config",
		Outcome: "ok",
	})
	c.JSON(http.StatusOK, gin.H{"status": "config_reloaded"})
}

// SyncStatus handles GET /admin/sync/status. Read-only path; no audit.
func (h *AdminHandler) SyncStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "idle",
		"active_sync": false,
	})
}

// AdminReload retains the original package-level function to avoid breaking
// existing call sites. Routes registered through it carry no audit sink.
func AdminReload(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "config_reloaded"})
}

func AdminSyncStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":      "idle",
		"active_sync": false,
	})
}

func RegisterAdminRoutes(r *gin.Engine, logger *slog.Logger) {
	admin := r.Group("/admin")
	{
		admin.POST("/reload", AdminReload)
		admin.GET("/sync/status", AdminSyncStatus)
	}
}
