package sender

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aiprofiler/profile_models"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer/logger"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/agentless_adapter"
	"github.com/volcengine/apminsight-server-sdk-go/trace/internal/utils"
)

type Sender interface {
	Start()
	Stop()

	Send()
}

const (
	agentSockPath = "/profile_upload"
	collectorPath = "/server_collect/profile_collect"

	stopIntervalHeaderKey = "X-ByteAPM-Stop"
)

type Config struct {
	Sock string

	Schema  string
	Host    string
	Timeout time.Duration

	BackoffInterval time.Duration
	RetryCount      int

	Logger logger.Logger
}

type HTTPSender struct {
	logger logger.Logger

	client          *http.Client
	url             string
	backoffInterval time.Duration
	retryCount      int

	in chan *profile_models.ProfileInfo
	wg sync.WaitGroup
}

func NewSender(cfg Config, in chan *profile_models.ProfileInfo) Sender {
	return newHTTPSender(cfg, in) //currently only http is supported
}

func newHTTPSender(cfg Config, in chan *profile_models.ProfileInfo) Sender {
	var (
		c   *http.Client
		url string
	)
	if cfg.Sock != "" && cfg.Host == "" {
		url = utils.URLViaUDS(agentSockPath)
		c = utils.NewHTTPClientViaUDS(cfg.Sock, cfg.Timeout)
	} else {
		url = fmt.Sprintf("%s://%s/%s", cfg.Schema, cfg.Host, strings.TrimPrefix(collectorPath, "/"))
		c = &http.Client{Timeout: cfg.Timeout}
	}
	return &HTTPSender{
		logger:          cfg.Logger,
		client:          c,
		url:             url,
		backoffInterval: cfg.BackoffInterval,
		retryCount:      cfg.RetryCount,
		in:              in,
	}
}

func (s *HTTPSender) Start() {
	s.wg.Add(1)
	go func() {
		defer func() {
			s.wg.Done()
		}()
		s.sendLoop()
	}()
}

func (s *HTTPSender) Stop() {
	close(s.in)
	s.wg.Wait()
}

// Send is an empty method to distinguish from other interfaces
func (s *HTTPSender) Send() {}

func (s *HTTPSender) sendLoop() {
	for {
		select {
		case item, ok := <-s.in:
			if !ok {
				return
			}

			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			writeFiled(writer, item.UploadInfo)
			for idx, data := range item.MultiData {
				_ = writeFile(writer, fmt.Sprintf("file_%d", idx), data)
			}

			// Important
			//if you do not close the multipart writer you will not have a terminating boundary
			_ = writer.Close()

			req, err := http.NewRequest(http.MethodPost, s.url, body)
			if err != nil {
				s.logger.Error("[sendLoop] new request fail. err=%+v", err)
				continue
			}
			req.Header.Set("Content-Type", writer.FormDataContentType())
			req.Header.Set(agentless_adapter.AppKey, agentless_adapter.GetAppKey())

			for i := 0; i <= s.retryCount; i++ {
				succ, stopDuraiton := s.sendRequest(req)
				if stopDuraiton != 0 && s.retryCount > 0 {
					time.Sleep(stopDuraiton)
				}
				if succ {
					break
				}
			}
		}
	}
}

func writeFile(w *multipart.Writer, fieldName string, file []byte) error {
	if file == nil {
		return nil
	}
	fw, err := w.CreateFormFile(fieldName, fieldName)
	if err != nil {
		return err
	}
	_, err = io.Copy(fw, bytes.NewBuffer(file))
	if err != nil {
		return err
	}
	return nil
}

func writeFiled(w *multipart.Writer, info *profile_models.UploadInfo) {
	_ = w.WriteField("service_name", info.ServiceName)
	_ = w.WriteField("profile_id", strconv.FormatInt(info.ProfileId, 10))
	_ = w.WriteField("upload_id", info.UploadId)
	_ = w.WriteField("hostname", info.Hostname)
	_ = w.WriteField("container_id", info.ContainerId)
	_ = w.WriteField("instance_id", info.InstanceId)
	_ = w.WriteField("record_name", info.RecordName)
	_ = w.WriteField("runtime_type", info.RuntimeType)
	_ = w.WriteField("format", info.Format)
	_ = w.WriteField("start_time", strconv.FormatInt(info.StartTime, 10))
	_ = w.WriteField("end_time", strconv.FormatInt(info.EndTime, 10))
	_ = w.WriteField("pt_config_list", info.PtConfigList)
	_ = w.WriteField("go_runtime_info", info.GoRuntimeInfo)
	_ = w.WriteField("cpu_limit", strconv.FormatInt(info.CpuLimit, 10))
	_ = w.WriteField("process_cpu_usage_ratio", strconv.FormatFloat(info.ProcessCpuUsageRatio, 'f', 6, 64))
	_ = w.WriteField("process_mem_rss_ratio", strconv.FormatFloat(info.ProcessMemRssRatio, 'f', 6, 64))
}

func (s *HTTPSender) sendRequest(request *http.Request) (bool, time.Duration) {
	response, err := s.client.Do(request)
	if err != nil || response == nil {
		s.logger.Error("forward send http response. err=%+v", err)
		return false, s.backoffInterval
	}

	s.logger.Info("[profiler] forward send http response code %d logid %s", response.StatusCode, response.Header.Get("x-tt-logid"))

	defer response.Body.Close()
	_, err = ioutil.ReadAll(response.Body)
	if err != nil {
		return false, s.backoffInterval
	}
	stopDuration := getStopDuration(response)
	return true, stopDuration
}

func getStopDuration(response *http.Response) time.Duration {
	stopMinuteStr := response.Header.Get(stopIntervalHeaderKey)
	if stopMinuteStr != "" {
		stopMinute, _ := strconv.ParseInt(stopMinuteStr, 10, 64)
		return time.Duration(stopMinute) * time.Minute
	}
	return 0
}
