package service_register

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/agentless_adapter"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/service_register/register_utils"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/utils"
)

const (
	agentSockPath = "/service_register"
	collectorPath = "/server_collect/service_register"
)

type Config struct {
	Sock string

	Schema string
	Host   string

	Interval time.Duration
	Timeout  time.Duration

	Logger logger.Logger
}

type Register struct {
	instanceId string

	url string

	service     string
	serviceType string

	logger   logger.Logger
	interval time.Duration

	client   *http.Client
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewRegister(serviceType, service string, config Config) *Register {
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

	return &Register{
		instanceId:  register_utils.GetInstanceID(),
		url:         url,
		service:     service,
		serviceType: serviceType,
		logger:      config.Logger,
		interval:    config.Interval,
		client:      c,
		stopChan:    make(chan struct{}),
	}
}

func (r *Register) Start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.runLoop()
	}()
}

func (r *Register) Stop() {
	close(r.stopChan)
	r.wg.Wait()
}

func (r *Register) runLoop() {
	r.register()

	t := time.NewTicker(r.interval)
	defer func() {
		t.Stop()
	}()
	for {
		select {
		case <-t.C:
			r.register()
		case <-r.stopChan:
			return
		}
	}
}

func (r *Register) register() {
	info, err := register_utils.GetInfo()
	if err != nil {
		r.logger.Error("get register info err %v", err)
		return
	}
	r.logger.Debug("register info %+v", info)
	// send docker id pid service name
	req, err := http.NewRequest(http.MethodGet, r.url, nil)
	if err != nil {
		r.logger.Error("send service info to agent err %v", err)
		return
	}
	q := req.URL.Query()
	q.Add("hostname", agentless_adapter.GetHostname())
	q.Add("pid", strconv.Itoa(info.Pid))
	q.Add("start_time", strconv.FormatInt(info.StartTime, 10))
	q.Add("service", r.service)
	q.Add("service_type", r.serviceType)
	q.Add("runtime_type", internal.RuntimeTypeGo)
	q.Add("runtime_bearer", agentless_adapter.GetRuntimeBearer())
	if info.ContainerId != "" {
		q.Add("container_id", info.ContainerId)
	}
	if info.Cmdline != "" {
		q.Add("full_cmd", info.Cmdline)
	}
	if r.instanceId != "" {
		q.Add("instance_id", r.instanceId)
	}

	req.URL.RawQuery = q.Encode()
	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Error("send service info to agent err %v", err)
		return
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		r.logger.Error("send service info to agent err %v", err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		r.logger.Error("send service info to agent error, http code %d, message %s", resp.StatusCode, string(bodyBytes))
		return
	}
}
