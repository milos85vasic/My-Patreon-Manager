# Credential Rotation Runbook

## Overview

My Patreon Manager uses API credentials for multiple external services. This runbook describes the procedure for rotating each credential type. Credential rotation should be performed:

- Immediately upon any suspected or confirmed credential exposure
- On a regular schedule (recommended: every 90 days)
- When team membership changes

## General Procedure

1. Generate a new credential from the provider's dashboard.
2. Update the credential in `.env` or your environment variable management system.
3. Validate the new credential: `go run ./cmd/cli validate`
4. Perform a dry-run sync: `go run ./cmd/cli sync --dry-run`
5. Revoke the old credential from the provider's dashboard.
6. If the old credential was committed to version control, follow the emergency purge procedure below.

---

## Patreon OAuth2 Tokens

### Access Token and Refresh Token

1. Navigate to the [Patreon Developer Portal](https://www.patreon.com/portal/registration/register-clients).
2. Locate your application and generate new OAuth2 credentials.
3. Update `.env`:
   ```
   PATREON_ACCESS_TOKEN=<new_access_token>
   PATREON_REFRESH_TOKEN=<new_refresh_token>
   ```
4. Validate: `go run ./cmd/cli validate`
5. The application supports automatic token refresh -- once the new refresh token is configured, access tokens will be refreshed automatically on expiry.

### Client ID and Client Secret

1. In the Patreon Developer Portal, regenerate the client secret for your application.
2. Update `.env`:
   ```
   PATREON_CLIENT_ID=<new_client_id>
   PATREON_CLIENT_SECRET=<new_client_secret>
   ```
3. Validate: `go run ./cmd/cli validate`

---

## GitHub Token

1. Navigate to [GitHub Settings > Developer settings > Personal access tokens](https://github.com/settings/tokens).
2. Generate a new fine-grained or classic token with the required scopes (`repo`, `read:org`).
3. Update `.env`:
   ```
   GITHUB_TOKEN=<new_token>
   ```
4. If using secondary failover tokens:
   ```
   GITHUB_TOKEN_SECONDARY=<new_secondary_token>
   ```
5. Validate: `go run ./cmd/cli validate`
6. Revoke the old token in GitHub Settings.

---

## GitLab Token

1. Navigate to [GitLab > User Settings > Access Tokens](https://gitlab.com/-/user_settings/personal_access_tokens).
2. Create a new personal access token with `read_api` scope.
3. Update `.env`:
   ```
   GITLAB_TOKEN=<new_token>
   ```
4. If using secondary failover tokens:
   ```
   GITLAB_TOKEN_SECONDARY=<new_secondary_token>
   ```
5. Validate: `go run ./cmd/cli validate`
6. Revoke the old token in GitLab Settings.

---

## GitFlic Token

1. Log in to [GitFlic](https://gitflic.ru) and navigate to your account settings.
2. Generate a new API token.
3. Update `.env`:
   ```
   GITFLIC_TOKEN=<new_token>
   ```
4. If using secondary failover tokens:
   ```
   GITFLIC_TOKEN_SECONDARY=<new_secondary_token>
   ```
5. Validate: `go run ./cmd/cli validate`
6. Revoke the old token.

---

## GitVerse Token

1. Log in to [GitVerse](https://gitverse.ru) and navigate to your account settings.
2. Generate a new API token.
3. Update `.env`:
   ```
   GITVERSE_TOKEN=<new_token>
   ```
4. If using secondary failover tokens:
   ```
   GITVERSE_TOKEN_SECONDARY=<new_secondary_token>
   ```
5. Validate: `go run ./cmd/cli validate`
6. Revoke the old token.

---

## HMAC Secret (Webhook Verification)

1. Generate a new random secret:
   ```sh
   openssl rand -hex 32
   ```
2. Update `.env`:
   ```
   HMAC_SECRET=<new_secret>
   ```
3. Update the webhook secret in any services that send webhooks to My Patreon Manager (e.g., Patreon webhook settings).
4. Validate: `go run ./cmd/cli validate`

---

## Emergency Purge: Credential Committed to Version Control

If a credential is accidentally committed:

1. **Rotate the credential immediately** using the provider-specific steps above.
2. **Purge from git history**:
   ```sh
   # Install git-filter-repo if not present
   pip install git-filter-repo

   # Create a replacements file
   echo "EXPOSED_CREDENTIAL==>***REDACTED***" > /tmp/replace-rules.txt

   # Purge from all history
   git filter-repo --replace-text /tmp/replace-rules.txt --force
   ```
3. **Force-push to all four mirrors**:
   ```sh
   bash Upstreams/GitHub.sh
   bash Upstreams/GitLab.sh
   bash Upstreams/GitFlic.sh
   bash Upstreams/GitVerse.sh
   ```
4. **Notify affected parties** if the credential provided access to shared resources.
5. **Audit access logs** on the affected provider for unauthorized usage during the exposure window.
