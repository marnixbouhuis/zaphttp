package zaphttp

import (
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type handler struct {
	options *handlerOptions
}

func NewHandler(opts ...HandlerOption) func(next http.Handler) http.Handler {
	h := &handler{
		options: buildHandlerOptions(opts...),
	}
	return h.Wrap
}

func (h *handler) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		h.handleRequest(w, req, next)
	})
}

func (h *handler) handleRequest(w http.ResponseWriter, req *http.Request, next http.Handler) {
	// Capture the request start time for logging how long a handler took.
	start := time.Now()

	// Build logger for this request.
	l := h.options.perRequestLoggerFn(h.options.logger, req)

	// Add trace information if tracing is configured.
	currentSpan := trace.SpanContextFromContext(req.Context())
	if currentSpan.IsValid() {
		fields := h.options.traceFormatter.GetTraceFields(req, currentSpan)
		l = l.With(fields...)
	}

	// Inject logger in the request context.
	req = injectLoggerInContext(req, l)

	// Wrap http.ResponseWriter so we can extract the status code from the response.
	sr := &statusRecorder{writer: w}

	var completed bool
	defer func() {
		if !completed {
			// next.ServeHTTP did not complete normally. We either panicked or runtime.Goexit() was called.
			// Do not recover the panic since this would mess with the stacktrace, just log it.
			h.logRequest(l, zapcore.ErrorLevel, "HTTP request panicked", req, &ResponseInfo{
				StatusCode:  sr.StatusCode,
				ContentType: sr.ContentType,
				Start:       start,
				Latency:     time.Since(start),
			})
		}
	}()

	h.logRequest(l, zapcore.DebugLevel, "Received HTTP request", req, &ResponseInfo{Start: start})

	next.ServeHTTP(sr, req)
	completed = true

	// Request handler finished, log the result.
	res := &ResponseInfo{
		StatusCode:  sr.StatusCode,
		ContentType: sr.ContentType,
		Start:       start,
		Latency:     time.Since(start),
	}

	if sr.StatusCode <= 399 {
		// Everything OK!
		h.logRequest(l, zapcore.InfoLevel, "HTTP request finished", req, res)
		return
	}

	if sr.StatusCode <= 499 {
		// Client side error.
		h.logRequest(l, zapcore.WarnLevel, "HTTP request failed due to a client error", req, res)
		return
	}

	// Other unknown code, likely a server error.
	h.logRequest(l, zapcore.ErrorLevel, "HTTP request failed", req, res)
}

func (h *handler) logRequest(l *zap.Logger, level zapcore.Level, msg string, req *http.Request, res *ResponseInfo) {
	if shouldLog := h.options.perRequestFilterFn(req, level); !shouldLog {
		return
	}

	if ce := l.Check(level, msg); ce != nil {
		fields := h.options.requestFormatter.GetRequestFields(req, res)
		ce.Write(fields...)
	}
}
