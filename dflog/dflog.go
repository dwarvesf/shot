// Package dflog wrap around logrus to supply capturing current line and function
package dflog

import (
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

// M is sort form for logrus.Fields
type M map[string]interface{}

// Fields is alias for logrus.Fields
type Fields map[string]interface{}

// Logger wraps logrus.Logger
type Logger struct {
	lg        *log.Logger
	DebugMode bool
}

// Level type
type Level uint8

// Convert the Level to a string. E.g. PanicLevel becomes "panic".
func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	case PanicLevel:
		return "panic"
	}

	return "unknown"
}

// AllLevels is constants exposing all logging levels
var AllLevels = []Level{
	PanicLevel,
	FatalLevel,
	ErrorLevel,
	WarnLevel,
	InfoLevel,
	DebugLevel,
}

// These are the different logging levels. You can set the logging level to log
// on your instance of logger, obtained with `logrus.New()`.
const (
	// PanicLevel level, highest level of severity. Logs and then calls panic with the
	// message passed to Debug, Info, ...
	PanicLevel Level = iota
	// FatalLevel level. Logs and then calls `os.Exit(1)`. It will exit even if the
	// logging level is set to Panic.
	FatalLevel
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)

// Entry wraps logrus.Entry
type Entry struct {
	*log.Entry
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stderr)
}

// New creates new Logger.
func New() Logger {
	logger := log.New()
	return Logger{logger, false}
}

// 2015-06-10 20:10:08.123456
func getTime() string {
	var buf [30]byte
	b := buf[:0]
	t := time.Now()
	year, month, day := t.Date()
	hour, min, sec := t.Clock()
	// nsec := t.Nanosecond()

	itoa(&b, year, 4)
	b = append(b, '-')
	itoa(&b, int(month), 2)
	b = append(b, '-')
	itoa(&b, day, 2)
	b = append(b, ' ')
	itoa(&b, hour, 2)
	b = append(b, ':')
	itoa(&b, min, 2)
	b = append(b, ':')
	itoa(&b, sec, 2)
	// b = append(b, '.')
	// itoa(&b, nsec/1e3, 6)

	return string(b)
}

// Taken from stdlib "log".
//
// Cheap integer to fixed-width decimal ASCII.  Give a negative width to avoid zero-padding.
func itoa(buf *[]byte, i int, wid int) {
	// Assemble decimal in reverse order.
	var b [20]byte
	bp := len(b) - 1
	for i >= 10 || wid > 1 {
		wid--
		q := i / 10
		b[bp] = byte('0' + i - q*10)
		bp--
		i = q
	}
	// i < 10
	b[bp] = byte('0' + i)
	*buf = append(*buf, b[bp:]...)
}

func trimFile(path string, line int) string {
	index := strings.Index(path, "/src/")
	if index < 0 {
		return path
	}
	return path[index+5:] + ":" + strconv.Itoa(line)
}

func getInfo() map[string]interface{} {
	_, path, line, _ := runtime.Caller(2)
	return map[string]interface{}{
		"*TIME": getTime(),
		"-FILE": trimFile(path, line),
	}
}

// SetFormatter sets formatter for logger.
func (l Logger) SetFormatter(formatter log.Formatter) {
	l.lg.Formatter = formatter
}

// SetLevel sets log level for logger.
func (l Logger) SetLevel(level log.Level) {
	l.lg.Level = level
}

// SetOutput sets output for logger.
func (l Logger) SetOutput(out io.Writer) {
	l.lg.Out = out
}

// WithField implements logrus WithField.
func (l Logger) WithField(key string, value interface{}) Entry {
	return Entry{l.lg.WithFields(getInfo()).WithField(key, value)}
}

// WithFields implements logrus WithFields.
func (l Logger) WithFields(fields map[string]interface{}) Entry {
	return Entry{l.lg.WithFields(getInfo()).WithFields(fields)}
}

// WithError implements logrus WithError.
func (l Logger) WithError(err error) Entry {
	return Entry{l.lg.WithFields(getInfo()).WithError(err)}
}

// WithStack implements logrus WithError with stacks.
func (l Logger) WithStack(err error) Entry {
	debug.PrintStack()
	return Entry{l.lg.WithFields(getInfo()).WithError(err)}
}

// Debugf implements logrus Debugf.
func (l Logger) Debugf(format string, args ...interface{}) {
	l.lg.WithFields(getInfo()).Debugf(format, args...)
}

// Infof implements logrus Infof.
func (l Logger) Infof(format string, args ...interface{}) {
	l.lg.WithFields(getInfo()).Infof(format, args...)
}

// Printf implements logrus Printf.
func (l Logger) Printf(format string, args ...interface{}) {
	l.lg.WithFields(getInfo()).Printf(format, args...)
}

// Warnf implements logrus Warnf.
func (l Logger) Warnf(format string, args ...interface{}) {
	l.lg.WithFields(getInfo()).Warnf(format, args...)
}

