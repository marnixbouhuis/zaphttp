# zaphttp - Structured Logging for HTTP Requests in Go
[![Go Reference](https://pkg.go.dev/badge/github.com/marnixbouhuis/zaphttp.svg)](https://pkg.go.dev/github.com/marnixbouhuis/zaphttp)
[![CI/CD Pipeline](https://github.com/marnixbouhuis/zaphttp/actions/workflows/cicd.yaml/badge.svg)](https://github.com/marnixbouhuis/zaphttp/actions/workflows/cicd.yaml)

zaphttp is a Go library that provides structured logging for HTTP requests using [zap](https://github.com/uber-go/zap). It creates a per-request logger that automatically has things like a trace ID injected into it.

## Features

- Per-request logger injection
- OpenTelemetry integration
- Flexible log formatting
- Built-in support for formats like Elastic Common Schema (ECS) and Google Cloud logging
- Customizable per-request logger behavior

## Installation

```bash
go get github.com/marnixbouhuis/zaphttp
```

## Quick Start

```go
logger := zap.NewExample()
zap.ReplaceGlobals(logger)
defer func() {
	_ = logger.Sync()
}()

mux := http.NewServeMux()
mux.HandleFunc("/demo/{$}", func(w http.ResponseWriter, req *http.Request) {
	// Optional, get the logger for this request from the context.
	// If you are using opentelemetry, the trace ID is automatically injected into each log message.
	l := zaphttp.FromContext(req.Context())

	// Optional, log something with the request logger.
	l.Info("Some message!")

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("Hello world!"))
})

requestLogger := zaphttp.NewHandler(
	zaphttp.WithLogger(logger),                                         // If no logger is supplied, zap.L(), will be used.
	zaphttp.WithTraceFormatter(zaphttp.ElasticCommonSchemaFormatter),   // If no format for trace metadata is supplied, ECS is used.
	zaphttp.WithRequestFormatter(zaphttp.ElasticCommonSchemaFormatter), // If no format for request metadata is supplied, ECS is used.
)

s := &http.Server{
	Addr:         ":8080",
	ReadTimeout:  5 * time.Second,
	WriteTimeout: 5 * time.Second,
	Handler:      requestLogger(mux), // Wrap the mux, all requests will now be logged.
}

// Do graceful shutdown of HTTP server here...

if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
	logger.Error("Failed to start server", zap.Error(err))
}
```

Every request received by the HTTP server is now logged.

## Core Concepts

### Handler
The main component is the `NewHandler()` function that creates a middleware to wrap your HTTP handlers.

Options:
- `WithLogger(logger *zap.Logger)` - Set a custom logger (default: `zap.L()`)
- `WithTraceFormatter(formatter TraceFormatter)` - Set a custom trace formatter (default: ECS)
- `WithRequestFormatter(formatter RequestFormatter)` - Set a custom request formatter (default: ECS)
- `WithPerRequestLogger(fn PerRequestLoggerFunc)` - Customize how the per-request logger is created

### Formatters
Formatters determine how request and trace information is structured in the logs.

Built-in formatters:
- `ElasticCommonSchemaFormatter` - Formats logs according to the Elastic Common Schema
- `NewGoogleCloudFormatter(projectID)` - Formats logs for Google Cloud Logging
- `NoopFormatter` - Disables all extra fields

### Per-Request Logger
The per-request logger is injected into the request context and can be retrieved using `FromContext()`. It automatically includes:

- Trace ID (if using OpenTelemetry)
- Custom fields added by the per-request logger function

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details.
