package logging

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

const spaceByte = byte(' ')

type logrusLoggerHook struct {
	PathFormatPrefix string // including  folder name. file full name = {prefix}.YYYY-MM-DD.log
	RetainMaxDays    int    // default 7 days, current day excluded
	MaxSizePerFile   int    // default 100*1024*1024=100MB
	MaxFilePerDay    int    // default 10 files. {file full name}.1  {file full name}.2  {file full name}.3 ....
	MinLogLevel      logrus.Level

	debugFlag bool

	outputStream         io.WriteCloser
	lock                 sync.Mutex
	currentLoggerContext struct {
		lastYearDay     int
		fileSize        int
		currentFilePath string
	}

	fs             afero.Fs
	filePathFormat string
}

func (l *logrusLoggerHook) Shutdown() error {
	//FIXME should support shutdown - release all the resources
	panic("implement me!")
}

func (l *logrusLoggerHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.PanicLevel,
		logrus.FatalLevel,
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}
}

func (l *logrusLoggerHook) Fire(entry *logrus.Entry) error {
	if entry.Level > l.MinLogLevel {
		if l.debugFlag {
			log.Printf("[LoggingDebug] log level below min log level: msg level=%s, min level=%s", fmt.Sprint(entry.Level), fmt.Sprint(l.MinLogLevel))
		}
		return nil
	}
	var caller *runtime.Frame
	{
		// caller report
		flagObj, ok := entry.Data[loggerReportCallerFlagField]
		if flag, convOk := flagObj.(bool); ok && convOk && flag {
			caller = getCaller000()
		}
	}

	var data []byte
	{
		buf := bytes.NewBuffer(nil)
		// format time
		timeString := entry.Time.Format(time.RFC3339Nano)
		buf.Write([]byte(timeString))
		buf.WriteByte(spaceByte)
		// level
		buf.WriteByte('[')
		buf.Write([]byte(strings.ToUpper(entry.Level.String())))
		buf.WriteByte(']')
		buf.WriteByte(spaceByte)
		// logger name
		loggerName, ok := entry.Data["FimLogger"]
		if ok {
			buf.Write([]byte(fmt.Sprint(loggerName)))
		}
		if caller != nil {
			// caller info
			//buf.Write([]byte(fmt.Sprintf(":%s:%s:%d", entry.Caller.File, entry.Caller.Function, entry.Caller.Line)))
			buf.Write([]byte(fmt.Sprintf(":%s:%d", caller.Function, caller.Line)))
		}
		buf.WriteByte(spaceByte)
		// split
		buf.Write([]byte("- "))
		// message
		buf.Write([]byte(entry.Message))
		buf.WriteByte('\n')

		data = buf.Bytes()
	}
	return l.writeData(data, entry)
}

func (l *logrusLoggerHook) writeData(data []byte, entry *logrus.Entry) error {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.ensureFile(data, entry)
	// increase size
	l.currentLoggerContext.fileSize += len(data)
	// do write
	if l.outputStream != nil {
		_, err := l.outputStream.Write(data)
		return err
	} else {
		log.Println("[ERROR] no open output stream for logging")
		return nil
	}
}

func (l *logrusLoggerHook) ensureFile(data []byte, entry *logrus.Entry) {
	needFileDateDeletion := false

	// need day rotation
	if l.currentLoggerContext.lastYearDay != entry.Time.YearDay() {
		fileSize := 0

		newFileName := fmt.Sprintf(l.filePathFormat, l.PathFormatPrefix, entry.Time.Year(), entry.Time.Month(), entry.Time.Day())
		exists, err := afero.Exists(l.fs, newFileName)
		if err != nil {
			log.Printf("[ERROR] check file=[%s] existing failed=[%s].", newFileName, err)
			return //do nothing when error
		}
		if exists {
			if stat, err := l.fs.Stat(newFileName); err != nil {
				log.Printf("[ERROR] stat file=[%s] failed=[%s].", newFileName, err)
				return //do nothing when error
			} else if stat.Size() > int64(l.MaxSizePerFile) {
				if err := l.rotateMaxFileSize(newFileName, entry); err != nil {
					log.Printf("[ERROR] rotateMaxFileSize for file=[%s] failed=[%s].", newFileName, err)
					return //do nothing when error
				}
			} else {
				fileSize = int(stat.Size())
			}
		}
		if err := l.fs.MkdirAll(filepath.Dir(newFileName), 0755); err != nil {
			log.Printf("[ERROR] mkdirAll log file=[%s] failed=[%s].", filepath.Dir(newFileName), err)
			return //do nothing when error
		}
		outputStream, err := l.fs.OpenFile(newFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("[ERROR] open log file=[%s] failed=[%s].", newFileName, err)
			return //do nothing when error
		}

		l.currentLoggerContext.lastYearDay = entry.Time.YearDay()
		l.currentLoggerContext.fileSize = fileSize
		l.currentLoggerContext.currentFilePath = newFileName
		if l.outputStream != nil {
			_ = l.outputStream.Close() // close anyway
		}
		l.outputStream = outputStream
		needFileDateDeletion = true
	}

	// rotate max file per day
	{
		if l.currentLoggerContext.fileSize > l.MaxSizePerFile {
			if err := l.rotateMaxFileSize(l.currentLoggerContext.currentFilePath, entry); err != nil {
				log.Printf("[ERROR] rotateMaxFileSize log file=[%s] failed=[%s].", l.currentLoggerContext.currentFilePath, err)
				return //do nothing when error
			}
			newFileName := l.currentLoggerContext.currentFilePath

			if err := l.fs.MkdirAll(filepath.Dir(newFileName), 0755); err != nil {
				log.Printf("[ERROR] mkdirAll log file=[%s] failed=[%s].", filepath.Dir(newFileName), err)
				return //do nothing when error
			}
			outputStream, err := l.fs.OpenFile(newFileName, os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				log.Printf("[ERROR] open log file=[%s] failed=[%s].", newFileName, err)
				return //do nothing when error
			}

			l.currentLoggerContext.lastYearDay = entry.Time.YearDay()
			l.currentLoggerContext.fileSize = 0
			l.currentLoggerContext.currentFilePath = newFileName
			if l.outputStream != nil {
				_ = l.outputStream.Close() // close anyway
			}
			l.outputStream = outputStream
		}
	}

	// delete file based on the limit file count per day
	if needFileDateDeletion {
		if err := l.removeOutOfDateFiles(entry); err != nil {
			log.Printf("[ERROR] removeOutOfDateFiles failed=[%s].", err)
			return //do nothing when error
		}
	}
}

