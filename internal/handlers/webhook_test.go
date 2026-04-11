package handlers

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/milos85vasic/My-Patreon-Manager/internal/metrics"
	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMetricsCollector implements metrics.MetricsCollector for testing
type mockMetricsCollector struct{}

var _ metrics.MetricsCollector = (*mockMetricsCollector)(nil)

func (m *mockMetricsCollector) RecordSyncDuration(service string, status string, seconds float64) {}
func (m *mockMetricsCollector) RecordReposProcessed(service, action string)                       {}
func (m *mockMetricsCollector) RecordAPIError(service, errorType string)                          {}
func (m *mockMetricsCollector) RecordLLMLatency(model string, seconds float64)                    {}
func (m *mockMetricsCollector) RecordLLMTokens(model, tokenType string, count int)                {}
func (m *mockMetricsCollector) RecordLLMQualityScore(repository string, score float64)            {}
func (m *mockMetricsCollector) RecordContentGenerated(format, qualityTier string)                 {}
func (m *mockMetricsCollector) RecordPostCreated(tier string)                                     {}
func (m *mockMetricsCollector) RecordPostUpdated(tier string)                                     {}
func (m *mockMetricsCollector) RecordWebhookEvent(service, eventType string)                      {}
func (m *mockMetricsCollector) SetActiveSyncs(count int)                                          {}
func (m *mockMetricsCollector) SetBudgetUtilization(percent float64)                              {}

func TestGitHubWebhook_DuplicateIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	router.POST("/webhook/github", handler.GitHubWebhook)

	// First request
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/webhook/github", nil)
	req1.Header.Set("X-GitHub-Delivery", "123")
	req1.Header.Set("X-GitHub-Event", "push")
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.JSONEq(t, `{"status":"queued","event":"push"}`, w1.Body.String())

	// Second request with same delivery ID (duplicate)
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/webhook/github", nil)
	req2.Header.Set("X-GitHub-Delivery", "123")
	req2.Header.Set("X-GitHub-Event", "push")
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.JSONEq(t, `{"status":"duplicate_ignored"}`, w2.Body.String())
}

func TestGitHubWebhook_ValidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	router.POST("/webhook/github", handler.GitHubWebhook)

	payload := `{"repository":{"full_name":"owner/repo","html_url":"https://github.com/owner/repo"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/webhook/github", bytes.NewBufferString(payload))
	req.Header.Set("X-GitHub-Delivery", "456")
	req.Header.Set("X-GitHub-Event", "push")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"queued","event":"push"}`, w.Body.String())
}

func TestGitHubWebhook_Queue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	queue := NewWebhookQueue[models.Repository](4)
	handler.Queue = queue
	router.POST("/webhook/github", handler.GitHubWebhook)

	payload := `{"repository":{"full_name":"owner/repo","html_url":"https://github.com/owner/repo"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/webhook/github", bytes.NewBufferString(payload))
	req.Header.Set("X-GitHub-Delivery", "789")
	req.Header.Set("X-GitHub-Event", "push")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"queued","event":"push"}`, w.Body.String())

	// Drain the repository back out and verify it was enqueued intact.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	var seen models.Repository
	_ = queue.Drain(ctx, func(r models.Repository) error {
		seen = r
		cancel()
		return nil
	})
	assert.Equal(t, "owner/repo", seen.ID)
	assert.Equal(t, "github", seen.Service)
	assert.Equal(t, "owner", seen.Owner)
	assert.Equal(t, "repo", seen.Name)
	assert.Equal(t, "https://github.com/owner/repo", seen.HTTPSURL)
}

func TestGitHubWebhook_QueueFull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	queue := NewWebhookQueue[models.Repository](1)
	handler.Queue = queue
	// Fill queue so the next enqueue must be rejected.
	require.True(t, queue.TryEnqueue(models.Repository{ID: "test"}))
	router.POST("/webhook/github", handler.GitHubWebhook)

	payload := `{"repository":{"full_name":"owner/repo","html_url":"https://github.com/owner/repo"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/webhook/github", bytes.NewBufferString(payload))
	req.Header.Set("X-GitHub-Delivery", "999")
	req.Header.Set("X-GitHub-Event", "push")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.JSONEq(t, `{"status":"queue_full","event":"push"}`, w.Body.String())
	assert.Contains(t, buf.String(), "webhook queue full")
}

func TestGitHubWebhook_DefaultQueueNonNil(t *testing.T) {
	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	if handler.Queue == nil {
		t.Fatal("Queue must be non-nil by default")
	}
	if handler.Queue.Cap() != DefaultWebhookQueueCapacity {
		t.Fatalf("default cap = %d, want %d", handler.Queue.Cap(), DefaultWebhookQueueCapacity)
	}
}

func TestGitLabWebhook_QueueFull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	queue := NewWebhookQueue[models.Repository](1)
	handler.Queue = queue
	require.True(t, queue.TryEnqueue(models.Repository{ID: "filler"}))
	router.POST("/webhook/gitlab", handler.GitLabWebhook)

	payload := `{"project":{"path_with_namespace":"group/project","web_url":"https://gitlab.com/group/project"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/webhook/gitlab", bytes.NewBufferString(payload))
	req.Header.Set("X-Gitlab-Event", "Push Hook")
	req.Header.Set("X-Gitlab-Token", "full-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.JSONEq(t, `{"status":"queue_full","event":"Push Hook"}`, w.Body.String())
	assert.Contains(t, buf.String(), "webhook queue full")
}

func TestGitLabWebhook_Queue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	queue := NewWebhookQueue[models.Repository](4)
	handler.Queue = queue
	router.POST("/webhook/gitlab", handler.GitLabWebhook)

	payload := `{"project":{"path_with_namespace":"group/project","web_url":"https://gitlab.com/group/project"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/webhook/gitlab", bytes.NewBufferString(payload))
	req.Header.Set("X-Gitlab-Event", "Push Hook")
	req.Header.Set("X-Gitlab-Token", "ok-token")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	var seen models.Repository
	_ = queue.Drain(ctx, func(r models.Repository) error {
		seen = r
		cancel()
		return nil
	})
	assert.Equal(t, "group/project", seen.ID)
	assert.Equal(t, "gitlab", seen.Service)
	assert.Equal(t, "group", seen.Owner)
	assert.Equal(t, "project", seen.Name)
	assert.Equal(t, "https://gitlab.com/group/project", seen.HTTPSURL)
}

func TestGitLabWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	router.POST("/webhook/gitlab", handler.GitLabWebhook)

	// Test duplicate detection
	eventID := "token123"
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/webhook/gitlab", nil)
	req1.Header.Set("X-Gitlab-Event", "Push Hook")
	req1.Header.Set("X-Gitlab-Token", eventID)
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	assert.JSONEq(t, `{"status":"queued","event":"Push Hook"}`, w1.Body.String())

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/webhook/gitlab", nil)
	req2.Header.Set("X-Gitlab-Event", "Push Hook")
	req2.Header.Set("X-Gitlab-Token", eventID)
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	assert.JSONEq(t, `{"status":"duplicate_ignored"}`, w2.Body.String())
}

func TestGenericWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	t.Cleanup(func() { _ = dedup.Close() })
	metrics := &mockMetricsCollector{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	router.POST("/webhook/:service", handler.GenericWebhook)

	// Test unknown service
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/webhook/unknown", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"queued","service":"unknown"}`, w.Body.String())
}
