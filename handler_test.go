package zaphttp_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marnixbouhuis/zaphttp"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewHandler(t *testing.T) {
	t.Parallel()

	t.Run("Should send an extra log message at the beginning of the request if debug log level is enabled", func(t *testing.T) {
		t.Parallel()

		// Create a logger with debug level. This will cause the extra message to appear.
		core, logs := observer.New(zapcore.DebugLevel)
		logger := zap.New(core)

		requestLogger := zaphttp.NewHandler(
			zaphttp.WithLogger(logger),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)

		// Make sure we logged the request
		lines := logs.All()

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Len(t, lines, 2)

		assert.Equal(t, zapcore.DebugLevel, lines[0].Level)
		assert.Equal(t, "Received HTTP request", lines[0].Message)
		assert.Equal(t, zapcore.InfoLevel, lines[1].Level)
		assert.Equal(t, "HTTP request finished", lines[1].Message)
	})

	t.Run("Emit the right log line for each status code", func(t *testing.T) {
		t.Parallel()

		for code := 100; code <= 599; code++ {
			t.Run(fmt.Sprintf("HTTP code %d", code), func(t *testing.T) {
				t.Parallel()

				core, logs := observer.New(zapcore.InfoLevel)
				logger := zap.New(core)

				requestLogger := zaphttp.NewHandler(
					zaphttp.WithLogger(logger),
				)

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				rec := httptest.NewRecorder()

				requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(code)
				})).ServeHTTP(rec, req)

				// Make sure we logged the request
				lines := logs.All()

				assert.Equal(t, code, rec.Code)
				assert.Len(t, lines, 1)

				switch {
				case code <= 399:
					assert.Equal(t, zapcore.InfoLevel, lines[0].Level)
					assert.Equal(t, "HTTP request finished", lines[0].Message)
				case code <= 499:
					assert.Equal(t, zapcore.WarnLevel, lines[0].Level)
					assert.Equal(t, "HTTP request failed due to a client error", lines[0].Message)
				default:
					assert.Equal(t, zapcore.ErrorLevel, lines[0].Level)
					assert.Equal(t, "HTTP request failed", lines[0].Message)
				}
			})
		}
	})

	t.Run("Log when a handler panicked", func(t *testing.T) {
		t.Parallel()

		t.Run("With error", func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)

			requestLogger := zaphttp.NewHandler(
				zaphttp.WithLogger(logger),
			)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			assert.Panics(t, func() {
				requestLogger(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
					panic(errors.New("broken"))
				})).ServeHTTP(rec, req)
			})

			lines := logs.All()
			assert.Len(t, lines, 1)
			assert.Equal(t, zapcore.ErrorLevel, lines[0].Level)
			assert.Equal(t, "HTTP request panicked", lines[0].Message)
		})

		t.Run("With nil", func(t *testing.T) {
			core, logs := observer.New(zapcore.InfoLevel)
			logger := zap.New(core)

			requestLogger := zaphttp.NewHandler(
				zaphttp.WithLogger(logger),
			)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			assert.Panics(t, func() {
				requestLogger(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
					panic(nil)
				})).ServeHTTP(rec, req)
			})

			lines := logs.All()
			assert.Len(t, lines, 1)
			assert.Equal(t, zapcore.ErrorLevel, lines[0].Level)
			assert.Equal(t, "HTTP request panicked", lines[0].Message)
		})
	})

	t.Run("Test different formatters", func(t *testing.T) {
		t.Parallel()

		formatters := []struct {
			name      string
			formatter zaphttp.Formatter
		}{
			{"ECS", zaphttp.ElasticCommonSchemaFormatter},
			{"GCloud", zaphttp.NewGoogleCloudFormatter("test-project")},
			{"Noop", zaphttp.NoopFormatter},
		}

		for _, f := range formatters {
			t.Run(f.name, func(t *testing.T) {
				t.Parallel()

				core, logs := observer.New(zapcore.InfoLevel)
				logger := zap.New(core)

				requestLogger := zaphttp.NewHandler(
					zaphttp.WithLogger(logger),
					zaphttp.WithRequestFormatter(f.formatter),
					zaphttp.WithTraceFormatter(f.formatter),
				)

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				rec := httptest.NewRecorder()

				requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})).ServeHTTP(rec, req)

				// Just verify that logging worked without errors
				lines := logs.All()

				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Len(t, lines, 1)
				assert.Equal(t, zapcore.InfoLevel, lines[0].Level)
				assert.Equal(t, "HTTP request finished", lines[0].Message)
			})
		}
	})

	t.Run("Should inject trace ID into logs when in active span", func(t *testing.T) {
		t.Parallel()

		core, logs := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		requestLogger := zaphttp.NewHandler(
			zaphttp.WithLogger(logger),
		)

		// Create a new tracer
		tracer := noop.NewTracerProvider().Tracer("test")
		parentSpan := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    trace.TraceID{12, 34, 56, 78, 90},
			SpanID:     trace.SpanID{43, 21},
			TraceFlags: trace.FlagsSampled,
			Remote:     true,
		})
		ctx, span := tracer.Start(trace.ContextWithSpanContext(context.Background(), parentSpan), "test-span", trace.WithNewRoot())
		defer span.End()

		// Create request with trace context
		req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
		rec := httptest.NewRecorder()

		requestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l := zaphttp.FromContext(r.Context())
			l.Info("This should also contain the trace ID")
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)

		lines := logs.All()

		assert.Len(t, lines, 2)
		assert.Equal(t, zapcore.InfoLevel, lines[0].Level)
		assert.Equal(t, "This should also contain the trace ID", lines[0].Message)
		assert.Equal(t, zapcore.InfoLevel, lines[1].Level)
		assert.Equal(t, "HTTP request finished", lines[1].Message)

		// Verify trace ID is in the context map
		traceID := span.SpanContext().TraceID().String()

		traceMap, ok := lines[0].ContextMap()["trace"].(map[string]interface{})
		assert.True(t, ok, "trace field should be a map")
		assert.Equal(t, traceID, traceMap["id"])
		assert.Equal(t, true, traceMap["sampled"])

		traceMap1, ok := lines[1].ContextMap()["trace"].(map[string]interface{})
		assert.True(t, ok, "trace field should be a map")
		assert.Equal(t, traceID, traceMap1["id"])
		assert.Equal(t, true, traceMap1["sampled"])
	})

	t.Run("Check custom per request logger function", func(t *testing.T) {
		t.Parallel()

		core, logs := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		customLoggerFn := func(parent *zap.Logger, _ *http.Request) *zap.Logger {
			return parent.With(zap.String("custom", "field"))
		}

		requestLogger := zaphttp.NewHandler(
			zaphttp.WithLogger(logger),
			zaphttp.WithPerRequestLogger(customLoggerFn),
			zaphttp.WithTraceFormatter(zaphttp.NoopFormatter),
			zaphttp.WithRequestFormatter(zaphttp.NoopFormatter),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)

		lines := logs.All()
		assert.Len(t, lines, 1)
		assert.Equal(t, "field", lines[0].ContextMap()["custom"])
	})

	t.Run("Test status code and content type are passed correctly", func(t *testing.T) {
		t.Parallel()

		core, logs := observer.New(zapcore.InfoLevel)
		logger := zap.New(core)

		requestLogger := zaphttp.NewHandler(
			zaphttp.WithLogger(logger),
		)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
		})).ServeHTTP(rec, req)

		lines := logs.All()
		assert.Len(t, lines, 1)

		httpMap, ok := lines[0].ContextMap()["http"].(map[string]interface{})
		assert.True(t, ok, "http field should be a map")

		responseMap, ok := httpMap["response"].(map[string]interface{})
		assert.True(t, ok, "response field should be a map")

		assert.Equal(t, 201, responseMap["status_code"])
		assert.Equal(t, "application/json", responseMap["mime_type"])
	})

	t.Run("Check custom per request filter function", func(t *testing.T) {
		t.Parallel()

		t.Run("Should not log request when filter returns false", func(t *testing.T) {
			t.Parallel()

			core, logs := observer.New(zapcore.DebugLevel)
			logger := zap.New(core)

			customFilterFunc := func(_ *http.Request, _ zapcore.Level) bool {
				return false
			}

			requestLogger := zaphttp.NewHandler(
				zaphttp.WithLogger(logger),
				zaphttp.WithPerRequestFilter(customFilterFunc),
				zaphttp.WithTraceFormatter(zaphttp.NoopFormatter),
				zaphttp.WithRequestFormatter(zaphttp.NoopFormatter),
			)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()

			requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			})).ServeHTTP(rec, req)

			lines := logs.All()
			assert.Empty(t, lines)
		})

		t.Run("Should not log for requests matching filter", func(t *testing.T) {
			t.Parallel()

			customFilterFunc := func(req *http.Request, level zapcore.Level) bool {
				// Take if we should log or not based on the supplied request, this comes from the tests below.
				shouldLog := req.URL.Query().Get("shouldLogLevel") == level.String()
				return shouldLog
			}

			t.Run("Filter debug message", func(t *testing.T) {
				core, logs := observer.New(zapcore.DebugLevel)
				logger := zap.New(core)

				requestLogger := zaphttp.NewHandler(
					zaphttp.WithLogger(logger),
					zaphttp.WithPerRequestFilter(customFilterFunc),
					zaphttp.WithTraceFormatter(zaphttp.NoopFormatter),
					zaphttp.WithRequestFormatter(zaphttp.NoopFormatter),
				)

				req := httptest.NewRequest(http.MethodGet, "/?shouldLogLevel=info", nil)
				rec := httptest.NewRecorder()

				requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})).ServeHTTP(rec, req)

				lines := logs.All()
				assert.Len(t, lines, 1)
				assert.Equal(t, zapcore.InfoLevel, lines[0].Level)
				assert.Equal(t, "HTTP request finished", lines[0].Message)
			})

			t.Run("Filter info message", func(t *testing.T) {
				core, logs := observer.New(zapcore.DebugLevel)
				logger := zap.New(core)

				requestLogger := zaphttp.NewHandler(
					zaphttp.WithLogger(logger),
					zaphttp.WithPerRequestFilter(customFilterFunc),
					zaphttp.WithTraceFormatter(zaphttp.NoopFormatter),
					zaphttp.WithRequestFormatter(zaphttp.NoopFormatter),
				)

				req := httptest.NewRequest(http.MethodGet, "/?shouldLogLevel=debug", nil)
				rec := httptest.NewRecorder()

				requestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
				})).ServeHTTP(rec, req)

				lines := logs.All()
				assert.Len(t, lines, 1)
				assert.Equal(t, zapcore.DebugLevel, lines[0].Level)
				assert.Equal(t, "Received HTTP request", lines[0].Message)
			})
		})
	})
}
