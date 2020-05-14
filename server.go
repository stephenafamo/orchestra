package orchestra

import (
	"context"
	"fmt"
	"log"
	"net/http"
)

// ServerPlayer is a type that extends the *http.Server
type ServerPlayer struct {
	*http.Server
}

// Play starts the server until the context is done
func (s ServerPlayer) Play(ctx context.Context) error {

	errChan := make(chan error, 1)
	go func() {
		log.Printf("starting server")
		if err := s.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				errChan <- fmt.Errorf("error: failed to start server: %w", err)
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("shutting down server")
		err := s.Shutdown(ctx)
		if err != nil {
			return fmt.Errorf("error while shutting down server: %v", err)
		}
		log.Println("shut down successfully")
		return nil

	case err := <-errChan:
		return err
	}
}