func (l *logrusLoggerHook) initLoad() error {
	return nil
}

func (l *logrusLoggerHook) rotateMaxFileSize(currentFilePath string, entry *logrus.Entry) error {
	// only rotate old files and prepare new file metadata

	// list all files with prefix
	newFileName := fmt.Sprintf(l.filePathFormat, l.PathFormatPrefix, entry.Time.Year(), entry.Time.Month(), entry.Time.Day())
	var fileNamePrefix string
	{
		i := len(newFileName) - 1
		for i >= 0 && !os.IsPathSeparator(newFileName[i]) {
			i--
		}
		fileNamePrefix = newFileName[i+1:] + "."
	}
	folder := filepath.Dir(newFileName)
	dirs, err := fs.ReadDir(afero.NewIOFS(l.fs), folder)
	if err != nil {
		return err
	}
	var existingFiles []struct {
		Name string
		Idx  int
	}
	for _, v := range dirs {
		if !v.IsDir() && strings.HasPrefix(v.Name(), fileNamePrefix) {
			number := v.Name()[len(fileNamePrefix):]
			n, err := strconv.Atoi(number)
			if err == nil {
				existingFiles = append(existingFiles, struct {
					Name string
					Idx  int
				}{Name: v.Name(), Idx: n})
			} else {
				log.Printf("[ERROR] error existing file name with non-number suffix=[%s]", v.Name())
			}
		}
	}
	sort.Slice(existingFiles, func(i, j int) bool {
		return existingFiles[i].Idx > existingFiles[j].Idx
	})

	// process files
	if len(existingFiles) == 0 {
		if l.outputStream != nil {
			_ = l.outputStream.Close() // close anyway
			l.outputStream = nil
		}
		// no existing files
		if err := l.fs.Rename(currentFilePath, l.currentLoggerContext.currentFilePath+".1"); err != nil {
			return err
		}
	} else if len(existingFiles)+1+1 > l.MaxFilePerDay {
		// need remove due to the number (existing files + 1 newly archived file + 1 new file) is over max file per day
		toRemoveFiles := existingFiles[l.MaxFilePerDay-2:]
		for _, v := range toRemoveFiles {
			if err := l.fs.Remove(filepath.Join(folder, v.Name)); err != nil {
				return err
			}
		}
		if l.outputStream != nil {
			_ = l.outputStream.Close() // close anyway
			l.outputStream = nil
		}
		if err := l.fs.Rename(currentFilePath, l.currentLoggerContext.currentFilePath+"."+fmt.Sprint(existingFiles[0].Idx+1)); err != nil {
			return err
		}
	} else {
		// have more file room. just create new file
		if l.outputStream != nil {
			_ = l.outputStream.Close() // close anyway
			l.outputStream = nil
		}
		if err := l.fs.Rename(currentFilePath, l.currentLoggerContext.currentFilePath+"."+fmt.Sprint(existingFiles[0].Idx+1)); err != nil {
			return err
		}
	}
	// self assignment is omitted - filepath, since it isn't changed
	l.currentLoggerContext.fileSize = 0
	l.currentLoggerContext.lastYearDay = entry.Time.YearDay()

	return nil
}

