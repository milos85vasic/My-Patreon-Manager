package utils

import (
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentHash(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string // we can't know exact hash, but we can verify consistency
	}{
		{"empty string", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"hello world", "hello world", "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"},
		{"multiline", "line1\nline2", "683376e290829b482c2655745caffa7a1dccfa10afaa62dac2b42dd6c68d0f83"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := ContentHash(tt.input)
			assert.Equal(t, tt.expected, hash)
		})
	}
}

func TestREADMEHash(t *testing.T) {
	hash := READMEHash("README content")
	assert.Len(t, hash, 64)
	// deterministic
	assert.Equal(t, READMEHash("README content"), hash)
}

func TestSignURLAndVerify(t *testing.T) {
	secret := "test-secret"
	contentID := "content-123"
	subscriberID := "user-456"
	ttl := 5 * time.Minute

	token, err := SignURL(contentID, subscriberID, secret, ttl)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// verify
	cid, sid, err := VerifySignedURL(token, secret)
	require.NoError(t, err)
	assert.Equal(t, contentID, cid)
	assert.Equal(t, subscriberID, sid)

	// wrong secret
	_, _, err = VerifySignedURL(token, "wrong-secret")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid signature")

	// expired token (create token with negative ttl)
	tokenExp, err := SignURL(contentID, subscriberID, secret, -time.Minute)
	require.NoError(t, err)
	_, _, err = VerifySignedURL(tokenExp, secret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token expired")

	// malformed token
	_, _, err = VerifySignedURL("bad:token", secret)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid token format")
}

func TestToJSON(t *testing.T) {
	data := map[string]interface{}{
		"name": "test",
		"num":  42,
	}
	jsonStr, err := ToJSON(data)
	require.NoError(t, err)
	assert.Contains(t, jsonStr, `"name":"test"`)
	assert.Contains(t, jsonStr, `"num":42`)
}

func TestFromJSON(t *testing.T) {
	jsonStr := `{"name":"test","num":42}`
	var result map[string]interface{}
	err := FromJSON(jsonStr, &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result["name"])
	assert.Equal(t, float64(42), result["num"])

	// empty string returns nil error
	err = FromJSON("", &result)
	assert.NoError(t, err)
}

func TestNormalizeToSSH(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://github.com/owner/repo.git", "git@github.com:owner/repo.git"},
		{"https://github.com/owner/repo", "git@github.com:owner/repo.git"},
		{"ssh://git@github.com/owner/repo.git", "git@github.com:owner/repo.git"},
		{"git@github.com:owner/repo.git", "git@github.com:owner/repo.git"},
		{"git@github.com:owner/repo", "git@github.com:owner/repo.git"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeToSSH(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNormalizeHTTPS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"git@github.com:owner/repo.git", "https://github.com/owner/repo.git"},
		{"ssh://git@github.com/owner/repo.git", "https://github.com/owner/repo.git"},
		{"https://github.com/owner/repo.git", "https://github.com/owner/repo.git"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeHTTPS(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestRedactString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"token=abc123", "token=******"},
		{"secret: xyz", "secret: ***"},
		{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9", "Bearer ************************************"},
		{"ghp_abcdefghijklmnopqrstuvwxyz1234567890", "****************************************"},
		{"glpat-abcdefghijklmnopqrst", "**************************"},
		{"no sensitive data", "no sensitive data"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := RedactString(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestRedactStringWithPatterns(t *testing.T) {
	customPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)password\s*=\s*(\S+)`),
	}
	input := "password=superSecret"
	got := RedactStringWithPatterns(input, customPatterns)
	assert.Equal(t, "password=***********", got)
}

func TestRedactURL(t *testing.T) {
	assert.Equal(t, "https://example.com/path?***", RedactURL("https://example.com/path?token=abc&secret=def"))
	assert.Equal(t, "https://example.com/path", RedactURL("https://example.com/path"))
}

func TestJaccardSimilarity(t *testing.T) {
	assert.Equal(t, 1.0, JaccardSimilarity("hello world", "hello world"))
	assert.Equal(t, 0.0, JaccardSimilarity("hello", "goodbye"))
	// partial overlap
	similarity := JaccardSimilarity("hello world", "world")
	assert.Greater(t, similarity, 0.0)
	assert.Less(t, similarity, 1.0)
	// empty strings
	assert.Equal(t, 0.0, JaccardSimilarity("", "hello"))
	assert.Equal(t, 0.0, JaccardSimilarity("hello", ""))
}

func TestNewUUID(t *testing.T) {
	uuid := NewUUID()
	assert.NotEmpty(t, uuid)
	// ensure it's a valid UUID format (contains hyphens)
	assert.Contains(t, uuid, "-")
	// second call should be different (extremely likely)
	uuid2 := NewUUID()
	assert.NotEqual(t, uuid, uuid2)
}
