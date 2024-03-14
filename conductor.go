package orchestra

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v4"
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

	// Only use the provided logger if the conductor's logger is nil
	if c.Logger != nil {
		logger = c.Logger
	}

	var wg sync.WaitGroup
	var lock sync.RWMutex

	// This will be sent to the sub players and canceled when the main context ends
	ctxWthCancel, cancel := context.WithCancel(ctx)
	defer cancel() // shutdown players no matter how it exits

	// This will be called after the main context is cancelled
	timedCtx, cancelTimed := context.WithCancel(context.WithoutCancel(ctx))
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
		go c.conductPlayer(ctxWthCancel, &wg, &lock, errs, name, p, logger.WithGroup(name))
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
		logger.Info("conductor stopped after timeout")
		return c.getTimeoutError(&lock)
	case <-allDone:
		logger.Info("conductor exited sucessfully")
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

		l.Info("starting player", slog.String("name", name))

		var bkoff backoff.BackOff = &backoff.StopBackOff{}
		if p, ok := p.(PlayerWithBackoff); ok {
			bkoff = p.Backoff()
		}

		bkoff = backoff.WithContext(bkoff, ctx)

		err := backoff.RetryNotify(func() error {
			if c, ok := p.(*Conductor); ok {
				return c.playWithLogger(ctx, l)
			}
			return p.Play(ctx)
		}, bkoff, func(err error, d time.Duration) {
			l.Error("player failed", slog.Any("err", err), slog.Duration("backoff", d))
		})
		if err != nil && !errors.Is(err, context.Canceled) {
			l.Error("player error", slog.Any("err", err))
			errs <- InstrumentError{name, err}
		}

		l.Info("player stopped")
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
