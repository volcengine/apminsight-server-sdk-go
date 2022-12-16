package http

import (
	"net/http"
	"strings"
	"unicode"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type Config struct {
	clientServiceType    string
	clientServiceGetter  func(req *http.Request) string
	clientResourceGetter func(req *http.Request) string
	operation            string
	tagsExtractor        func(req *http.Request) map[string]string // tags extracted from gin.Context will be set in span tags
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

func WithClientResourceGetter(f func(req *http.Request) string) Option {
	return func(cfg *Config) {
		if f != nil {
			cfg.clientResourceGetter = f
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
		clientResourceGetter: func(req *http.Request) string {
			if req.URL != nil {
				return defaultPathNormalizer(req.URL.Path)
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

	clientResource := req.Method
	if cr := rt.cfg.clientResourceGetter(req); cr != "" {
		clientResource = cr
	}
	span, ctx := rt.tracer.StartClientSpanFromContext(req.Context(), rt.cfg.operation,
		aitracer.ClientResourceAs(rt.cfg.clientServiceType, clientService, clientResource))

	span.SetTagString(aitracer.HttpMethod, req.Method)
	if req.URL != nil {
		span.SetTagString(aitracer.HttpScheme, req.URL.Scheme)
		span.SetTagString(aitracer.HttpHost, req.URL.Host)
		span.SetTagString(aitracer.HttpPath, req.URL.Path)
	}

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

// copied from gin-gonic/gin
//defaultPathNormalizer aggregates path to patterns
/*
1. replace all digits
	/ 1 -> /?
	/ 11 -> /?
2. replace segments with mixed-characters
	"/a1/v2" ->  "/?/v2"
	"/ABC/av-1/b_2/c.3/d4d/v5f/v699/7"  -> "/ABC/?/?/?/?/?/?/?"
*/
func defaultPathNormalizer(escapedPath string) string {
	if len(escapedPath) == 0 {
		return "/"
	}

	findSplitters := func(escapedPath string) []int {
		positions := make([]int, 0)
		for idx := range escapedPath {
			if escapedPath[idx] == '/' {
				positions = append(positions, idx)
			}
		}
		if escapedPath[len(escapedPath)-1] != '/' {
			positions = append(positions, len(escapedPath))
		}
		return positions
	}

	hasNumber := func(escapedPath string) bool {
		hasNumeric := false
		for idx := range escapedPath {
			hasNumeric = unicode.IsDigit(rune(escapedPath[idx]))
			if hasNumeric {
				break
			}
		}
		return hasNumeric
	}

	splitPositions := findSplitters(escapedPath)

	sb := strings.Builder{}
	start := 0
	for _, end := range splitPositions {
		if start < end {
			sb.WriteRune('/')
			segLen := end - start
			if segLen > 1 && segLen <= 3 {
				if escapedPath[start] == 'v' || escapedPath[start] == 'V' { // reserve version identifiers, v1 v2, etc
					isVersionNum := true
					for j := start + 1; j < end; j++ {
						isVersionNum = isVersionNum && unicode.IsDigit(rune(escapedPath[j]))
					}
					if isVersionNum || !hasNumber(escapedPath[start:end]) {
						sb.WriteString(escapedPath[start:end])
					} else {
						sb.WriteRune('?')
					}
				} else { // abc jk1
					if hasNumber(escapedPath[start:end]) {
						sb.WriteRune('?')
					} else {
						sb.WriteString(escapedPath[start:end])
					}
				}
			} else if segLen > 3 && segLen <= 24 { // trans mixed to ?
				if hasNumber(escapedPath[start:end]) {
					sb.WriteRune('?')
				} else {
					sb.WriteString(escapedPath[start:end])
				}
			} else { //len is greater than 24
				sb.WriteRune('?')
			}
		}
		start = end + 1
	}
	if sb.Len() == 0 {
		return "/"
	}
	return sb.String()
}
