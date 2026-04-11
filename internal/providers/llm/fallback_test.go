package llm

import (
	"context"
	"errors"
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockProvider struct {
	generateContentFunc      func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error)
	getAvailableModelsFunc   func(ctx context.Context) ([]models.ModelInfo, error)
	getModelQualityScoreFunc func(ctx context.Context, modelID string) (float64, error)
	getTokenUsageFunc        func(ctx context.Context) (models.UsageStats, error)
}

func (m *mockProvider) GenerateContent(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
	if m.generateContentFunc != nil {
		return m.generateContentFunc(ctx, prompt, opts)
	}
	return models.Content{}, errors.New("not implemented")
}

func (m *mockProvider) GetAvailableModels(ctx context.Context) ([]models.ModelInfo, error) {
	if m.getAvailableModelsFunc != nil {
		return m.getAvailableModelsFunc(ctx)
	}
	return nil, errors.New("not implemented")
}

func (m *mockProvider) GetModelQualityScore(ctx context.Context, modelID string) (float64, error) {
	if m.getModelQualityScoreFunc != nil {
		return m.getModelQualityScoreFunc(ctx, modelID)
	}
	return 0, errors.New("not implemented")
}

func (m *mockProvider) GetTokenUsage(ctx context.Context) (models.UsageStats, error) {
	if m.getTokenUsageFunc != nil {
		return m.getTokenUsageFunc(ctx)
	}
	return models.UsageStats{}, errors.New("not implemented")
}

func TestFallbackChain_GenerateContent_SuccessFirstProvider(t *testing.T) {
	provider1 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			return models.Content{
				Title:        "Title from provider 1",
				Body:         "Body from provider 1",
				QualityScore: 0.95,
				ModelUsed:    "model-1",
				TokenCount:   100,
			}, nil
		},
	}
	provider2 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			return models.Content{}, errors.New("should not be called")
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	prompt := models.Prompt{}
	opts := models.GenerationOptions{}
	ctx := context.Background()

	content, err := fc.GenerateContent(ctx, prompt, opts)
	require.NoError(t, err)
	assert.Equal(t, "Title from provider 1", content.Title)
	assert.Equal(t, "Body from provider 1", content.Body)
	assert.Equal(t, 0.95, content.QualityScore)
	assert.Equal(t, "model-1", content.ModelUsed)
	assert.Equal(t, 100, content.TokenCount)
}

func TestFallbackChain_GenerateContent_FallbackOnError(t *testing.T) {
	callCount := 0
	provider1 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			callCount++
			return models.Content{}, errors.New("provider 1 failed")
		},
	}
	provider2 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			callCount++
			return models.Content{
				Title:        "Title from provider 2",
				Body:         "Body from provider 2",
				QualityScore: 0.90,
				ModelUsed:    "model-2",
				TokenCount:   120,
			}, nil
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	prompt := models.Prompt{}
	opts := models.GenerationOptions{}
	ctx := context.Background()

	content, err := fc.GenerateContent(ctx, prompt, opts)
	require.NoError(t, err)
	assert.Equal(t, "Title from provider 2", content.Title)
	assert.Equal(t, "Body from provider 2", content.Body)
	assert.Equal(t, 0.90, content.QualityScore)
	assert.Equal(t, "model-2", content.ModelUsed)
	assert.Equal(t, 120, content.TokenCount)
	assert.Equal(t, 2, callCount) // both providers called
}

func TestFallbackChain_GenerateContent_FallbackOnLowQuality(t *testing.T) {
	callCount := 0
	provider1 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			callCount++
			return models.Content{
				Title:        "Low quality title",
				Body:         "Low quality body",
				QualityScore: 0.70, // below threshold 0.8
				ModelUsed:    "model-1",
				TokenCount:   80,
			}, nil
		},
	}
	provider2 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			callCount++
			return models.Content{
				Title:        "High quality title",
				Body:         "High quality body",
				QualityScore: 0.95,
				ModelUsed:    "model-2",
				TokenCount:   150,
			}, nil
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	prompt := models.Prompt{}
	opts := models.GenerationOptions{}
	ctx := context.Background()

	content, err := fc.GenerateContent(ctx, prompt, opts)
	require.NoError(t, err)
	assert.Equal(t, "High quality title", content.Title)
	assert.Equal(t, "High quality body", content.Body)
	assert.Equal(t, 0.95, content.QualityScore)
	assert.Equal(t, "model-2", content.ModelUsed)
	assert.Equal(t, 150, content.TokenCount)
	assert.Equal(t, 2, callCount)
}

