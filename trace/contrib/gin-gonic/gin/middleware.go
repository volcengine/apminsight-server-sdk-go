package gin

import (
	"context"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

type Config struct {
	ignoreRequest  func(c *gin.Context) bool              // request to be ignored when tracing. requests will still be processed by handler but no tracing will be recorded
	pathNormalizer func(escapedPath string) string        // getting resource from path needs to decrease cardinality
	tagsExtractor  func(c *gin.Context) map[string]string // tags extracted from gin.Context will be set in span tags
}

type Option func(*Config)

func WithIgnoreRequest(f func(c *gin.Context) bool) Option {
	return func(cfg *Config) {
		if f != nil {
			cfg.ignoreRequest = f
		}
	}
}

func WithPathNormalizer(f func(escapedPath string) string) Option {
	return func(cfg *Config) {
		if f != nil {
			cfg.pathNormalizer = f
		}
	}
}

func WithTagsExtractor(f func(c *gin.Context) map[string]string) Option {
	return func(cfg *Config) {
		if f != nil {
			cfg.tagsExtractor = f
		}
	}
}

func newDefaultConfig() *Config {
	return &Config{
		pathNormalizer: defaultPathNormalizer,
	}
}

func NewMiddleware(tracer aitracer.Tracer, opts ...Option) gin.HandlerFunc {
	if tracer == nil {
		panic("tracer is nil")
	}
	return func(c *gin.Context) {
		cfg := newDefaultConfig()
		for _, opt := range opts {
			opt(cfg)
		}

		// these requests will not be traced
		if cfg.ignoreRequest != nil && cfg.ignoreRequest(c) {
			return
		}

		// when pathNotFound, use path as resourceName
		resourceName := "unknown"
		if c.FullPath() != "" {
			resourceName = c.FullPath()
		} else if c.Request != nil && c.Request.URL != nil && c.Request.URL.Path != "" {
			resourceName = cfg.pathNormalizer(c.Request.URL.Path) // pathNormalizer is never nil
		}

		chainSpanContext, _ := tracer.Extract(aitracer.HTTPHeaders, aitracer.HTTPHeadersCarrier(c.Request.Header))
		span := tracer.StartServerSpan("request", aitracer.ChildOf(chainSpanContext), aitracer.ServerResourceAs(resourceName))
		spanContext := span.Context()
		c.Request = c.Request.WithContext(aitracer.ContextWithSpan(c.Request.Context(), span))
		c.Writer.Header().Add("x-trace-id", spanContext.TraceID())

		span.SetTag(aitracer.HttpMethod, c.Request.Method)
		if c.Request.URL != nil {
			span.SetTag(aitracer.HttpScheme, c.Request.URL.Scheme)
			span.SetTag(aitracer.HttpHost, c.Request.URL.Host)
			span.SetTag(aitracer.HttpPath, c.Request.URL.Path)
		}
		// set custom tags
		if cfg.tagsExtractor != nil {
			for k, v := range cfg.tagsExtractor(c) {
				span.SetTag(k, v)
			}
		}

		// Finish should be called directly by defer
		defer span.Finish()

		isPanic := true
		defer func() {
			status := c.Writer.Status()
			if isPanic {
				status = http.StatusInternalServerError //trace middle is executed before gin.defaultHandleRecovery
			}
			// set statusCode. statusCode will display on custom filters
			span.SetTag(aitracer.HttpStatusCode, status)
			// distinguish status and statusCode. status is always 0 or 1, and 1 indicates error
			if status >= http.StatusBadRequest {
				span.SetStatus(aitracer.StatusCodeError)
			}
			for _, err := range c.Errors {
				span.RecordError(err, aitracer.WithErrorKind(aitracer.ErrorKindBusinessError))
			}
		}()

		c.Next()
		isPanic = false
	}
}

// NewGinContextAdapter is used to run logrus/trace with gin.Context() rather than gin.Context.Request.Context(), however this will not work when ctx is wrapped (such as kitex)
// Deprecated
// Recommended solution:
// 1. Use gin.Context.Request.Context() when tracing
// 2. For gin version>=1.8.1, set engin.ContextWithFallback=true, which solve this problem perfectly.
func NewGinContextAdapter() func(context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		if ctx == nil {
			return nil
		}
		if c, ok := ctx.(*gin.Context); ok { // when ctx is wrapped, this solution fails
			return c.Request.Context()
		}
		return ctx
	}
}

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
