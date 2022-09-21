package settings_fetcher

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/agentless_adapter"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/settings_fetcher/settings_models"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/utils"
)

const (
	agentSockPath = "/settings"
	collectorPath = "/server_collect/settings"
)

type SettingsFetcherConfig struct {
	Service string
	Logger  logger.Logger

	Sock    string
	Schema  string
	Host    string
	Timeout time.Duration

	Notifier []func(*settings_models.Settings)
}

type Fetcher struct {
	service string
	client  *http.Client
	url     string
	logger  logger.Logger

	notifier []func(*settings_models.Settings)

	oldSettings *settings_models.Settings

	wg sync.WaitGroup

	closeChan chan struct{}

	fetchLock sync.Mutex
}

func NewSettingsFetcher(config SettingsFetcherConfig) *Fetcher {
	if config.Logger == nil {
		config.Logger = &logger.NoopLogger{}
	}
	var (
		c   *http.Client
		url string
	)
	if config.Sock != "" && config.Host == "" {
		if config.Timeout <= 0 {
			config.Timeout = 500 * time.Millisecond
		}
		url = utils.URLViaUDS(agentSockPath)
		c = utils.NewHTTPClientViaUDS(config.Sock, config.Timeout)
	} else {
		if config.Timeout <= 0 {
			config.Timeout = 5 * time.Second
		}
		url = fmt.Sprintf("%s://%s/%s", config.Schema, config.Host, strings.TrimPrefix(collectorPath, "/"))
		c = &http.Client{Timeout: config.Timeout}
	}

	f := &Fetcher{
		service:   config.Service,
		client:    c,
		url:       url,
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
				return
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
	req, err := http.NewRequest(http.MethodGet, f.url, nil)
	if err != nil || req == nil {
		return settings_models.Settings{}, err
	}
	req.Header.Add("X-ByteAPM-Service", f.service)
	req.Header.Add("service", f.service) // for compatibility
	req.Header.Add(agentless_adapter.AppKey, agentless_adapter.GetAppKey())
	resp, err := f.client.Do(req)
	if err != nil || resp == nil {
		f.logger.Error("[getSettings] http fail. err=%+v", err)
		return settings_models.Settings{}, err
	}
	if resp.StatusCode != http.StatusOK {
		f.logger.Error("[getSettings] http fail. statusCode=%+v", resp.StatusCode)
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
