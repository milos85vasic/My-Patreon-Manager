package lazy

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValue_ComputesOnce(t *testing.T) {
	var calls atomic.Int64
	v := New(func() (int, error) {
		calls.Add(1)
		return 42, nil
	})

	got, err := v.Get()
	require.NoError(t, err)
	assert.Equal(t, 42, got)
	assert.Equal(t, int64(1), calls.Load())

	// Second call returns cached value.
	got2, err2 := v.Get()
	require.NoError(t, err2)
	assert.Equal(t, 42, got2)
	assert.Equal(t, int64(1), calls.Load())
}

func TestValue_PropagatesError(t *testing.T) {
	expectedErr := errors.New("init failed")
	v := New(func() (string, error) {
		return "", expectedErr
	})

	val, err := v.Get()
	assert.ErrorIs(t, err, expectedErr)
	assert.Equal(t, "", val)

	// Error is cached too.
	val2, err2 := v.Get()
	assert.ErrorIs(t, err2, expectedErr)
	assert.Equal(t, "", val2)
}

func TestValue_ConcurrentAccess(t *testing.T) {
	const goroutines = 32
	var calls atomic.Int64
	v := New(func() (string, error) {
		calls.Add(1)
		return "result", nil
	})

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			got, err := v.Get()
			assert.NoError(t, err)
			assert.Equal(t, "result", got)
		}()
	}
	wg.Wait()

	assert.Equal(t, int64(1), calls.Load(), "factory must be called exactly once under contention")
}
