package zaphttp

import (
	"net/http"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type noopFormatter struct{}

var NoopFormatter Formatter = &noopFormatter{}

func (*noopFormatter) GetTraceFields(_ *http.Request, _ trace.SpanContext) []zap.Field {
	return nil
}

func (*noopFormatter) GetRequestFields(_ *http.Request, _ *ResponseInfo) []zap.Field {
	return nil
}
