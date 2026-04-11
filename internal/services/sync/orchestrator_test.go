package sync

import (
	"context"
	"log/slog"
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/git"
	"github.com/milos85vasic/My-Patreon-Manager/internal/providers/renderer"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/content"
	"github.com/milos85vasic/My-Patreon-Manager/tests/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockMetricsCollector struct{}

func (m *mockMetricsCollector) RecordWebhookEvent(service, eventType string)               {}
func (m *mockMetricsCollector) RecordSyncDuration(service, status string, seconds float64) {}
func (m *mockMetricsCollector) RecordReposProcessed(service, action string)                {}
func (m *mockMetricsCollector) RecordAPIError(service, errorType string)                   {}
func (m *mockMetricsCollector) RecordLLMLatency(model string, seconds float64)             {}
func (m *mockMetricsCollector) RecordLLMTokens(model, tokenType string, count int)         {}
func (m *mockMetricsCollector) RecordLLMQualityScore(repository string, score float64)     {}
func (m *mockMetricsCollector) RecordContentGenerated(format, qualityTier string)          {}
func (m *mockMetricsCollector) RecordPostCreated(tier string)                              {}
func (m *mockMetricsCollector) RecordPostUpdated(tier string)                              {}
func (m *mockMetricsCollector) SetActiveSyncs(count int)                                   {}
func (m *mockMetricsCollector) SetBudgetUtilization(percent float64)                       {}

type mockGenerator struct{}

func (m *mockGenerator) GenerateForRepository(ctx context.Context, repo models.Repository, templates []models.ContentTemplate, mirrorURLs []renderer.MirrorURL) (*models.GeneratedContent, error) {
	return nil, nil
}

func (m *mockGenerator) SetReviewQueue(rq *content.ReviewQueue) {}

func TestOrchestrator_Run_NoRepos(t *testing.T) {
	ctx := context.Background()
	db := &mocks.MockDatabase{}
	providers := []git.RepositoryProvider{}
	patreon := &mocks.PatreonClient{}
	metrics := &mockMetricsCollector{}
	logger := slog.Default()

	orc := NewOrchestrator(db, providers, patreon, nil, metrics, logger, nil)
	result, err := orc.Run(ctx, SyncOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Processed)
	assert.Equal(t, 0, result.Failed)
	assert.Equal(t, 0, result.Skipped)
	assert.Len(t, result.Errors, 0)
}
