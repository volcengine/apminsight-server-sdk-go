package settings_fetcher

import (
	"fmt"
	"testing"
	"time"
)

func TestFetcher(t *testing.T) {
	f := NewSettingsFetcher(SettingsFetcherConfig{
		Service:  "server_a",
		Logger:   &l{},
		Schema:   "http",
		Host:     "0.0.0.0:8080",
		Timeout:  1 * time.Second,
		Notifier: nil,
	})
	f.Start()
	d, _ := f.getSettings()
	fmt.Printf("settings is %+v \n", d)
	time.Sleep(60 * time.Second)
	f.Stop()
}

type l struct{}

func (l *l) Debug(format string, args ...interface{}) {
	fmt.Printf("[Debug]"+format+"\n", args...)
}
func (l *l) Info(format string, args ...interface{}) {
	fmt.Printf("[Info]"+format+"\n", args...)
}
func (l *l) Error(format string, args ...interface{}) {
	fmt.Printf("[Error]"+format+"\n", args...)
}