func TestFallbackChain_GenerateContent_AllFailReturnBestBelowThreshold(t *testing.T) {
	callCount := 0
	provider1 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			callCount++
			return models.Content{
				Title:        "Low quality title 1",
				Body:         "Low quality body 1",
				QualityScore: 0.60,
				ModelUsed:    "model-1",
				TokenCount:   70,
			}, nil
		},
	}
	provider2 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			callCount++
			return models.Content{
				Title:        "Low quality title 2",
				Body:         "Low quality body 2",
				QualityScore: 0.75, // higher but still below threshold
				ModelUsed:    "model-2",
				TokenCount:   90,
			}, nil
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	prompt := models.Prompt{}
	opts := models.GenerationOptions{}
	ctx := context.Background()

	content, err := fc.GenerateContent(ctx, prompt, opts)
	require.NoError(t, err)
	// Should return the best content (highest score) even below threshold
	assert.Equal(t, "Low quality title 2", content.Title)
	assert.Equal(t, "Low quality body 2", content.Body)
	assert.Equal(t, 0.75, content.QualityScore)
	assert.Equal(t, "model-2", content.ModelUsed)
	assert.Equal(t, 90, content.TokenCount)
	assert.Equal(t, 2, callCount)
}

func TestFallbackChain_GenerateContent_AllProvidersFail(t *testing.T) {
	provider1 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			return models.Content{}, errors.New("provider 1 error")
		},
	}
	provider2 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			return models.Content{}, errors.New("provider 2 error")
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	prompt := models.Prompt{}
	opts := models.GenerationOptions{}
	ctx := context.Background()

	_, err := fc.GenerateContent(ctx, prompt, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "all providers failed")
}

func TestFallbackChain_GenerateContent_CircuitOpenSkipsProvider(t *testing.T) {
	callCount := 0
	provider1 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			callCount++
			return models.Content{}, errors.New("provider 1 failed")
		},
	}
	provider2 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			callCount++
			return models.Content{
				Title:        "Title from provider 2",
				Body:         "Body from provider 2",
				QualityScore: 0.95,
				ModelUsed:    "model-2",
				TokenCount:   100,
			}, nil
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	// Trip circuit breaker for provider1 by making it fail multiple times
	for i := 0; i < 5; i++ {
		_, _ = fc.GenerateContent(context.Background(), models.Prompt{}, models.GenerationOptions{})
	}
	// Reset callCount to track new calls
	callCount = 0
	// Now provider1's breaker should be open, so only provider2 should be called
	content, err := fc.GenerateContent(context.Background(), models.Prompt{}, models.GenerationOptions{})
	require.NoError(t, err)
	assert.Equal(t, "Title from provider 2", content.Title)
	assert.Equal(t, 1, callCount) // only provider2 called
}

