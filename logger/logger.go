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

type Zapper struct {
	start    time.Time
	withTime bool
	fiedls   []zap.Field
	log      Logger
}

func New(l Logger) *Zapper {
	return &Zapper{
		log: l,
	}
}

func (z *Zapper) Start() *Zapper {
	return &Zapper{
		log: z.log,
	}
}
func (z *Zapper) StartWithTime() *Zapper {
	return &Zapper{
		start: time.Now(),
		log:   z.log,
	}
}

func (z *Zapper) Log(level zap.Level, val string, fields ...zap.Field) {
	if z.withTime {
		now := time.Now()
		z.fiedls = append(z.fiedls, zap.Duration("elapsed time", now.Sub(z.start)))
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

func (z *Zapper) Info(arg string, fields ...zap.Field) {
	z.Log(zap.InfoLevel, arg, fields...)
}

func (z *Zapper) Debug(arg string, fields ...zap.Field) {
	z.Log(zap.InfoLevel, arg, fields...)
}

func (z *Zapper) Fatal(arg string, fields ...zap.Field) {
	z.Log(zap.FatalLevel, arg, fields...)
}
func (z *Zapper) Warn(arg string, fields ...zap.Field) {
	z.Log(zap.WarnLevel, arg, fields...)
}
