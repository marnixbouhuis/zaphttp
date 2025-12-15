package zaphttp

import (
	"net/http"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type PerRequestLoggerFunc func(parent *zap.Logger, req *http.Request) *zap.Logger

func DefaultPerRequestLoggerFunc(parent *zap.Logger, _ *http.Request) *zap.Logger {
	return parent.Named("request")
}

// PerRequestFilterFunc is a function that allows filtering out log messages for specific requests.
// The function should return true if the request should be logged, false otherwise.
type PerRequestFilterFunc func(req *http.Request, level zapcore.Level) bool

func DefaultPerRequestFilterFunc(_ *http.Request, _ zapcore.Level) bool {
	return true
}

type handlerOptions struct {
	logger             *zap.Logger
	perRequestLoggerFn PerRequestLoggerFunc
	perRequestFilterFn PerRequestFilterFunc
	traceFormatter     TraceFormatter
	requestFormatter   RequestFormatter
}

func defaultHandlerOptions() *handlerOptions {
	return &handlerOptions{
		logger:             zap.L(),
		perRequestLoggerFn: DefaultPerRequestLoggerFunc,
		perRequestFilterFn: DefaultPerRequestFilterFunc,
		traceFormatter:     DefaultFormatter,
		requestFormatter:   DefaultFormatter,
	}
}

type HandlerOption func(*handlerOptions)

func buildHandlerOptions(opts ...HandlerOption) *handlerOptions {
	options := defaultHandlerOptions()
	for _, fn := range opts {
		fn(options)
	}
	return options
}

func WithLogger(logger *zap.Logger) HandlerOption {
	return func(options *handlerOptions) {
		options.logger = logger
	}
}

func WithPerRequestLogger(fn PerRequestLoggerFunc) HandlerOption {
	return func(options *handlerOptions) {
		options.perRequestLoggerFn = fn
	}
}

// WithPerRequestFilter is an option that allows filtering out log messages for specific requests.
func WithPerRequestFilter(fn PerRequestFilterFunc) HandlerOption {
	return func(options *handlerOptions) {
		options.perRequestFilterFn = fn
	}
}

func WithTraceFormatter(f TraceFormatter) HandlerOption {
	return func(options *handlerOptions) {
		options.traceFormatter = f
	}
}

func WithRequestFormatter(f RequestFormatter) HandlerOption {
	return func(options *handlerOptions) {
		options.requestFormatter = f
	}
}
