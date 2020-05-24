package orchestra

import (
	"log"
	"os"
)

type logger interface {
	Printf(format string, v ...interface{})
}

// Logger to print lgos
var Logger logger = log.New(os.Stderr, "", log.LstdFlags)
