package meshLog

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/fatih/color"
)

// LogLevel indicates the level of logging
type LogLevel int

const (
	debug LogLevel = iota
	info
	warn
	fatal
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 34
	gray    = 37
)

// Date / Time format
const dtFormat = "Jan 2 15:04:05"

// Logger is the wrapper around a given logging message
type Logger struct {
	envVar   string
	messages []loggedMessage
}

type loggedMessage struct {
	level   LogLevel
	message string
}

/**
 * String Logging
 */

// Write allows the logger to conform to io.Writer. Assume
// info level logging
func Write(data []byte) (int, error) {
	Info(string(data))
	return len(data), nil
}

// Debug is a convenience method appending a debug message to the logger
func Debug(obj interface{}) {
	// Get the line number and calling func sig
	_, fn, line, _ := runtime.Caller(1)
	msg := fmt.Sprintf("%+v\n%s:%d\n\n", obj, fn, line)
	formattedMessage := formattedLogMessage("DEBUG", msg)
	color.Green(formattedMessage)
}

// Info is a convenience method appending a info style message to the logger
func Info(obj interface{}) {
	// Get the line number and calling func sig
	_, fn, line, _ := runtime.Caller(1)
	msg := fmt.Sprintf("%+v\n%s:%d\n\n", obj, fn, line)
	formattedMessage := formattedLogMessage("INFO", msg)
	color.White(formattedMessage)
}

// Warn is a convenience method appending a warning message to the logger
func Warn(obj interface{}) {
	// Get the line number and calling func sig
	_, fn, line, _ := runtime.Caller(1)
	msg := fmt.Sprintf("%+v\n%s:%d\n\n", obj, fn, line)
	formattedMessage := formattedLogMessage("WARN", msg)
	color.Yellow(formattedMessage)
}

// Fatal is a convenience method appending a fatal message to the logger
func Fatal(obj interface{}) {
	// Get the line number and calling func sig
	_, fn, line, _ := runtime.Caller(1)
	msg := fmt.Sprintf("%+v\n%s:%d\n\n", obj, fn, line)
	formattedMessage := formattedLogMessage("ERROR", msg)
	color.Red(formattedMessage)
}

/**
 * Formatted Strings
 */

// Debugf is a convenience method appending a debug message to the logger
func Debugf(msg string, a ...interface{}) {
	_, fn, line, _ := runtime.Caller(1)
	msg = fmt.Sprintf(msg, a...)
	msg = fmt.Sprintf("%+v%s:%d\n\n", msg, fn, line)
	formattedMessage := formattedLogMessage("DEBUG", msg)
	color.Green(formattedMessage)
}

// Infof is a convenience method appending a info style message to the logger
func Infof(msg string, a ...interface{}) {
	_, fn, line, _ := runtime.Caller(1)
	msg = fmt.Sprintf(msg, a...)
	msg = fmt.Sprintf("%+v%s:%d\n\n", msg, fn, line)
	formattedMessage := formattedLogMessage("INFO", msg)
	color.White(formattedMessage)
}

// Warnf is a convenience method appending a warning message to the logger
func Warnf(msg string, a ...interface{}) {
	_, fn, line, _ := runtime.Caller(1)
	msg = fmt.Sprintf(msg, a...)
	msg = fmt.Sprintf("%+v%s:%d\n\n", msg, fn, line)
	formattedMessage := formattedLogMessage("WARN", msg)
	color.Yellow(formattedMessage)
}

// Fatalf is a convenience method appending a fatal message to the logger
func Fatalf(msg string, a ...interface{}) {
	_, fn, line, _ := runtime.Caller(1)
	msg = fmt.Sprintf(msg, a...)
	msg = fmt.Sprintf("%+v%s:%d\n\n", msg, fn, line)
	formattedMessage := formattedLogMessage("ERROR", msg)
	color.Red(formattedMessage)
}

/**
 * Internal Formatting
 */

func formattedLogMessage(level string, logMessage string) string {
	// Set ENB
	env := "LOCAL"
	if len(os.Getenv("ENV")) > 0 {
		env = strings.ToUpper(os.Getenv("ENV"))
	}

	return fmt.Sprintf("[%s] - %s: %s", env, level, logMessage)
}

func formatColoredMessage(message string, level LogLevel) string {
	var levelColor int
	switch level {
	case debug:
		levelColor = yellow
	case info:
		levelColor = gray
	case warn:
		levelColor = green
	case fatal:
		levelColor = red
	}

	// levelText := strings.ToUpper(message)[0:4]
	return fmt.Sprintf("\x1b[%dm%s\x1b", levelColor, message)
}

func stringValueForLogLevel(level LogLevel) string {
	switch level {
	case debug:
		return "DEBUG"
	case info:
		return "INFO"
	case warn:
		return "WARN"
	case fatal:
		return "FATAL"
	}
	return "INFO"
}

/**
 * Convenience for panic / err
 */

// Perror is Syntax Sugga for panicing on error
func Perror(err error) {
	if err != nil {
		Fatal(err)
		panic(err)
	}
}
