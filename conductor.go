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
	Logger  Logger

	playing map[string]struct{}
}

// Play starts all the players and gracefully shuts them down
func (c *Conductor) Play(ctx context.Context) error {
	logger := c.Logger
	if logger == nil {
		logger = DefaultLogger
	}

	return c.playWithLogger(ctx, logger)
}

func (c *Conductor) playWithLogger(ctx context.Context, logger Logger) error {
	if c.playing == nil {
		c.playing = map[string]struct{}{}
	}

	var wg sync.WaitGroup
	var lock sync.RWMutex

	// This will be sent to the sub players and canceled when the main context ends
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

	errs := make(chan InstrumentError, len(c.Players))
	allDone := make(chan struct{})

	wg.Add(len(c.Players))
	for name, p := range c.Players {
		go c.conductPlayer(ctxWthCancel, &wg, &lock, errs, name, p, logger)
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
		logger.Log("msg", "conductor stopped after timeout")
		return c.getTimeoutError(&lock)
	case <-allDone:
		logger.Log("msg", "conductor exited sucessfully")
		return nil
	}
}

// conductPlayer is how the conductor directs each player
func (c *Conductor) conductPlayer(ctx context.Context, wg *sync.WaitGroup, lock *sync.RWMutex, errs chan<- InstrumentError, name string, p Player, l Logger) {
	defer wg.Done()

	lock.RLock()
	_, exists := c.playing[name]
	lock.RUnlock()

	if !exists {
		lock.Lock()
		c.playing[name] = struct{}{}
		lock.Unlock()

		l.Log("msg", "starting player", "name", name)

		var err error
		if c, ok := p.(*Conductor); ok {
			err = c.playWithLogger(ctx, subConductorLogger{
				name: name,
				l:    c.Logger,
			})
		} else {
			err = p.Play(ctx)
		}

		if err != nil {
			DefaultLogger.Log("error in " + name)
			errs <- InstrumentError{name, err}
		}
		l.Log("msg", "stopped player", "name", name)
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
