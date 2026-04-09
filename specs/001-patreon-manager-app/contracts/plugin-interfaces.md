# Plugin Interface Contracts

Go interfaces that all pluggable modules must implement. These are the
contracts between the core sync pipeline and external service adapters.

## RepositoryProvider

Implemented by each Git service adapter (GitHub, GitLab, GitFlic, GitVerse).

```go
type RepositoryProvider interface {
    Name() string
    Authenticate(ctx context.Context, credentials Credentials) error
    ListRepositories(ctx context.Context, org string, opts ListOptions) ([]Repository, error)
    GetRepositoryMetadata(ctx context.Context, repo Repository) (Metadata, error)
    DetectMirrors(ctx context.Context, repos []Repository) (MirrorMap, error)
    CheckRepositoryState(ctx context.Context, repo Repository) (State, error)
}
```

**Methods**:

| Method | Input | Output | Errors |
|--------|-------|--------|--------|
| `Name` | none | service name string | none |
| `Authenticate` | credentials | none | `ErrInvalidCredentials`, `ErrNetworkTimeout` |
| `ListRepositories` | org name, pagination opts | slice of Repository | `ErrRateLimited`, `ErrPermissionDenied` |
| `GetRepositoryMetadata` | Repository | Metadata struct | `ErrNotFound`, `ErrRateLimited` |
| `DetectMirrors` | slice of Repository | MirrorMap | none (best-effort) |
| `CheckRepositoryState` | Repository | State struct | `ErrNotFound`, `ErrNetworkTimeout` |

**Error types**: All providers return typed errors implementing
`ProviderError` interface with `Code()`, `Retryable() bool`, and
`RateLimitReset() time.Time`.

## LLMProvider

Implemented by the LLMsVerifier client adapter.

```go
type LLMProvider interface {
    GenerateContent(ctx context.Context, prompt Prompt, opts GenerationOptions) (Content, error)
    GetAvailableModels(ctx context.Context) ([]ModelInfo, error)
    GetModelQualityScore(ctx context.Context, modelID string) (float64, error)
    GetTokenUsage(ctx context.Context) (UsageStats, error)
}
```

**Structs**:

```go
type Prompt struct {
    TemplateName string
    Variables    map[string]string
    ContentType  string
}

type GenerationOptions struct {
    ModelID       string
    MaxTokens     int
    QualityTier   string
    Timeout       time.Duration
    FallbackChain []string
}

type Content struct {
    Title        string
    Body         string
    QualityScore float64
    ModelUsed    string
    TokenCount   int
}

type ModelInfo struct {
    ID           string
    Name         string
    QualityScore float64
    LatencyP95   time.Duration
    CostPer1KTok float64
}

type UsageStats struct {
    TotalTokens    int64
    EstimatedCost  float64
    BudgetLimit    float64
    BudgetUsedPct  float64
}
```

## FormatRenderer

Implemented by each output format (Markdown, HTML, PDF, Video).

```go
type FormatRenderer interface {
    Format() string
    Render(ctx context.Context, content Content, opts RenderOptions) ([]byte, error)
    SupportedContentTypes() []string
}
```

**Methods**:

| Method | Input | Output | Errors |
|--------|-------|--------|--------|
| `Format` | none | format name string | none |
| `Render` | Content + options | rendered bytes | `ErrRenderingFailed`, `ErrTimeout` |
| `SupportedContentTypes` | none | slice of content types | none |

## Database

Implemented by SQLite and PostgreSQL backends.

```go
type Database interface {
    Connect(ctx context.Context, dsn string) error
    Close() error
    Migrate(ctx context.Context) error
    
    Repositories() RepositoryStore
    SyncStates() SyncStateStore
    MirrorMaps() MirrorMapStore
    GeneratedContents() GeneratedContentStore
    Posts() PostStore
    AuditEntries() AuditEntryStore
    
    AcquireLock(ctx context.Context, lockInfo SyncLock) error
    ReleaseLock(ctx context.Context) error
    IsLocked(ctx context.Context) (bool, *SyncLock, error)
    
    BeginTx(ctx context.Context) (*sql.Tx, error)
}
```

Each sub-store follows standard CRUD patterns:
- `Create(ctx, entity) error`
- `GetByID(ctx, id) (entity, error)`
- `List(ctx, filter) ([]entity, error)`
- `Update(ctx, entity) error`
- `Delete(ctx, id) error`

## CircuitBreaker

Wraps all external service calls.

```go
type CircuitBreaker interface {
    Execute(fn func() (interface{}, error)) (interface{}, error)
    State() CircuitState
    HalfOpen() bool
    Trip()
    Reset()
}

type CircuitState int

const (
    CircuitClosed   CircuitState = iota
    CircuitOpen
    CircuitHalfOpen
)
```

**Configuration**: Failure threshold, cooldown duration, half-open probe count.
