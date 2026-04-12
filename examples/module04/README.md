# Module 04 Exercise: Content Generation

1. Set `CONTENT_QUALITY_THRESHOLD=0.5` in `.env`
2. Run `./patreon-manager generate --dry-run --repo your-org/your-repo`
3. Increase threshold to `0.9` and re-run — observe rejections
4. Try `--format=pdf` (requires `PDF_RENDERING_ENABLED=true`)
