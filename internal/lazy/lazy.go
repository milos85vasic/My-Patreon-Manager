package lazy

import "sync"

// Value is a generic lazily-initialized value. The factory function is
// called at most once, on the first call to Get, even under concurrent
// access from multiple goroutines.
type Value[T any] struct {
	once sync.Once
	fn   func() (T, error)
	val  T
	err  error
}

// New returns a Value that will call fn on the first Get.
func New[T any](fn func() (T, error)) *Value[T] { return &Value[T]{fn: fn} }

// Get returns the lazily-computed value (and any error from the factory).
// Subsequent calls always return the cached result.
func (v *Value[T]) Get() (T, error) {
	v.once.Do(func() { v.val, v.err = v.fn() })
	return v.val, v.err
}
