// Package git implements the RepositoryProvider interface for GitHub, GitLab,
// GitFlic, and GitVerse. Each provider adapter handles service-specific
// authentication, pagination, rate limiting, mirror detection, and .repoignore
// filtering behind a unified abstraction.
package git
