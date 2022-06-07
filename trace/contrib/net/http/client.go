package http

import (
	"net/http"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type Config struct {
	clientServiceGetter func(req *http.Request) string
}

type Option func(*Config)

func WithClientServiceGetter(f func(req *http.Request) string) Option {
	return func(cfg *Config) {
		if f != nil {
			cfg.clientServiceGetter = f
		}
	}
}

func newDefaultConfig() *Config {
	return &Config{
		clientServiceGetter: func(req *http.Request) string {
			if req.URL != nil {
				return req.URL.Host
			}
			return ""
		},
	}
}

type roundTripper struct {
	cfg    *Config
	tracer aitracer.Tracer
	base   http.RoundTripper
}

func (rt *roundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	if req == nil {
		return rt.base.RoundTrip(req)
	}
	clientService := "empty"
	if cs := rt.cfg.clientServiceGetter(req); cs != "" {
		clientService = cs
	}
	span, ctx := rt.tracer.StartClientSpanFromContext(req.Context(), "http_call",
		aitracer.ClientResourceAs(aitracer.Http, clientService, req.Method))
	_ = rt.tracer.Inject(span.Context(), aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(req.Header))
	res, err = rt.base.RoundTrip(req.WithContext(ctx))
	if err != nil {
		span.FinishWithOption(aitracer.FinishSpanOption{
			Status: 1,
		})
	} else {
		if res.StatusCode == http.StatusOK {
			span.Finish()
		} else {
			span.FinishWithOption(aitracer.FinishSpanOption{
				Status: int64(res.StatusCode),
			})
		}
	}
	return res, err
}

func WrapClient(c *http.Client, tracer aitracer.Tracer, opts ...Option) *http.Client {
	if c.Transport == nil {
		c.Transport = http.DefaultTransport
	}
	cfg := newDefaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}
	c.Transport = &roundTripper{
		cfg:    cfg,
		tracer: tracer,
		base:   c.Transport,
	}
	return c
}
