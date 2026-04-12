package handlers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStubTierGater_VerifyAccess(t *testing.T) {
	s := StubTierGater()
	ok, url, err := s.VerifyAccess(context.Background(), "patron", "content", "tier", nil)
	assert.False(t, ok)
	assert.Empty(t, url)
	assert.ErrorIs(t, err, ErrAccessNotConfigured)
}

func TestStubURLGenerator_VerifySignedURL(t *testing.T) {
	s := StubURLGenerator()
	ok := s.VerifySignedURL("token", "content", "sub", 12345)
	assert.False(t, ok)
}
