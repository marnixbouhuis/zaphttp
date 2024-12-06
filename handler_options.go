package zaphttp

import (
	"net/http"

	"go.uber.org/zap"
)

type PerRequestLoggerFunc func(parent *zap.Logger, req *http.Request) *zap.Logger

func DefaultPerRequestLoggerFunc(parent *zap.Logger, _ *http.Request) *zap.Logger {
	return parent.Named("request")
}

type handlerOptions struct {
	logger             *zap.Logger
	perRequestLoggerFn PerRequestLoggerFunc
	traceFormatter     TraceFormatter
	requestFormatter   RequestFormatter
}

func defaultHandlerOptions() *handlerOptions {
	return &handlerOptions{
		logger:             zap.L(),
		perRequestLoggerFn: DefaultPerRequestLoggerFunc,
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
