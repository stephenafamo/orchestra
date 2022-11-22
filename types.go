package orchestra

import (
	"errors"
	"fmt"
)

// Logger is accepted by some Players ([Conductor], [ServerPlayer])
type Logger interface {
	Log(keyvals ...interface{}) error
}

// DefaultLogger is used when a conductor's logger is nil
var DefaultLogger Logger = defaultLogger{}

type defaultLogger struct{}

func (d defaultLogger) Log(keyvals ...interface{}) error {
	pairLen := len(keyvals)

	if pairLen < 1 || pairLen%2 != 0 {
		return errors.New("non-even number of values to log")
	}

	for i := range keyvals {
		if i%2 != 0 {
			continue
		}

		fmt.Printf("%v=%q ", keyvals[i], keyvals[i+1])
	}

	// Move to next line
	fmt.Println()

	return nil
}

type subConductorLogger struct {
	name string
	l    Logger
}

func (s subConductorLogger) Log(keyvals ...interface{}) error {
	l := s.l
	if s.l == nil {
		l = DefaultLogger
	}

	return l.Log(append([]interface{}{"conductor", s.name}, keyvals...)...)
}
