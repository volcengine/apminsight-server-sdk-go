package settings_fetcher

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"bytes"

	"sync"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/settings_fetcher/settings_models"
)

const (
	path    = "settings"
	network = "unix"
)

type SettingsFetcherConfig struct {
	Service  string
	Logger   Logger
	Sock     string
	Notifier []func(*settings_models.Settings)
}

type Fetcher struct {
	service string
	client  *http.Client
	logger  logger.Logger

	notifier []func(*settings_models.Settings)

	oldSettings *settings_models.Settings

	wg sync.WaitGroup

	closeChan chan struct{}

	fetchLock sync.Mutex
}

func NewSettingsFetcher(config SettingsFetcherConfig) *Fetcher {
	if config.Sock == "" {
		panic("sock address is empty")
	}
	if config.Logger == nil {
		config.Logger = &logger.NoopLogger{}
	}
	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, network, config.Sock)
			},
		},
		Timeout: 100 * time.Millisecond,
	}
	f := &Fetcher{
		service:   config.Service,
		client:    &c,
		logger:    config.Logger,
		notifier:  config.Notifier,
		closeChan: make(chan struct{}),
	}
	return f
}

func (f *Fetcher) Start() {
	f.refreshSettings()
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		t := time.NewTicker(time.Second * 30)
		defer func() {
			t.Stop()
		}()
		for {
			select {
			case <-t.C:
				f.refreshSettings()
			case <-f.closeChan:
				break
			}
		}
	}()
}

func (f *Fetcher) Stop() {
	close(f.closeChan)
	f.wg.Wait()
}

func (f *Fetcher) refreshSettings() {
	settings, err := f.getSettings()
	if err != nil {
		f.logger.Error("[refreshSettings] get settings error %v", err)
		return
	}
	f.fetchLock.Lock()
	defer func() {
		f.fetchLock.Unlock()
	}()
	if f.oldSettings != nil {
		o, _ := f.oldSettings.Marshal()
		n, _ := settings.Marshal()
		if bytes.Equal(o, n) {
			f.logger.Debug("[refreshSettings] get same settings")
			return
		}
	}
	f.oldSettings = &settings
	for _, nf := range f.notifier {
		nf(&settings)
	}
}

func (f *Fetcher) getSettings() (settings_models.Settings, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s/%s", network, path), nil)
	if err != nil || req == nil {
		return settings_models.Settings{}, err
	}
	req.Header.Add("service", f.service)
	resp, err := f.client.Do(req)
	if err != nil {
		f.logger.Error("[getSettings] http fail. err=%+v", err)
		return settings_models.Settings{}, err
	}

	rawData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		f.logger.Error("[getSettings] read body fail. err=%+v", err)
		return settings_models.Settings{}, err
	}

	settings := settings_models.Settings{}
	err = settings.Unmarshal(rawData)
	if err != nil {
		f.logger.Error("[getSettings] unmarshal fail. err=%+v", err)
		return settings_models.Settings{}, err
	}

	f.logger.Info("[getSettings] success. settings=%s", settings.String())
	return settings, nil
}

type Logger interface {
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}
