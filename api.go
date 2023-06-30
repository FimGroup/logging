package logging

type LoggerManager interface {
	GetLogger(loggerName string) Logger
}

type Logger interface {
	Trace(objects ...interface{})
	TraceF(format string, args ...interface{})
	IsTraceEnabled() bool

	Debug(objects ...interface{})
	DebugF(format string, args ...interface{})
	IsDebugEnabled() bool

	Info(objects ...interface{})
	InfoF(format string, args ...interface{})
	IsInfoEnabled() bool

	Warn(objects ...interface{})
	WarnF(format string, args ...interface{})
	IsWarnEnabled() bool

	Error(objects ...interface{})
	ErrorF(format string, args ...interface{})
	IsErrorEnabled() bool
}
