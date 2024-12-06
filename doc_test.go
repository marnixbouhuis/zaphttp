package zaphttp_test

import (
	"errors"
	"net/http"
	"time"

	"github.com/marnixbouhuis/zaphttp"
	"go.uber.org/zap"
)

func Example() {
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
}
