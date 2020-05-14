package orchestra

import (
	"context"
)

// PlayerFunc is a function type that satisfies the Player interface
type PlayerFunc func(ctx context.Context) error

// Play satisfies the Player interface
func (f PlayerFunc) Play(ctx context.Context) error {
	return f(ctx)
}
