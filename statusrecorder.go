package zaphttp

import (
	"net/http"
)

type statusRecorder struct {
	writer            http.ResponseWriter
	writeHeaderCalled bool

	StatusCode  int
	ContentType string
}

var _ http.ResponseWriter = &statusRecorder{}

func (s *statusRecorder) Header() http.Header {
	return s.writer.Header()
}

func (s *statusRecorder) Write(data []byte) (int, error) {
	if !s.writeHeaderCalled {
		// Replicate behaviour from http.ResponseWriter.
		// When Write() is called before WriteHeader(), a 200 OK is returned.
		s.WriteHeader(http.StatusOK)
	}
	return s.writer.Write(data)
}

func (s *statusRecorder) WriteHeader(statusCode int) {
	s.writeHeaderCalled = true
	s.StatusCode = statusCode
	s.ContentType = s.writer.Header().Get("Content-Type")
	s.writer.WriteHeader(statusCode)
}

// Unwrap implements the http.unWrapper interface (not exported). This is used for the http.ResponseController.
func (s *statusRecorder) Unwrap() http.ResponseWriter {
	return s.writer
}