func (l *logrusLoggerHook) removeOutOfDateFiles(entry *logrus.Entry) error {
	//FIXME directly remove oldest files according to MaxDays

	var loggerFilePrefix = l.PathFormatPrefix
	// extrace file name prefix (parent folder excluded)
	{
		i := len(loggerFilePrefix) - 1
		for i >= 0 && !os.IsPathSeparator(loggerFilePrefix[i]) {
			i--
		}
		loggerFilePrefix = loggerFilePrefix[i+1:] + "."
	}

	var retainedFilePrefix []string
	for i := 0; i < l.RetainMaxDays; i++ {
		curTime := entry.Time.Add(-(time.Duration(i) * 24 * time.Hour))
		newFileName := fmt.Sprintf(l.filePathFormat, l.PathFormatPrefix, curTime.Year(), curTime.Month(), curTime.Day())
		var fileNamePrefix string
		{
			i := len(newFileName) - 1
			for i >= 0 && !os.IsPathSeparator(newFileName[i]) {
				i--
			}
			fileNamePrefix = newFileName[i+1:]
		}
		retainedFilePrefix = append(retainedFilePrefix, fileNamePrefix)
	}

	newFileName := fmt.Sprintf(l.filePathFormat, l.PathFormatPrefix, entry.Time.Year(), entry.Time.Month(), entry.Time.Day())
	folder := filepath.Dir(newFileName)
	dirs, err := fs.ReadDir(afero.NewIOFS(l.fs), folder)
	if err != nil {
		return err
	}
	var filesToDelete []string
Loop:
	for _, v := range dirs {
		if !v.IsDir() {
			for _, re := range retainedFilePrefix {
				if strings.HasPrefix(v.Name(), re) {
					continue Loop
				}
			}
			// avoid deleting other files without the specific prefix
			if strings.HasPrefix(v.Name(), loggerFilePrefix) {
				filesToDelete = append(filesToDelete, v.Name())
			}
		}
	}

	// do deletion
	for _, v := range filesToDelete {
		if err := l.fs.Remove(filepath.Join(folder, v)); err != nil {
			return err
		}
	}

	return nil
}

func newLogrusLoggerHook(filePathPrefix string, maxDays, maxFileSize, maxFilePerDay int, lv logrus.Level, debugFlag bool) (logrus.Hook, error) {
	if maxDays <= 0 {
		maxDays = 7
	}
	if maxFileSize <= 0 {
		maxFileSize = 100 * 1024 * 1024
	}
	if maxFilePerDay < 2 { //FIXME should >= 2, for current file and 1 archive file
		maxFilePerDay = 10
	}

	l := &logrusLoggerHook{
		PathFormatPrefix: filePathPrefix,
		RetainMaxDays:    maxDays + 1,
		MaxSizePerFile:   maxFileSize,
		MinLogLevel:      lv,
		MaxFilePerDay:    maxFilePerDay,
		debugFlag:        debugFlag,
		fs:               afero.NewOsFs(),
		filePathFormat:   "%s.%4d-%02d-%02d.log",
	}

	if err := l.initLoad(); err != nil {
		return nil, err
	}

	return l, nil
}

type logrusStructuralLoggerHook struct {
	//FIXME TBD
}

//============================ the following codes are from logrus==============================
// In order to support more levels of wrapping logrus raw api, the codes have to be modified

var (
	logrusPackage      string
	minimumCallerDepth int
	callerInitOnce     sync.Once
)

const (
	maximumCallerDepth int = 25
	knownLogrusFrames  int = 9
)

// getCaller000 retrieves the name of the first non-logrus calling function
func getCaller000() *runtime.Frame {
	// cache this package's fully-qualified name
	callerInitOnce.Do(func() {
		pcs := make([]uintptr, maximumCallerDepth)
		_ = runtime.Callers(0, pcs)

		// dynamic get the package name and the minimum caller depth
		for i := 0; i < maximumCallerDepth; i++ {
			funcName := runtime.FuncForPC(pcs[i]).Name()
			if strings.Contains(funcName, "getCaller000") {
				logrusPackage = getPackageName(funcName)
				break
			}
		}

		minimumCallerDepth = knownLogrusFrames
	})

	// Restrict the lookback frames to avoid runaway lookups
	pcs := make([]uintptr, maximumCallerDepth)
	depth := runtime.Callers(minimumCallerDepth, pcs)
	frames := runtime.CallersFrames(pcs[:depth])

	for f, again := frames.Next(); again; f, again = frames.Next() {
		pkg := getPackageName(f.Function)

		// If the caller isn't part of this package, we're done
		if pkg != logrusPackage {
			return &f //nolint:scopelint
		}
	}

	// if we got here, we failed to find the caller's context
	return nil
}

// getPackageName reduces a fully qualified function name to the package name
// There really ought to be to be a better way...
func getPackageName(f string) string {
	for {
		lastPeriod := strings.LastIndex(f, ".")
		lastSlash := strings.LastIndex(f, "/")
		if lastPeriod > lastSlash {
			f = f[:lastPeriod]
		} else {
			break
		}
	}

	return f
}
