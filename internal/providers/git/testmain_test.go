package git

import (
	"testing"

	"github.com/milos85vasic/My-Patreon-Manager/internal/testhelpers"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m, testhelpers.GoleakIgnores()...)
}
