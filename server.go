package orchestra

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// ServerPlayer is a type that extends the *http.Server
type ServerPlayer struct {
	server          *http.Server
	shutdownTimeout time.Duration
	tls             bool
	backoff         backoff.BackOff
}

// ServerPlayerOption is a function interface to configure the ServerPlayer
type ServerPlayerOption func(s *ServerPlayer)

// NewServerPlayer creates a new ServerPlayer
func NewServerPlayer(srv *http.Server, opts ...ServerPlayerOption) *ServerPlayer {
	if srv == nil {
		srv = &http.Server{}
	}
	s := &ServerPlayer{
		server:          srv,
		shutdownTimeout: 10 * time.Second,
		backoff:         &backoff.StopBackOff{},
	}
	for _, f := range opts {
		f(s)
	}
	return s
}

// WithShutdownTimeout sets the shutdown timeout of ServerPlayer (10s by default)
func WithShutdownTimeout(timeout time.Duration) ServerPlayerOption {
	return func(s *ServerPlayer) {
		s.shutdownTimeout = timeout
	}
}

// WithTLS indicates that the ServerPlayer uses TLS
// so it will use ListenAndServeTLS instead of ListenAndServe
func WithTLS() ServerPlayerOption {
	return func(s *ServerPlayer) {
		s.tls = true
	}
}

// WithBackoff sets the backoff strategy for the ServerPlayer
func WithBackoff(b backoff.BackOff) ServerPlayerOption {
	return func(s *ServerPlayer) {
		s.backoff = b
	}
}

// Play starts the server until the context is done
func (s ServerPlayer) Play(ctxMain context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		var err error
		if s.tls {
			err = s.server.ListenAndServeTLS("", "")
		} else {
			err = s.server.ListenAndServe()
		}

		if err != nil {
			if err != http.ErrServerClosed {
				errChan <- fmt.Errorf("error: failed to start server: %w", err)
				return
			}
		}
	}()

	select {
	case <-ctxMain.Done():
		timeout := s.shutdownTimeout

		timedCtx, cancel := context.WithTimeout(context.WithoutCancel(ctxMain), timeout)
		defer cancel()

		err := s.server.Shutdown(timedCtx)
		if err != nil {
			return fmt.Errorf("error while shutting down server: %v", err)
		}

		return nil

	case err := <-errChan:
		return err
	}
}

// Backoff satisfies the PlayerWithBackoff interface
func (s ServerPlayer) Backoff() backoff.BackOff {
	if s.backoff == nil {
		return &backoff.StopBackOff{}
	}
	return s.backoff
}
