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
}

// Play starts the server until the context is done
func (s ServerPlayer) Play(ctxMain context.Context) error {

	errChan := make(chan error, 1)
	go func() {
		Logger.Printf("starting server")
		if err := s.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				errChan <- fmt.Errorf("error: failed to start server: %w", err)
				return
			}
		}
	}()

	select {
	case <-ctxMain.Done():
		Logger.Printf("shutting down server")

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := s.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("error while shutting down server: %v", err)
		}

		Logger.Printf("shut down successfully")
		return nil

	case err := <-errChan:
		return err
	}
}
