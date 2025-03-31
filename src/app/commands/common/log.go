package app_commands_common

import (
	lib_debug "application/src/lib/debug"
	"fmt"
)

type Logger struct {
	prefix  string
	isDebug bool
}

func NewLogger(prefix string, isDebug bool) *Logger {
	return &Logger{
		prefix:  prefix,
		isDebug: isDebug,
	}
}
func (logger *Logger) Debug(msg string, args ...interface{}) {
	if logger.isDebug {
		lib_debug.Info("%s %s", logger.prefix, fmt.Sprintf(msg, args...))
	} else {
		lib_debug.Debug("%s %s", logger.prefix, fmt.Sprintf(msg, args...))
	}
}
func (logger *Logger) Info(msg string, args ...interface{}) {
	lib_debug.Info("%s %s", logger.prefix, fmt.Sprintf(msg, args...))
}
func (logger *Logger) Error(msg string, args ...interface{}) {
	lib_debug.Error("%s %s", logger.prefix, fmt.Sprintf(msg, args...))
}
