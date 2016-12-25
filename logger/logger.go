package logger

type Logger interface {
	Print(v ...interface{})
	Println(v ...interface{})
}
