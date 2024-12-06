package zaphttp

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type ResponseInfo struct {
	StatusCode  int
	ContentType string
	Start       time.Time
	Latency     time.Duration
}

type TraceFormatter interface {
	GetTraceFields(req *http.Request, spanCtx trace.SpanContext) []zap.Field
}

type RequestFormatter interface {
	GetRequestFields(req *http.Request, res *ResponseInfo) []zap.Field
}

type Formatter interface {
	TraceFormatter
	RequestFormatter
}

var DefaultFormatter = ElasticCommonSchemaFormatter
