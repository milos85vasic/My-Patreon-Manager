package handlers

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/audit"
)

// interfaces for mocking
type tierGater interface {
	VerifyAccess(ctx context.Context, patronID, contentID, requiredTier string, patronTiers []string) (bool, string, error)
}

type signedURLGenerator interface {
	VerifySignedURL(token, contentID, subscriberID string, expires int64) bool
}

type AccessHandler struct {
	gater  tierGater
	urlGen signedURLGenerator
	logger *slog.Logger
	// audit is the structured audit-log sink. Always non-nil after
	// NewAccessHandler: defaults to a bounded ring store. Each download
	// or access check emits exactly one audit.Entry — see Phase 2 Task 2.
	audit audit.Store
}

func NewAccessHandler(gater tierGater, urlGen signedURLGenerator, logger *slog.Logger) *AccessHandler {
	return &AccessHandler{
		gater:  gater,
		urlGen: urlGen,
		logger: logger,
		audit:  audit.NewRingStore(1024),
	}
}

// SetAuditStore replaces the handler's audit sink. Passing nil resets it to a
// bounded in-memory ring store so the handler never holds a nil audit.Store.
func (h *AccessHandler) SetAuditStore(s audit.Store) {
	if s == nil {
		s = audit.NewRingStore(1024)
	}
	h.audit = s
}

// AuditStore returns the handler's current audit sink. Test-only accessor.
func (h *AccessHandler) AuditStore() audit.Store { return h.audit }

func (h *AccessHandler) emitAudit(ctx context.Context, e audit.Entry) {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	_ = h.audit.Write(ctx, e)
}

func (h *AccessHandler) Download(c *gin.Context) {
	contentID := c.Param("content_id")
	token := c.Query("token")
	sub := c.Query("sub")
	expStr := c.Query("exp")

	if token == "" || sub == "" || expStr == "" {
		h.emitAudit(c.Request.Context(), audit.Entry{
			Actor:    "access",
			Action:   "access.download",
			Target:   contentID,
			Outcome:  "error",
			Metadata: map[string]string{"error": "missing token parameters"},
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing token parameters"})
		return
	}

	expires, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil {
		h.emitAudit(c.Request.Context(), audit.Entry{
			Actor:    "access",
			Action:   "access.download",
			Target:   contentID,
			Outcome:  "error",
			Metadata: map[string]string{"error": "invalid expiry"},
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expiry"})
		return
	}

	if !h.urlGen.VerifySignedURL(token, contentID, sub, expires) {
		h.emitAudit(c.Request.Context(), audit.Entry{
			Actor:    "access",
			Action:   "access.download",
			Target:   contentID,
			Outcome:  "denied",
			Metadata: map[string]string{"sub": sub},
		})
		c.JSON(http.StatusForbidden, gin.H{"error": "invalid or expired token"})
		return
	}

	h.emitAudit(c.Request.Context(), audit.Entry{
		Actor:    "access",
		Action:   "access.download",
		Target:   contentID,
		Outcome:  "ok",
		Metadata: map[string]string{"sub": sub},
	})

	c.Header("Content-Disposition", "attachment; filename="+contentID)
	c.JSON(http.StatusOK, gin.H{"content_id": contentID, "status": "download_ready"})
}

func (h *AccessHandler) CheckAccess(c *gin.Context) {
	contentID := c.Param("content_id")
	patronID := c.Query("patron_id")
	requiredTier := c.Query("required_tier")

	if patronID == "" || requiredTier == "" {
		h.emitAudit(c.Request.Context(), audit.Entry{
			Actor:    "access",
			Action:   "access.check",
			Target:   contentID,
			Outcome:  "error",
			Metadata: map[string]string{"error": "missing patron_id or required_tier"},
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing patron_id or required_tier"})
		return
	}

	hasAccess, upgradeURL, _ := h.gater.VerifyAccess(c.Request.Context(), patronID, contentID, requiredTier, nil)

	response := gin.H{
		"access":        hasAccess,
		"content_id":    contentID,
		"required_tier": requiredTier,
	}

	outcome := "ok"
	if !hasAccess {
		response["upgrade_url"] = upgradeURL
		outcome = "denied"
	}
	h.emitAudit(c.Request.Context(), audit.Entry{
		Actor:    "access",
		Action:   "access.check",
		Target:   contentID,
		Outcome:  outcome,
		Metadata: map[string]string{"patron_id": patronID, "required_tier": requiredTier},
	})

	c.JSON(http.StatusOK, response)
}
