package zaphttp

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

type contextKey string

const (
	loggerContextKey contextKey = "logger"
)

func injectLoggerInContext(req *http.Request, l *zap.Logger) *http.Request {
	ctx := context.WithValue(req.Context(), loggerContextKey, l)
	return req.WithContext(ctx)
}

func FromContext(ctx context.Context) *zap.Logger {
	l, ok := ctx.Value(loggerContextKey).(*zap.Logger)
	if !ok {
		// Logger is not injected in the context, use the default global logger
		l = zap.L()
		l.Debug("FromContext is used outside of a HTTP request context. Make sure the HTTP handler is wrapped in a logging handler.")
	}
	return l
}
