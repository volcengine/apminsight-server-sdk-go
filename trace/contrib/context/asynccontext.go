package context

import (
	"context"

	"github.com/volcengine/apminsight-server-sdk-go/trace/aitracer"
)

// NewContextForAsyncTracing create a ctx for async work tracing.
// In order to trace, ctx must be passed in, while in async work load, ctx is passed in and then be canceled by parent, async work is canceled too.
// So we must create a new ctx with the same tracing-info but remove cancel ctx.
func NewContextForAsyncTracing(ctx context.Context) context.Context {
	span := aitracer.GetSpanFromContext(ctx)
	return aitracer.ContextWithSpan(context.Background(), span)
}
