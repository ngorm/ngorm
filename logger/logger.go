// Package logger defines interface for logging and provide a reference
// implementation.
package logger

import (
	"time"
)

// Logger is an interface for logging.
type Logger interface {
}

//Zapper is the ngorm logger that is based on uber/zap package.
type Zapper struct {
	start    time.Time
	withTime bool
	log      Logger
}

//New return new Zapper instance with the l set as the default logger.
func New(l Logger) *Zapper {
	return &Zapper{
		log: l,
	}
}

//Start return Zapper instance
func (z *Zapper) Start() *Zapper {
	return &Zapper{
		log: z.log,
	}
}

func (z *Zapper) Info(v ...interface{}) {
}
