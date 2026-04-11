package audit

import (
	"testing"

	"go.uber.org/goleak"

	"github.com/milos85vasic/My-Patreon-Manager/internal/testhelpers"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, testhelpers.GoleakIgnores()...)
}
