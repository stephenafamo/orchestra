package orchestra

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

const defaultTimeout time.Duration = 9 * time.Second

// Conductor is a group of players. It is also a Player itself **evil laugh**
type Conductor struct {
	Timeout time.Duration
	Players map[string]Player

	playing map[string]struct{}
}

// Play starts all the players and gracefully shuts them down
func (c *Conductor) Play(ctxMain context.Context) error {
	c.playing = map[string]struct{}{}

	// This will be sent to the sub daemons and canceled when the main context ends
	ctxWthCancel, cancel := context.WithCancel(ctxMain)

	// This will be called after the main context is cancelled
	timedCtx, cancelTimed := context.WithCancel(context.Background())

	if c.Timeout < 1 {
		c.Timeout = defaultTimeout
	}

	// cancel all wkers if we receive a signal on the channel
	go func() {
		<-ctxMain.Done()
		cancel()

		// Cancel the timed context
		time.AfterFunc(c.Timeout, func() {
			cancelTimed()
		})
	}()

	// Unbuffered channels would block if nothing is reading off them
	// So buffered them to the len of Players so goroutines are not blocked
	// if they all error.
	var errs = make(chan InstrumentError, len(c.Players))
	var allDone = make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(len(c.Players))
	for name, p := range c.Players {
		_, exists := c.playing[name]

		if exists {
			// Ensure we inform all other players to shutdown
			cancel()
			return fmt.Errorf("%s already exists", name)
		}

		go func(name string, p Player) {
			defer wg.Done()

			if err := p.Play(ctxWthCancel); err != nil {
				errs <- InstrumentError{name, err}
			}
		}(name, p)
	}

	// Wait for all the players to be done in another goroutine
	go func() {
		wg.Wait()
		close(allDone)
	}()

	select {
	case err := <-errs:
		// Ensure we inform all other players to shutdown
		cancel()
		// Handle the error once (logging an error is handling it once,
		// and returning it is a second action on the same error).
		return fmt.Errorf("Error occured in a player: %s", err.Name)
	case <-timedCtx.Done():
		// If this times out, then players aren't shutting down correctly.
		// We need to inform the users of this library that one or more of their
		// players hasn't shutdown when the context's done channel was closed.
		return errors.New("Forcing shutdown of conductor, not all players may have shutdown")
	case <-allDone:
		log.Println("All players exited sucessfully")
		return nil
	}
}

// InstrumentError is an error that happens in an instrument started by a conductor
// It carries the name of the instrument
type InstrumentError struct {
	Name string
	Err  error
}

func (e InstrumentError) Error() string {
	return fmt.Sprintf("%s | %s", e.Name, e.Err)
}

func (e InstrumentError) Unwrap() error {
	return e.Err
}
