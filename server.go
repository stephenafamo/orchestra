package orchestra

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ServerPlayer is a type that extends the *http.Server
type ServerPlayer struct {
	server          *http.Server
	shutdownTimeout time.Duration
}

// ServerPlayerOption is a function interface to configure the ServerPlayer
type ServerPlayerOption func(s *ServerPlayer)

// NewServerPlayer creates a new ServerPlayer
func NewServerPlayer(srv *http.Server, opts ...ServerPlayerOption) *ServerPlayer {
	s := &ServerPlayer{
		server:          srv,
		shutdownTimeout: 10 * time.Second,
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

// Play starts the server until the context is done
func (s ServerPlayer) Play(ctxMain context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				errChan <- fmt.Errorf("error: failed to start server: %w", err)
				return
			}
		}
	}()

	select {
	case <-ctxMain.Done():
		timeout := s.shutdownTimeout

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := s.server.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("error while shutting down server: %v", err)
		}

		return nil

	case err := <-errChan:
		return err
	}
}
