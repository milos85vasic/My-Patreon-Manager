package handlers

import (
	"bytes"
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
	metrics := &mockMetricsCollector{}
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	queue := make(chan models.Repository, 1)
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

	// Verify repository was queued
	select {
	case repo := <-queue:
		assert.Equal(t, "owner/repo", repo.ID)
		assert.Equal(t, "github", repo.Service)
		assert.Equal(t, "owner", repo.Owner)
		assert.Equal(t, "repo", repo.Name)
		assert.Equal(t, "https://github.com/owner/repo", repo.HTTPSURL)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("expected repository in queue")
	}
}

func TestGitHubWebhook_QueueFull(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
	metrics := &mockMetricsCollector{}
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := NewWebhookHandler(dedup, metrics, logger)
	queue := make(chan models.Repository, 1)
	handler.Queue = queue
	// Fill queue
	queue <- models.Repository{ID: "test"}
	router.POST("/webhook/github", handler.GitHubWebhook)

	payload := `{"repository":{"full_name":"owner/repo","html_url":"https://github.com/owner/repo"}}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/webhook/github", bytes.NewBufferString(payload))
	req.Header.Set("X-GitHub-Delivery", "999")
	req.Header.Set("X-GitHub-Event", "push")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"queued","event":"push"}`, w.Body.String())
	// Verify warning was logged
	assert.Contains(t, buf.String(), "webhook queue full")
}

func TestGitLabWebhook(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	dedup := sync.NewEventDeduplicator(5 * time.Minute)
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
