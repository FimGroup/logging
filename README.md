# logging

Simple logging package with direct file and other appenders

### Goal of the package

* simple logging package to persist every log to file
* flexible and easy file rotation configurations
* wrap commonly used logging package(currently logrus) for providing general api

# Get started

### 1. Create LoggingManager for file persistent

`manager, err := logging.NewLoggerManager("logs/fim", 1, 1*1024*1024, 2, logrus.TraceLevel, true, true)`

Parameters:

* log file path+prefix
* max days
* max file size
* max file per day
* log level
* enable caller info
* enable print log to console

### 2. Get Logger from LoggingManager

`logger := manager.GetLogger("demoLogger001")`

Parameter:

* Logger name

### 3. Use Logger

```text
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
```

# Dependency

* Logging package: github.com/sirupsen/logrus
* FS manipulation: github.com/spf13/afero

