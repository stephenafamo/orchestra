package orchestra

import (
	"log"
	"os"
)

// Logger is accepted by some Players ([Conductor], [ServerPlayer])
type Logger interface {
	Printf(format string, v ...interface{})
}

// DefaultLogger is used when a conductor's logger is nil
var DefaultLogger Logger = log.New(os.Stderr, "", log.LstdFlags)

type subConductorLogger struct {
	name string
	l    Logger
}

func (s subConductorLogger) Printf(format string, v ...interface{}) {
	l := s.l
	if s.l == nil {
		l = DefaultLogger
	}

	l.Printf("%s: "+format, append([]interface{}{s.name}, v...)...)
}
