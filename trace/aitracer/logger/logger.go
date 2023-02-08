package logger

import "fmt"

type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}

type NoopLogger struct{}

func (l *NoopLogger) Debug(format string, args ...interface{}) {}
func (l *NoopLogger) Info(format string, args ...interface{})  {}
func (l *NoopLogger) Error(format string, args ...interface{}) {}

type DebugLogger struct{}

func (l *DebugLogger) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug]"+format+"\n", args...)
}
func (l *DebugLogger) Info(format string, args ...interface{}) {
	fmt.Printf("[Info]"+format+"\n", args...)
}
func (l *DebugLogger) Error(format string, args ...interface{}) {
	fmt.Printf("[Error]"+format+"\n", args...)
}
