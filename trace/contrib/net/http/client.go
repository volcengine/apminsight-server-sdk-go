package http

import (
	"net/http"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type Config struct {
	clientServiceType   string
	clientServiceGetter func(req *http.Request) string
	operation           string
	tagsExtractor       func(req *http.Request) map[string]string // tags extracted from gin.Context will be set in span tags
}

type Option func(*Config)

func WithClientServiceType(svcType string) Option {
	return func(cfg *Config) {
		cfg.clientServiceType = svcType
	}
}

func WithClientServiceGetter(f func(req *http.Request) string) Option {
	return func(cfg *Config) {
		if f != nil {
			cfg.clientServiceGetter = f
		}
	}
}

func WithOperation(op string) Option {
	return func(cfg *Config) {
		cfg.operation = op
	}
}

func WithTagsExtractor(f func(req *http.Request) map[string]string) Option {
	return func(cfg *Config) {
		cfg.tagsExtractor = f
	}
}

func newDefaultConfig() *Config {
	return &Config{
		clientServiceType: aitracer.Http,
		clientServiceGetter: func(req *http.Request) string {
			if req.URL != nil {
				return req.URL.Host
			}
			return ""
		},
		operation: "http_call",
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
	span, ctx := rt.tracer.StartClientSpanFromContext(req.Context(), rt.cfg.operation,
		aitracer.ClientResourceAs(rt.cfg.clientServiceType, clientService, req.Method))

	if rt.cfg.tagsExtractor != nil {
		for k, v := range rt.cfg.tagsExtractor(req) {
			span.SetTagString(k, v)
		}
	}

	_ = rt.tracer.Inject(span.Context(), aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(req.Header))
	res, err = rt.base.RoundTrip(req.WithContext(ctx))
	if err != nil {
		span.SetTag(aitracer.HttpStatusCode, http.StatusInternalServerError)
		span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindExternalServiceError))
		span.FinishWithOption(aitracer.FinishSpanOption{
			Status: 1,
		})
	} else {
		span.SetTag(aitracer.HttpStatusCode, res.StatusCode)
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
