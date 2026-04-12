// Package lazy provides a thread-safe lazy initialization wrapper that
// defers expensive resource creation until first use, using sync.Once
// to guarantee single initialization.
package lazy
