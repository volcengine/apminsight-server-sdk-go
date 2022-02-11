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

/*
	depth is used to indicate how many times logrus is wrapped
	eg:
	func LogWrapper(){
		logrus.Info()
	}
	LogWrapper is used in code to logging. In this case, depth=1 should be set to get the real position where LogWrapper is called.
	If unset, position reported in logrus is always inside LogWrapper, irrelate to called place.
	If set, logrus.Info() can not be used directly on your code as incorrect position will be reported
*/

func NewHookWithDepth(tracer aitracer.Tracer, levels []logrus.Level, depth int) logrus.Hook {
	return &Hook{
		tracer: tracer,
		levels: levels,
		depth:  depth,
	}
}

type Hook struct {
	tracer aitracer.Tracer
	levels []logrus.Level
	depth  int
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

	if h.depth > 0 {
		if c := h.getCallerWithDepth(); c != nil {
			logData.FileName = c.File
			logData.FileLine = int64(c.Line)
		}
	} else if e.Caller != nil {
		logData.FileName = e.Caller.File
		logData.FileLine = int64(e.Caller.Line)
	}

	logData.Source = "logrus"
	h.tracer.Log(e.Context, logData)
	return nil
}
