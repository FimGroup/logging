package logging

import (
	"errors"
	"log"
)

var loggerManager LoggerManager = defaultLoggerManager{}

func SetLoggerManager(lm LoggerManager) {
	if lm == nil {
		panic(errors.New("LoggerManager should not be nil"))
	}
	if _, ok := loggerManager.(defaultLoggerManager); ok {
		loggerManager = lm
	} else {
		panic(errors.New("cannot re-update LoggerManager"))
	}
}

func GetLoggerManager() LoggerManager {
	return loggerManager
}

type defaultLoggerManager struct {
}

func (d defaultLoggerManager) GetLogger(loggerName string) Logger {
	return defaultLogger{}
}

type defaultLogger struct {
}

func (d defaultLogger) Trace(objects ...interface{}) {
	log.Println(objects...)
}

func (d defaultLogger) TraceF(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (d defaultLogger) IsTraceEnabled() bool {
	return true
}

func (d defaultLogger) Debug(objects ...interface{}) {
	log.Println(objects...)
}

func (d defaultLogger) DebugF(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (d defaultLogger) IsDebugEnabled() bool {
	return true
}

func (d defaultLogger) Info(objects ...interface{}) {
	log.Println(objects...)
}

func (d defaultLogger) InfoF(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (d defaultLogger) IsInfoEnabled() bool {
	return true
}

func (d defaultLogger) Warn(objects ...interface{}) {
	log.Println(objects...)
}

func (d defaultLogger) WarnF(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (d defaultLogger) IsWarnEnabled() bool {
	return true
}

func (d defaultLogger) Error(objects ...interface{}) {
	log.Println(objects...)
}

func (d defaultLogger) ErrorF(format string, args ...interface{}) {
	log.Printf(format, args...)
}

func (d defaultLogger) IsErrorEnabled() bool {
	return true
}
