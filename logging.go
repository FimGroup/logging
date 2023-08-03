package logging

import (
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	defaultLoggerName           = "DEFAULT_LOGGER"
	loggerNameField             = "FimLogger"
	loggerReportCallerFlagField = "FimLoggerReportCaller"
)

type loggingManagerImpl struct {
	hook                logrus.Hook
	reportCaller        bool
	enableConsoleOutput bool

	loggerLevel logrus.Level
}

func (l *loggingManagerImpl) GetLogger(loggerName string) Logger {
	loggerName = strings.TrimSpace(loggerName)
	logger := logrus.New()
	logger.SetLevel(l.loggerLevel)
	if !l.enableConsoleOutput {
		logger.Out = nilWriteCloser{}
	}
	if l.hook != nil {
		logger.AddHook(l.hook)
	}
	var ln string
	if len(loggerName) > 0 {
		ln = loggerName
	} else {
		ln = defaultLoggerName
	}

	return &loggerImpl{
		logger:       logger,
		loggerName:   ln,
		reportCaller: l.reportCaller,
	}
}

func NewLoggerManager(filePathPrefix string, maxDays, maxFileSize, maxFilePerDay int, lv logrus.Level, enableCallerInfo, enableConsoleOutput bool) (LoggerManager, error) {
	hook, err := newLogrusLoggerHook(filePathPrefix, maxDays, maxFileSize, maxFilePerDay, lv, false)
	return &loggingManagerImpl{
		hook:                hook,
		reportCaller:        enableCallerInfo,
		enableConsoleOutput: enableConsoleOutput,
		loggerLevel:         lv,
	}, err
}

func NewNoFileLoggerManager(lv logrus.Level) LoggerManager {
	return &loggingManagerImpl{
		hook:         nil,
		reportCaller: true,
		loggerLevel:  lv,
	}
}

type loggerImpl struct {
	logger       *logrus.Logger
	loggerName   string
	reportCaller bool
}

func (l *loggerImpl) newEntry() *logrus.Entry {
	return l.logger.WithFields(logrus.Fields{
		loggerNameField:             l.loggerName,
		loggerReportCallerFlagField: l.reportCaller,
	})
}

func (l *loggerImpl) Trace(objects ...interface{}) {
	l.newEntry().Trace(objects...)
}

func (l *loggerImpl) TraceF(format string, args ...interface{}) {
	l.newEntry().Tracef(format, args...)
}

func (l *loggerImpl) IsTraceEnabled() bool {
	return l.logger.IsLevelEnabled(logrus.TraceLevel)
}

func (l *loggerImpl) Debug(objects ...interface{}) {
	l.newEntry().Debug(objects...)
}

func (l *loggerImpl) DebugF(format string, args ...interface{}) {
	l.newEntry().Debugf(format, args...)
}

func (l *loggerImpl) IsDebugEnabled() bool {
	return l.logger.IsLevelEnabled(logrus.DebugLevel)
}

func (l *loggerImpl) Info(objects ...interface{}) {
	l.newEntry().Info(objects...)
}

func (l *loggerImpl) InfoF(format string, args ...interface{}) {
	l.newEntry().Infof(format, args...)
}

func (l *loggerImpl) IsInfoEnabled() bool {
	return l.logger.IsLevelEnabled(logrus.InfoLevel)
}

func (l *loggerImpl) Warn(objects ...interface{}) {
	l.newEntry().Warn(objects...)
}

func (l *loggerImpl) WarnF(format string, args ...interface{}) {
	l.newEntry().Warnf(format, args...)
}

func (l *loggerImpl) IsWarnEnabled() bool {
	return l.logger.IsLevelEnabled(logrus.WarnLevel)
}

func (l *loggerImpl) Error(objects ...interface{}) {
	l.newEntry().Error(objects...)
}

func (l *loggerImpl) ErrorF(format string, args ...interface{}) {
	l.newEntry().Errorf(format, args...)
}

func (l *loggerImpl) IsErrorEnabled() bool {
	return l.logger.IsLevelEnabled(logrus.ErrorLevel)
}
