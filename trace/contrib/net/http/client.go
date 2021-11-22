package http

import (
	"net/http"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type roundTripper struct {
	tracer aitracer.Tracer
	base   http.RoundTripper
}

func (rt *roundTripper) RoundTrip(req *http.Request) (res *http.Response, err error) {
	if req == nil {
		return rt.base.RoundTrip(req)
	}
	clientService := "empty"
	if req.URL != nil {
		clientService = req.URL.Host
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

func WrapClient(c *http.Client, tracer aitracer.Tracer) *http.Client {
	if c.Transport == nil {
		c.Transport = http.DefaultTransport
	}
	c.Transport = &roundTripper{
		tracer: tracer,
		base:   c.Transport,
	}
	return c
}
