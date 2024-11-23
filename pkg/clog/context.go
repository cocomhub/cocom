package clog

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

type ctxKey string

var (
	KeyTraceID    ctxKey = "trace_id"
	KeyDebug      ctxKey = "debug"
	KeyXRequestID ctxKey = "X-Request-ID"
)

// NewTraceContext 创建带有追踪 ID 的上下文
func NewTraceContext(name string) context.Context {
	ctx := context.Background()
	ctx = context.WithValue(ctx, KeyTraceID, name)
	ctx = context.WithValue(ctx, KeyDebug, IsDebug())
	ctx = context.WithValue(ctx, KeyXRequestID, name)
	return ctx
}

// NewTraceCtx 是 NewTraceContext 的别名，保持向后兼容
var NewTraceCtx = NewTraceContext

// WithTraceID 为上下文添加追踪 ID
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, KeyXRequestID, traceID)
}

// GetTraceID 从上下文获取追踪 ID
func GetTraceID(ctx context.Context) string {
	if val, ok := ctx.Value(KeyXRequestID).(string); ok {
		return val
	}

	span := trace.SpanContextFromContext(ctx)
	if span.HasTraceID() {
		return span.TraceID().String()
	}
	return ""
}
