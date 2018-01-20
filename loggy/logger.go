package loggy

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var logFile *os.File
var ECHO bool = false
var SILENT bool = false
var LogFolder string = "./logs/"

type Logger struct {
	logFile *os.File
	id      int
	app     string
}

var loggers map[int]*Logger
var app string

func Get(id int) *Logger {
	if loggers == nil {
		loggers = make(map[int]*Logger)
	}
	l, ok := loggers[id]
	if !ok {
		l = NewLogger(id, app)
		loggers[id] = l
	}
	return l
}

func NewLogger(id int, app string) *Logger {

	if app == "" {
		app = "dskalyzer"
	}

	filename := fmt.Sprintf("%s_%d_%s.log", app, id, fts())
	os.MkdirAll(LogFolder, 0755)

	logFile, _ = os.Create(LogFolder + filename)
	l := &Logger{
		id:      id,
		logFile: logFile,
		app:     app,
	}

	return l
}

func ts() string {
	t := time.Now()
	return fmt.Sprintf(
		"%.4d/%.2d/%.2d %.2d:%.2d:%.2d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
	)
}

func fts() string {
	t := time.Now()
	return fmt.Sprintf(
		"%.4d%.2d%.2d%.2d%.2d%.2d",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
	)
}

func (l *Logger) llogf(format string, designator string, v ...interface{}) {

	format = ts() + " " + designator + " :: " + format

	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	l.logFile.WriteString(fmt.Sprintf(format, v...))
	l.logFile.Sync()

	if ECHO {
		os.Stderr.WriteString(fmt.Sprintf(format, v...))
	}

}

func (l *Logger) llog(designator string, v ...interface{}) {

	format := ts() + " " + designator + " :: "
	for _, vv := range v {
		format += fmt.Sprintf("%v ", vv)
	}
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}

	l.logFile.WriteString(format)
	l.logFile.Sync()

	if ECHO {
		os.Stderr.WriteString(format)
	}
}

func (l *Logger) Logf(format string, v ...interface{}) {
	l.llogf(format, "INFO ", v...)
}

func (l *Logger) Log(v ...interface{}) {
	l.llog("INFO ", v...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.llogf(format, "ERROR", v...)
}

func (l *Logger) Error(v ...interface{}) {
	l.llog("ERROR", v...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.llogf(format, "DEBUG", v...)
}

func (l *Logger) Debug(v ...interface{}) {
	l.llog("DEBUG", v...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.llogf(format, "FATAL", v...)
}

func (l *Logger) Fatal(v ...interface{}) {
	l.llog("FATAL", v...)
}
