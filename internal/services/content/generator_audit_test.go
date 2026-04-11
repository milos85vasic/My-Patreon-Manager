package content

import (
	"context"
	"errors"
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	"github.com/milos85vasic/My-Patreon-Manager/internal/services/audit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticLLM returns the same content for any call.
type staticLLM struct{ c models.Content }

func (s *staticLLM) GenerateContent(_ context.Context, _ models.Prompt, _ models.GenerationOptions) (models.Content, error) {
	return s.c, nil
}
func (s *staticLLM) GetAvailableModels(_ context.Context) ([]models.ModelInfo, error) {
	return nil, nil
}
func (s *staticLLM) GetModelQualityScore(_ context.Context, _ string) (float64, error) {
	return 0, nil
}
func (s *staticLLM) GetTokenUsage(_ context.Context) (models.UsageStats, error) {
	return models.UsageStats{}, nil
}

// failingLLMNamed always errors.
type failingLLMNamed struct{}

func (f *failingLLMNamed) GenerateContent(_ context.Context, _ models.Prompt, _ models.GenerationOptions) (models.Content, error) {
	return models.Content{}, errors.New("nope")
}
func (f *failingLLMNamed) GetAvailableModels(_ context.Context) ([]models.ModelInfo, error) {
	return nil, nil
}
func (f *failingLLMNamed) GetModelQualityScore(_ context.Context, _ string) (float64, error) {
	return 0, nil
}
func (f *failingLLMNamed) GetTokenUsage(_ context.Context) (models.UsageStats, error) {
	return models.UsageStats{}, nil
}

// failingStore makes generated-content persistence fail to exercise the
// store-error audit branch.
type failingStore struct{}

func (failingStore) Create(_ context.Context, _ *models.GeneratedContent) error {
	return errors.New("store boom")
}
func (failingStore) GetByID(_ context.Context, _ string) (*models.GeneratedContent, error) {
	return nil, nil
}
func (failingStore) GetLatestByRepo(_ context.Context, _ string) (*models.GeneratedContent, error) {
	return nil, nil
}
func (failingStore) GetByQualityRange(_ context.Context, _, _ float64) ([]*models.GeneratedContent, error) {
	return nil, nil
}
func (failingStore) ListByRepository(_ context.Context, _ string) ([]*models.GeneratedContent, error) {
	return nil, nil
}
func (failingStore) Update(_ context.Context, _ *models.GeneratedContent) error { return nil }

func newRepo(id string) models.Repository {
	return models.Repository{ID: id, Owner: "o", Name: "r", HTTPSURL: "https://x"}
}

func TestGenerator_GenerateForRepository_AuditOK(t *testing.T) {
	llm := &staticLLM{c: models.Content{Title: "T", Body: "B", QualityScore: 0.9, ModelUsed: "m", TokenCount: 10}}
	gen := NewGenerator(llm, NewTokenBudget(100000), NewQualityGate(0.5), nil, nil, nil)

	_, err := gen.GenerateForRepository(context.Background(), newRepo("ok"), nil, nil)
	require.NoError(t, err)

	entries, err := gen.AuditStore().List(context.Background(), 10)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "content", entries[0].Actor)
	assert.Equal(t, "content.generate", entries[0].Action)
	assert.Equal(t, "ok", entries[0].Outcome)
	assert.Equal(t, "ok", entries[0].Target)
	assert.False(t, entries[0].CreatedAt.IsZero())
}

func TestGenerator_GenerateForRepository_AuditRejected(t *testing.T) {
	llm := &staticLLM{c: models.Content{Title: "T", Body: "B", QualityScore: 0.1, ModelUsed: "m", TokenCount: 10}}
	gen := NewGenerator(llm, NewTokenBudget(100000), NewQualityGate(0.9), nil, nil, nil)

	_, err := gen.GenerateForRepository(context.Background(), newRepo("rej"), nil, nil)
	require.NoError(t, err)

	entries, _ := gen.AuditStore().List(context.Background(), 10)
	require.Len(t, entries, 1)
	assert.Equal(t, "rejected", entries[0].Outcome)
}

func TestGenerator_GenerateForRepository_AuditBudgetExhausted(t *testing.T) {
	llm := &staticLLM{c: models.Content{}}
	budget := NewTokenBudget(1) // very small budget
	gen := NewGenerator(llm, budget, NewQualityGate(0.5), nil, nil, nil)

	_, err := gen.GenerateForRepository(context.Background(), newRepo("budget"), nil, nil)
	require.Error(t, err)

	entries, _ := gen.AuditStore().List(context.Background(), 10)
	require.Len(t, entries, 1)
	assert.Equal(t, "error", entries[0].Outcome)
	assert.Equal(t, "budget exhausted", entries[0].Metadata["error"])
}

func TestGenerator_GenerateForRepository_AuditLLMRetriesExhausted(t *testing.T) {
	gen := NewGenerator(&failingLLMNamed{}, NewTokenBudget(100000), NewQualityGate(0.5), nil, nil, nil)

	_, err := gen.GenerateForRepository(context.Background(), newRepo("llm"), nil, nil)
	require.Error(t, err)

	entries, _ := gen.AuditStore().List(context.Background(), 10)
	require.Len(t, entries, 1)
	assert.Equal(t, "error", entries[0].Outcome)
	assert.Equal(t, "llm exhausted retries", entries[0].Metadata["error"])
}

func TestGenerator_GenerateForRepository_AuditContextCancelledImmediate(t *testing.T) {
	gen := NewGenerator(&failingLLMNamed{}, NewTokenBudget(100000), NewQualityGate(0.5), nil, nil, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before call so the first ctx.Err() check fires after the first failure

	_, err := gen.GenerateForRepository(ctx, newRepo("ctx"), nil, nil)
	require.Error(t, err)

	entries, _ := gen.AuditStore().List(context.Background(), 10)
	require.Len(t, entries, 1)
	assert.Equal(t, "error", entries[0].Outcome)
	assert.Equal(t, "context cancelled", entries[0].Metadata["error"])
}

func TestGenerator_GenerateForRepository_AuditStoreFailure(t *testing.T) {
	llm := &staticLLM{c: models.Content{Title: "T", Body: "B", QualityScore: 0.9, ModelUsed: "m", TokenCount: 10}}
	gen := NewGenerator(llm, NewTokenBudget(100000), NewQualityGate(0.5), failingStore{}, nil, nil)

	_, err := gen.GenerateForRepository(context.Background(), newRepo("store"), nil, nil)
	require.Error(t, err)

	entries, _ := gen.AuditStore().List(context.Background(), 10)
	require.Len(t, entries, 1)
	assert.Equal(t, "error", entries[0].Outcome)
	assert.Equal(t, "store failed", entries[0].Metadata["error"])
}

func TestGenerator_SetAuditStore(t *testing.T) {
	gen := NewGenerator(&staticLLM{}, NewTokenBudget(100), NewQualityGate(0.5), nil, nil, nil)
	custom := audit.NewRingStore(8)
	gen.SetAuditStore(custom)
	assert.Same(t, custom, gen.AuditStore())

	gen.SetAuditStore(nil)
	assert.NotNil(t, gen.AuditStore())
	assert.NotSame(t, custom, gen.AuditStore())
}
