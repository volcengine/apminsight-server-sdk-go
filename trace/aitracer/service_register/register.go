package service_register

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/service_register/register_utils"
)

const (
	network = "unix"
)

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

func NewRegister(service, serviceType, instanceId, sock string, interval time.Duration, logger logger.Logger) *Register {
	url := fmt.Sprintf("http://%s/service_register", network)
	c := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				dialer := net.Dialer{}
				return dialer.DialContext(ctx, network, sock)
			},
		},
		Timeout: 1 * time.Second,
	}
	return &Register{
		instanceId:  instanceId,
		url:         url,
		service:     service,
		serviceType: serviceType,
		logger:      logger,
		interval:    interval,
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
	q.Add("pid", strconv.Itoa(info.Pid))
	q.Add("start_time", strconv.FormatInt(info.StartTime, 10))
	q.Add("service", r.service)
	q.Add("service_type", r.serviceType)
	q.Add("runtime_type", "Go")
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
