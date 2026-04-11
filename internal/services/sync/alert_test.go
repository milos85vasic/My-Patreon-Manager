package sync

import (
	"testing"
)

func TestLogAlert_Send(t *testing.T) {
	alert := &LogAlert{}
	// Should not panic
	alert.Send("subject", "body")
	// No return value to assert
}
