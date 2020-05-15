package orchestra

import (
	"context"
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
	if c.playing == nil {
		c.playing = map[string]struct{}{}
	}

	var wg sync.WaitGroup
	var lock sync.RWMutex

	// This will be sent to the sub daemons and canceled when the main context ends
	ctxWthCancel, cancel := context.WithCancel(context.Background())

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

	var errs = make(chan InstrumentError)
	var allDone = make(chan struct{})

	wg.Add(len(c.Players))
	for name, p := range c.Players {
		go func(name string, p Player) {

			// The function to play our player
			play := func(p Player) {
				err := p.Play(ctxWthCancel)
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

			wg.Done()
		}(name, p)
	}

	// Wait for all the players to be done in another goroutine
	go func() {
		wg.Wait()
		close(allDone)
	}()

	select {
	case err := <-errs:
		log.Printf("Error occured in a player: %s\n", err.Name)
		return err
	case <-timedCtx.Done():
		log.Println("Pool stopped after timeout")
		return nil
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