// Warningf implements logrus Warningf.
func (l Logger) Warningf(format string, args ...interface{}) {
	l.lg.WithFields(getInfo()).Warningf(format, args...)
}

// Errorf implements logrus Errorf.
func (l Logger) Errorf(format string, args ...interface{}) {
	l.lg.WithFields(getInfo()).Errorf(format, args...)
}

// Fatalf implements logrus Fatalf.
func (l Logger) Fatalf(format string, args ...interface{}) {
	l.lg.WithFields(getInfo()).Fatalf(format, args...)
}

// Panicf implements logrus Panicf.
func (l Logger) Panicf(format string, args ...interface{}) {
	l.lg.WithFields(getInfo()).Panicf(format, args...)
}

// Debug implements logrus Debug.
func (l Logger) Debug(args ...interface{}) {
	l.lg.WithFields(getInfo()).Debug(args...)
}

// Info implements logrus Info.
func (l Logger) Info(args ...interface{}) {
	l.lg.WithFields(getInfo()).Info(args...)
}

// Print implements logrus Print.
func (l Logger) Print(args ...interface{}) {
	l.lg.WithFields(getInfo()).Print(args...)
}

// Warn implements logrus Warn.
func (l Logger) Warn(args ...interface{}) {
	l.lg.WithFields(getInfo()).Warn(args...)
}

// Warning implements logrus Warning.
func (l Logger) Warning(args ...interface{}) {
	l.lg.WithFields(getInfo()).Warning(args...)
}

// Error implements logrus Error.
func (l Logger) Error(args ...interface{}) {
	l.lg.WithFields(getInfo()).Error(args...)
}

// Fatal implements logrus Fatal.
func (l Logger) Fatal(args ...interface{}) {
	l.lg.WithFields(getInfo()).Fatal(args...)
}

// Panic implements logrus Panic.
func (l Logger) Panic(args ...interface{}) {
	l.lg.WithFields(getInfo()).Panic(args...)
}

// Debugln implements logrus Debugln.
func (l Logger) Debugln(args ...interface{}) {
	l.lg.WithFields(getInfo()).Debugln(args...)
}

// Infoln implements logrus Infoln.
func (l Logger) Infoln(args ...interface{}) {
	l.lg.WithFields(getInfo()).Infoln(args...)
}

// Println implements logrus Println.
func (l Logger) Println(args ...interface{}) {
	l.lg.WithFields(getInfo()).Println(args...)
}

// Warnln implements logrus Warnln.
func (l Logger) Warnln(args ...interface{}) {
	l.lg.WithFields(getInfo()).Warnln(args...)
}

// Warningln implements logrus Warningln.
func (l Logger) Warningln(args ...interface{}) {
	l.lg.WithFields(getInfo()).Warningln(args...)
}

// Errorln implements logrus Errorln.
func (l Logger) Errorln(args ...interface{}) {
	l.lg.WithFields(getInfo()).Errorln(args...)
}

// Fatalln implements logrus Fatalln.
func (l Logger) Fatalln(args ...interface{}) {
	l.lg.WithFields(getInfo()).Fatalln(args...)
}

// Panicln implements logrus Panicln.
func (l Logger) Panicln(args ...interface{}) {
	l.lg.WithFields(getInfo()).Panicln(args...)
}

// Log implements logrus Panicln.
func (l Logger) Log(level Level, msg string, err error, fields Fields) {
	if l.DebugMode {
		switch level {
		case PanicLevel:
			l.WithFields(fields).WithError(err).Panic(msg)
		case FatalLevel:
			l.WithFields(fields).WithError(err).Fatal(msg)
		case ErrorLevel:
			l.WithFields(fields).WithError(err).Error(msg)
		case WarnLevel:
			l.WithFields(fields).WithError(err).Warn(msg)
		case InfoLevel:
			l.WithFields(fields).WithError(err).Info(msg)
		case DebugLevel:
			l.WithFields(fields).WithError(err).Debug(msg)
		}
	} else {
		switch level {
		case PanicLevel:
			l.Panic(msg)
		case FatalLevel:
			l.Fatal(msg)
		case ErrorLevel:
			l.Error(msg)
		case WarnLevel:
			l.Warn(msg)
		case InfoLevel:
			l.Info(msg)
		}
	}
}

// WithField implements logrus Entry WithField.
func (e Entry) WithField(key string, value interface{}) Entry {
	return Entry{e.Entry.WithField(key, value)}
}

// WithFields implements logrus Entry method WithFields.
func (e Entry) WithFields(fields map[string]interface{}) Entry {
	return Entry{e.Entry.WithFields(fields)}
}

// WithError implements logrus Entry method WithError.
func (e Entry) WithError(err error) Entry {
	return Entry{e.Entry.WithError(err)}
}
