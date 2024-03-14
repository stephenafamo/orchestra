package orchestra

import (
	"fmt"
	"strings"
)

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

// TimeoutErr is an error that happens when a conductor terminates because of the timeout
// and does not wait for all players to exit sucessfully
type TimeoutErr struct {
	Left []string
}

func (e TimeoutErr) Error() string {
	return fmt.Sprintf("conductor stopped after timeout\nThe following players did not stop gracefully:%s", strings.Join(e.Left, "\n"))
}
