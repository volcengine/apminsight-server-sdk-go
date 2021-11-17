package logrus

import (
	"github.com/sirupsen/logrus"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

func NewHook(tracer aitracer.Tracer, levels []logrus.Level) logrus.Hook {
	return &Hook{
		tracer: tracer,
		levels: levels,
	}
}

type Hook struct {
	tracer aitracer.Tracer
	levels []logrus.Level
}

func (h *Hook) Levels() []logrus.Level {
	return h.levels
}

func (h *Hook) Fire(e *logrus.Entry) error {
	if e == nil {
		return nil
	}
	logData := aitracer.LogData{
		Message:   []byte(e.Message),
		Timestamp: e.Time,
	}
	switch e.Level {
	case logrus.TraceLevel:
		logData.LogLevel = aitracer.LogLevelTrace
	case logrus.DebugLevel:
		logData.LogLevel = aitracer.LogLevelDebug
	case logrus.InfoLevel:
		logData.LogLevel = aitracer.LogLevelInfo
	case logrus.WarnLevel:
		logData.LogLevel = aitracer.LogLevelWarn
	case logrus.ErrorLevel:
		logData.LogLevel = aitracer.LogLevelError
	case logrus.FatalLevel:
		logData.LogLevel = aitracer.LogLevelFatal
	}
	if e.Caller != nil {
		logData.FileName = e.Caller.File
		logData.FileLine = int64(e.Caller.Line)
	}
	logData.Source = "logrus"
	h.tracer.Log(e.Context, logData)
	return nil
}
