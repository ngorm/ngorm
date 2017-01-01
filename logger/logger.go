// Package logger defines interface for logging and provide a reference
// implementation.
package logger

// Logger is an interface for logging.
type Logger interface {
	Print(v ...interface{})
	Println(v ...interface{})
}
