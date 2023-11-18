package orchestra

import (
	"context"
	"os"
	"os/signal"
)

// Player is a long running background worker
type Player interface {
	Play(context.Context) error
}

// PlayUntilSignal starts the player and stops when it receives os.Signals
func PlayUntilSignal(ctx context.Context, p Player, sig ...os.Signal) error {
	ctx, cancel := signal.NotifyContext(ctx, sig...)
	defer cancel()

	return p.Play(ctx)
}
