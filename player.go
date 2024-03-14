package orchestra

import (
	"context"
	"os"
	"os/signal"

	"github.com/cenkalti/backoff/v4"
)

// Player is a long running background worker
type Player interface {
	Play(context.Context) error
}

// PlayerWithBackoff is a player that can be restarted with a backoff strategy
type PlayerWithBackoff interface {
	Player
	// A backoff strategy to use when the player fails but returns ErrRestart
	// NOTE: This is only called once before the player is started, so it should be
	// idempotent
	Backoff() backoff.BackOff
}

// PlayUntilSignal starts the player and stops when it receives os.Signals
func PlayUntilSignal(ctx context.Context, p Player, sig ...os.Signal) error {
	ctx, cancel := signal.NotifyContext(ctx, sig...)
	defer cancel()

	return p.Play(ctx)
}
