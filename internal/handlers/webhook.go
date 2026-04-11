package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/milos85vasic/My-Patreon-Manager/internal/metrics"
	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/audit"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
)

func splitFullName(full string) (owner, name string) {
	parts := strings.Split(full, "/")
	if len(parts) >= 2 {
		owner = parts[0]
		name = parts[1]
	}
	return
}

// DefaultWebhookQueueCapacity is the default bounded capacity used when a
// caller does not inject its own WebhookQueue.
const DefaultWebhookQueueCapacity = 1024

type WebhookHandler struct {
	dedup   *sync.EventDeduplicator
	metrics metrics.MetricsCollector
	logger  *slog.Logger
	// Queue is a required, bounded queue of repositories produced by webhook
	// events. It is never nil after NewWebhookHandler; callers may replace it
	// before registering routes but must keep it non-nil. Overflow returns
	// HTTP 429 rather than silently dropping events.
	Queue *WebhookQueue[models.Repository]
	// audit is the structured audit-log sink. Always non-nil after
	// NewWebhookHandler: defaults to a bounded ring store. Each enqueue
	// or rejection emits exactly one audit.Entry — see Phase 2 Task 2.
	audit audit.Store
}

func NewWebhookHandler(dedup *sync.EventDeduplicator, m metrics.MetricsCollector, logger *slog.Logger) *WebhookHandler {
	return &WebhookHandler{
		dedup:   dedup,
		metrics: m,
		logger:  logger,
		Queue:   NewWebhookQueue[models.Repository](DefaultWebhookQueueCapacity),
		audit:   audit.NewRingStore(1024),
	}
}

// SetAuditStore replaces the handler's audit sink. Passing nil resets it to a
// bounded in-memory ring store so the handler never holds a nil audit.Store.
func (h *WebhookHandler) SetAuditStore(s audit.Store) {
	if s == nil {
		s = audit.NewRingStore(1024)
	}
	h.audit = s
}

// AuditStore returns the handler's current audit sink. Test-only accessor.
func (h *WebhookHandler) AuditStore() audit.Store { return h.audit }

func (h *WebhookHandler) emitAudit(ctx context.Context, e audit.Entry) {
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	_ = h.audit.Write(ctx, e)
}

func (h *WebhookHandler) GitHubWebhook(c *gin.Context) {
	eventID := c.GetHeader("X-GitHub-Delivery")
	eventType := c.GetHeader("X-GitHub-Event")

	if h.logger != nil {
		h.logger.Debug("GitHubWebhook invoked")
	}

	if h.dedup != nil {
		if h.dedup.IsDuplicate(eventID) {
			c.JSON(200, gin.H{"status": "duplicate_ignored"})
			return
		}
		h.dedup.TrackEvent(eventID)
	}

	// Parse repository from webhook payload
	var payload struct {
		Repository struct {
			FullName string `json:"full_name"`
			HTMLURL  string `json:"html_url"`
		} `json:"repository"`
	}
	body, err := c.GetRawData()
	if err == nil && len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err == nil && payload.Repository.FullName != "" {
			// Extract owner and name
			owner, name := splitFullName(payload.Repository.FullName)
			repo := models.Repository{
				ID:       payload.Repository.FullName,
				Service:  "github",
				Owner:    owner,
				Name:     name,
				HTTPSURL: payload.Repository.HTMLURL,
			}
			if !h.Queue.TryEnqueue(repo) {
				if h.logger != nil {
					h.logger.Warn("webhook queue full, rejecting repository", slog.String("repo", repo.ID))
				}
				if h.metrics != nil {
					h.metrics.RecordWebhookEvent("github", eventType)
				}
				h.emitAudit(c.Request.Context(), audit.Entry{
					Actor:    "webhook",
					Action:   "webhook.reject",
					Target:   repo.ID,
					Outcome:  "full",
					Metadata: map[string]string{"service": "github", "event": eventType},
				})
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"status": "queue_full", "event": eventType})
				return
			}
			h.emitAudit(c.Request.Context(), audit.Entry{
				Actor:    "webhook",
				Action:   "webhook.enqueue",
				Target:   repo.ID,
				Outcome:  "ok",
				Metadata: map[string]string{"service": "github", "event": eventType},
			})
		}
	}

	if h.metrics != nil {
		h.metrics.RecordWebhookEvent("github", eventType)
	}

	if h.logger != nil {
		h.logger.Info("github webhook received", slog.String("event", eventType), slog.String("delivery", eventID))
	}

	c.JSON(200, gin.H{"status": "queued", "event": eventType})
}

func (h *WebhookHandler) GitLabWebhook(c *gin.Context) {
	eventType := c.GetHeader("X-Gitlab-Event")
	eventID := c.GetHeader("X-Gitlab-Token")

	if h.dedup != nil {
		if h.dedup.IsDuplicate(eventID) {
			c.JSON(200, gin.H{"status": "duplicate_ignored"})
			return
		}
		h.dedup.TrackEvent(eventID)
	}

	// Parse repository from webhook payload
	var payload struct {
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
			WebURL            string `json:"web_url"`
		} `json:"project"`
	}
	body, err := c.GetRawData()
	if err == nil && len(body) > 0 {
		if err := json.Unmarshal(body, &payload); err == nil && payload.Project.PathWithNamespace != "" {
			// Extract owner and name
			owner, name := splitFullName(payload.Project.PathWithNamespace)
			repo := models.Repository{
				ID:       payload.Project.PathWithNamespace,
				Service:  "gitlab",
				Owner:    owner,
				Name:     name,
				HTTPSURL: payload.Project.WebURL,
			}
			if !h.Queue.TryEnqueue(repo) {
				if h.logger != nil {
					h.logger.Warn("webhook queue full, rejecting repository", slog.String("repo", repo.ID))
				}
				if h.metrics != nil {
					h.metrics.RecordWebhookEvent("gitlab", eventType)
				}
				h.emitAudit(c.Request.Context(), audit.Entry{
					Actor:    "webhook",
					Action:   "webhook.reject",
					Target:   repo.ID,
					Outcome:  "full",
					Metadata: map[string]string{"service": "gitlab", "event": eventType},
				})
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"status": "queue_full", "event": eventType})
				return
			}
			h.emitAudit(c.Request.Context(), audit.Entry{
				Actor:    "webhook",
				Action:   "webhook.enqueue",
				Target:   repo.ID,
				Outcome:  "ok",
				Metadata: map[string]string{"service": "gitlab", "event": eventType},
			})
		}
	}

	if h.metrics != nil {
		h.metrics.RecordWebhookEvent("gitlab", eventType)
	}

	if h.logger != nil {
		h.logger.Info("gitlab webhook received", slog.String("event", eventType))
	}

	c.JSON(200, gin.H{"status": "queued", "event": eventType})
}

func (h *WebhookHandler) GenericWebhook(c *gin.Context) {
	service := c.Param("service")

	if h.metrics != nil {
		h.metrics.RecordWebhookEvent(service, "push")
	}

	if h.logger != nil {
		h.logger.Info("webhook received", slog.String("service", service))
	}

	h.emitAudit(c.Request.Context(), audit.Entry{
		Actor:    "webhook",
		Action:   "webhook.enqueue",
		Target:   service,
		Outcome:  "ok",
		Metadata: map[string]string{"service": service, "event": "push"},
	})

	c.JSON(200, gin.H{"status": "queued", "service": service})
}
