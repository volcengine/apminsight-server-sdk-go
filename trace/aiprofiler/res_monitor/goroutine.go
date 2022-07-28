package res_monitor

import "runtime"

type GoRoutineMonitor struct {
	goRoutineNum int64
}

func NewGoRoutineMonitor() *GoRoutineMonitor {
	return &GoRoutineMonitor{}
}

// GetGoRoutineNum gets the current goRoutine number. Be aware this method is not cached.
func (m *GoRoutineMonitor) GetGoRoutineNum() int64 {
	m.goRoutineNum = int64(runtime.NumGoroutine())
	return m.goRoutineNum
}
