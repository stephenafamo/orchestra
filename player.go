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

	signals := make(chan os.Signal)
	ctx, cancel := context.WithCancel(context.Background())
	signal.Notify(signals, sig...)

	// cancel the context if we receive a SIGINT or SIGTERM
	go func() {
		<-signals
		cancel()
	}()

	return p.Play(ctx)
}
