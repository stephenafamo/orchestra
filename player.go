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

// PlayUntilSignal starts the player and stops when it recieves os.Signals
func PlayUntilSignal(p Player, sig ...os.Signal) error {
	ctx, cancel := signal.NotifyContext(context.Background(), sig...)
	defer cancel()

	return p.Play(ctx)
}
