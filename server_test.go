package orchestra

import (
	"net/http"
	"testing"
	"time"
)

func TestNewServerPlayer(t *testing.T) {
	srv := NewServerPlayer(
		WithHTTPServer(&http.Server{Addr: "localhost:4321"}),
		WithTimeout(5*time.Second),
	)
	if srv.Addr != "localhost:4321" {
		t.Errorf(`expected srv.Addr to be "localhost:4321", got: %s`, srv.Addr)
	}
	if srv.Timeout != (time.Second * 5) {
		t.Errorf(`expected srv.Timeout to be "5s", got: %s`, srv.Timeout)
	}
}
