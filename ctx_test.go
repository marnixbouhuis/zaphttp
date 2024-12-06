package zaphttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marnixbouhuis/zaphttp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestFromContext(t *testing.T) {
	t.Parallel()

	t.Run("should use the per request logger inside the request context", func(t *testing.T) {
		t.Parallel()

		// Create an observable logger to capture log output
		core, logs := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		requestLogger := zaphttp.NewHandler(
			zaphttp.WithLogger(logger),
			zaphttp.WithPerRequestLogger(func(parent *zap.Logger, _ *http.Request) *zap.Logger {
				return parent.With(zap.String("request_id", "test-123"))
			}),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		requestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get logger from context and log something
			l := zaphttp.FromContext(r.Context())
			l.Info("test message")

			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)

		// Verify that the log message contains our injected field
		assert.Equal(t, 2, logs.Len()) // One for the request log, one for our test message
		assert.Equal(t, "test message", logs.All()[0].Message)
		assert.Equal(t, "test-123", logs.All()[0].ContextMap()["request_id"])
	})

	//nolint:paralleltest // Can not run in parallel since we replace the global zap logger.
	t.Run("Should return working logger with warning when used outside request context", func(t *testing.T) {
		// This test can't run in parallel since we replace the global logger.
		// Keep a reference to the old global logger and restore in on test completion.
		oldGlobalLogger := zap.L()
		t.Cleanup(func() {
			zap.ReplaceGlobals(oldGlobalLogger)
		})

		// Create an observable logger to capture log output
		core, logs := observer.New(zapcore.DebugLevel)
		logger := zap.New(core)

		// Replace the global logger temporarily
		originalLogger := zap.L()
		zap.ReplaceGlobals(logger)
		defer zap.ReplaceGlobals(originalLogger)

		// Use FromContext with a context that doesn't have a logger
		ctx := context.Background()
		l := zaphttp.FromContext(ctx)

		// Log something to verify the logger works
		l.Info("test message")

		// Should have two log entries:
		// 1. Debug warning about using outside request context
		// 2. Our test message
		assert.Equal(t, 2, logs.Len())

		// Verify the warning message
		assert.Equal(t, zapcore.DebugLevel, logs.All()[0].Level)
		assert.Contains(t, logs.All()[0].Message, "FromContext is used outside of a HTTP request context")

		// Verify our test message got through
		assert.Equal(t, zapcore.InfoLevel, logs.All()[1].Level)
		assert.Equal(t, "test message", logs.All()[1].Message)
	})
}
