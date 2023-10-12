package orchestra

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ServerPlayer is a type that extends the *http.Server
type ServerPlayer struct {
	*http.Server
	ShutdownTimeout time.Duration
}

// ServerPlayerOption is a function interface to configure the ServerPlayer
type ServerPlayerOption func(s *ServerPlayer)

func NewServerPlayer(opts ...ServerPlayerOption) *ServerPlayer {
	s := &ServerPlayer{
		Server:          &http.Server{},
		ShutdownTimeout: 10 * time.Second,
	}
	for _, f := range opts {
		f(s)
	}
	return s
}

// WithShutdownTimeout sets the shutdown timeout of ServerPlayer (10s by default)
func WithShutdownTimeout(timeout time.Duration) ServerPlayerOption {
	return func(s *ServerPlayer) {
		s.ShutdownTimeout = timeout
	}
}

// WithHTTPServer allow configuring the http.Server of ServerPlayer
func WithHTTPServer(srv *http.Server) ServerPlayerOption {
	return func(s *ServerPlayer) {
		s.Server = srv
	}
}

// Play starts the server until the context is done
func (s ServerPlayer) Play(ctxMain context.Context) error {
	errChan := make(chan error, 1)
	go func() {
		if err := s.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				errChan <- fmt.Errorf("error: failed to start server: %w", err)
				return
			}
		}
	}()

	select {
	case <-ctxMain.Done():
		timeout := s.ShutdownTimeout

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		err := s.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("error while shutting down server: %v", err)
		}

		return nil

	case err := <-errChan:
		return err
	}
}
