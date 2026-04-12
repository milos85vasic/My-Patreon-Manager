# Developer Manual

## Adding a New Git Provider

1. Create `internal/providers/git/newprovider.go` implementing `RepositoryProvider`:
   - `ListRepositories(ctx, org) ([]models.Repository, error)`
   - `GetRepositoryMetadata(ctx, repo) (*models.Repository, error)`
   - `Authenticate(ctx) error`

2. Add token config in `internal/config/config.go`:
   ```go
   NewProviderToken string `env:"NEWPROVIDER_TOKEN"`
   ```

3. Register in `cmd/cli/main.go` provider construction.

4. Add webhook route in `cmd/server/main.go`:
   ```go
   wh.POST("/newprovider", ...)
   ```

5. Write tests using `httptest.NewServer` — never hardcode URLs.

6. Run `bash scripts/coverage.sh` — must stay at 100%.

## Adding a New Renderer

1. Create `internal/providers/renderer/newformat.go` implementing `FormatRenderer`:
   - `Format() string`
   - `SupportedContentTypes() []string`
   - `Render(ctx, title string, vars map[string]interface{}) ([]byte, error)`

2. Add config flag: `NEWFORMAT_RENDERING_ENABLED`

3. Wire in `cmd/cli/renderers.go`:
   ```go
   if cfg.NewFormatRenderingEnabled {
       rs = append(rs, renderer.NewNewFormatRenderer())
   }
   ```

4. Add golden-file tests under `testdata/golden/newformat/`.

## Adding a Migration

1. Create `internal/database/migrations/000N_description.up.sql` and `.down.sql`
2. The app runs migrations on startup when `DATABASE_AUTO_MIGRATE=true`
3. Test with `go test ./internal/database/...`

## Writing Tests

### Test types required
- **Unit**: table-driven, in-package `_test.go`
- **Race**: all tests run under `go test -race`
- **Fuzz**: `Fuzz*` functions in `tests/fuzz/`
- **Integration**: `tests/integration/`
- **goleak**: every package has `TestMain` with `goleak.VerifyTestMain`

### PR checklist
- [ ] Tests pass: `go test -race ./...`
- [ ] Coverage: `bash scripts/coverage.sh`
- [ ] Vet: `go vet ./...`
- [ ] Lint: `golangci-lint run`
- [ ] Docs updated if API changed
- [ ] ADR written if architectural decision made
