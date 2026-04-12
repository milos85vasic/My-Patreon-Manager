package handlers

import (
	"context"
	"errors"
)

// ErrAccessNotConfigured is returned by stub implementations of tierGater
// and signedURLGenerator when the real access-control dependencies have not
// been wired into the server. This prevents nil-pointer panics while
// surfacing a clear "not configured" error to callers.
var ErrAccessNotConfigured = errors.New("access control not configured")

// stubTierGater satisfies the tierGater interface but always returns a
// service-unavailable error. Used as a safe default when no real tier gater
// is available.
type stubTierGater struct{}

func (stubTierGater) VerifyAccess(_ context.Context, _, _, _ string, _ []string) (bool, string, error) {
	return false, "", ErrAccessNotConfigured
}

// StubTierGater returns a tierGater that rejects all access checks with
// ErrAccessNotConfigured. Exported so cmd/server can use it as a safe
// default instead of passing nil.
func StubTierGater() stubTierGater { return stubTierGater{} }

// stubURLGenerator satisfies the signedURLGenerator interface but always
// returns false. Used as a safe default when no real URL generator is
// available.
type stubURLGenerator struct{}

func (stubURLGenerator) VerifySignedURL(_, _, _ string, _ int64) bool {
	return false
}

// StubURLGenerator returns a signedURLGenerator that rejects all URL
// verifications. Exported so cmd/server can use it as a safe default
// instead of passing nil.
func StubURLGenerator() stubURLGenerator { return stubURLGenerator{} }
