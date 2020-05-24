package orchestra

import (
	"context"
	"fmt"
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
func (c *Conductor) Play(ctx context.Context) error {
	if c.playing == nil {
		c.playing = map[string]struct{}{}
	}

	var wg sync.WaitGroup
	var lock sync.RWMutex

	// This will be sent to the sub daemons and canceled when the main context ends
	ctxWthCancel, cancel := context.WithCancel(ctx)
	defer cancel() // shutdown players no matter how it exits

	// This will be called after the main context is cancelled
	timedCtx, cancelTimed := context.WithCancel(context.Background())
	defer cancelTimed() // release resources at the end regardless

	if c.Timeout < 1 {
		c.Timeout = defaultTimeout
	}

	// cancel all wkers if we receive a signal on the channel
	go func() {
		<-ctx.Done()

		// Cancel the timed context
		time.AfterFunc(c.Timeout, func() {
			cancelTimed()
		})
	}()

	var errs = make(chan InstrumentError, len(c.Players))
	var allDone = make(chan struct{})

	wg.Add(len(c.Players))
	for name, p := range c.Players {
		go c.conductPlayer(ctxWthCancel, &wg, &lock, errs, name, p)
	}

	// Wait for all the players to be done in another goroutine
	go func() {
		wg.Wait()
		close(allDone)
	}()

	select {
	case err := <-errs:
		return fmt.Errorf("error occured in a player: %w", err)
	case <-timedCtx.Done():
		Logger.Printf("Conductor stopped after timeout")
		return c.getTimeoutError(&lock)
	case <-allDone:
		Logger.Printf("All players exited sucessfully")
		return nil
	}
}

// conductPlayer is how the conductor directs each player
func (c *Conductor) conductPlayer(ctx context.Context, wg *sync.WaitGroup, lock *sync.RWMutex, errs chan<- InstrumentError, name string, p Player) {
	defer wg.Done()

	// The function to play our player
	play := func(p Player) {
		err := p.Play(ctx)
		if err != nil {
			errs <- InstrumentError{name, err}
		}
	}

	lock.RLock()
	_, exists := c.playing[name]
	lock.RUnlock()

	if !exists {
		lock.Lock()
		c.playing[name] = struct{}{}
		lock.Unlock()

		play(p)
	}

	lock.Lock()
	delete(c.playing, name)
	lock.Unlock()
}

// getTimeoutError builds a TimeoutErr for the conductor
// It get the names of the players that have not yet stopped to return
func (c *Conductor) getTimeoutError(lock *sync.RWMutex) TimeoutErr {
	lock.RLock()
	err := TimeoutErr{
		Left: make([]string, len(c.playing)),
	}
	for name := range c.playing {
		err.Left = append(err.Left, name)
	}
	lock.RUnlock()

	return err
}
