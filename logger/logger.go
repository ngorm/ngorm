package logger

type Logger interface {
	Print(v ...interface{})
}

// LogWriter log writer interface
type LogWriter interface {
	Println(v ...interface{})
}
