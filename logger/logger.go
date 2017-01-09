// Package logger defines interface for logging and provide a reference
// implementation.
package logger

import (
	"time"

	"github.com/uber-go/zap"
)

// Logger is an interface for logging.
type Logger interface {
	zap.Logger
}

//Zapper is the ngorm logger that is based on uber/zap package.
type Zapper struct {
	start    time.Time
	withTime bool
	fiedls   []zap.Field
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

//StartWithTime  return Zapper instance with start time set to now. This is useful if you
//want to tract duration of a certain event.
//
// Any log methof called on the returned istance will record the duration.
func (z *Zapper) StartWithTime() *Zapper {
	return &Zapper{
		start:    time.Now(),
		log:      z.log,
		withTime: true,
	}
}

// Log logs val with level or fields.
func (z *Zapper) Log(level zap.Level, val string, fields ...zap.Field) {
	if z.withTime {
		now := time.Now()
		z.fiedls = append(z.fiedls, zap.Stringer("elapsed time", now.Sub(z.start)))
		z.start = now
	}
	var f []zap.Field
	for _, fd := range z.fiedls {
		f = append(f, fd)
	}
	for _, fd := range fields {
		f = append(f, fd)
	}
	z.log.Log(level, val, f...)
}

//Info logs with level set to info
func (z *Zapper) Info(arg string, fields ...zap.Field) {
	z.Log(zap.InfoLevel, arg, fields...)
}

// Debug logs with level set to debug
func (z *Zapper) Debug(arg string, fields ...zap.Field) {
	z.Log(zap.DebugLevel, arg, fields...)
}

//Warn logs warnings
func (z *Zapper) Warn(arg string, fields ...zap.Field) {
	z.Log(zap.WarnLevel, arg, fields...)
}

//Fields Add fields
func (z *Zapper) Fields(f ...zap.Field) {
	z.fiedls = append(z.fiedls, f...)
}