func TestFallbackChain_GetAvailableModels_Success(t *testing.T) {
	provider1 := &mockProvider{
		getAvailableModelsFunc: func(ctx context.Context) ([]models.ModelInfo, error) {
			return []models.ModelInfo{
				{ID: "model-1", Name: "Model One"},
			}, nil
		},
	}
	provider2 := &mockProvider{
		getAvailableModelsFunc: func(ctx context.Context) ([]models.ModelInfo, error) {
			return []models.ModelInfo{
				{ID: "model-2", Name: "Model Two"},
			}, nil
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	ctx := context.Background()
	models, err := fc.GetAvailableModels(ctx)
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "model-1", models[0].ID)
	assert.Equal(t, "Model One", models[0].Name)
}

func TestFallbackChain_GetAvailableModels_Fallback(t *testing.T) {
	provider1 := &mockProvider{
		getAvailableModelsFunc: func(ctx context.Context) ([]models.ModelInfo, error) {
			return nil, errors.New("provider 1 error")
		},
	}
	provider2 := &mockProvider{
		getAvailableModelsFunc: func(ctx context.Context) ([]models.ModelInfo, error) {
			return []models.ModelInfo{
				{ID: "model-2", Name: "Model Two"},
			}, nil
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	ctx := context.Background()
	models, err := fc.GetAvailableModels(ctx)
	require.NoError(t, err)
	require.Len(t, models, 1)
	assert.Equal(t, "model-2", models[0].ID)
	assert.Equal(t, "Model Two", models[0].Name)
}

func TestFallbackChain_GetModelQualityScore_Success(t *testing.T) {
	provider1 := &mockProvider{
		getModelQualityScoreFunc: func(ctx context.Context, modelID string) (float64, error) {
			return 0.85, nil
		},
	}
	provider2 := &mockProvider{
		getModelQualityScoreFunc: func(ctx context.Context, modelID string) (float64, error) {
			return 0.92, nil
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	ctx := context.Background()
	score, err := fc.GetModelQualityScore(ctx, "model-123")
	require.NoError(t, err)
	assert.Equal(t, 0.85, score)
}

func TestFallbackChain_GetModelQualityScore_Fallback(t *testing.T) {
	provider1 := &mockProvider{
		getModelQualityScoreFunc: func(ctx context.Context, modelID string) (float64, error) {
			return 0, errors.New("provider 1 error")
		},
	}
	provider2 := &mockProvider{
		getModelQualityScoreFunc: func(ctx context.Context, modelID string) (float64, error) {
			return 0.92, nil
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	ctx := context.Background()
	score, err := fc.GetModelQualityScore(ctx, "model-123")
	require.NoError(t, err)
	assert.Equal(t, 0.92, score)
}

func TestFallbackChain_GetTokenUsage_UsesFirstProvider(t *testing.T) {
	provider1 := &mockProvider{
		getTokenUsageFunc: func(ctx context.Context) (models.UsageStats, error) {
			return models.UsageStats{
				TotalTokens:   1000,
				EstimatedCost: 5.0,
				BudgetLimit:   100.0,
				BudgetUsedPct: 5.0,
			}, nil
		},
	}
	provider2 := &mockProvider{
		getTokenUsageFunc: func(ctx context.Context) (models.UsageStats, error) {
			return models.UsageStats{}, errors.New("should not be called")
		},
	}

	fc := NewFallbackChain([]LLMProvider{provider1, provider2}, 0.8, nil)
	ctx := context.Background()
	stats, err := fc.GetTokenUsage(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), stats.TotalTokens)
	assert.Equal(t, 5.0, stats.EstimatedCost)
	assert.Equal(t, 100.0, stats.BudgetLimit)
	assert.Equal(t, 5.0, stats.BudgetUsedPct)
}

func TestFallbackChain_GetTokenUsage_NoProviders(t *testing.T) {
	fc := NewFallbackChain([]LLMProvider{}, 0.8, nil)
	ctx := context.Background()
	_, err := fc.GetTokenUsage(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no providers available")
}

func TestFallbackChain_MetricsRecording(t *testing.T) {
	mockMetrics := &mockMetricsCollector{}
	provider1 := &mockProvider{
		generateContentFunc: func(ctx context.Context, prompt models.Prompt, opts models.GenerationOptions) (models.Content, error) {
			return models.Content{
				Title:        "Title",
				Body:         "Body",
				QualityScore: 0.75,
				ModelUsed:    "model-1",
				TokenCount:   100,
			}, nil
		},
	}
	fc := NewFallbackChain([]LLMProvider{provider1}, 0.8, mockMetrics)
	ctx := context.Background()
	_, err := fc.GenerateContent(ctx, models.Prompt{}, models.GenerationOptions{})
	require.NoError(t, err)
	assert.Len(t, mockMetrics.qualityScoreCalls, 1)
	assert.Equal(t, 0.75, mockMetrics.qualityScoreCalls[0].score)
}
