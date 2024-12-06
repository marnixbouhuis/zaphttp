package zaphttp

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// gcloudHTTPRequest is the logging format used for HTTP requests for google cloud.
// See: https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#HttpRequest
type gcloudHTTPRequest struct {
	RequestMethod string
	RequestURL    string
	RequestSize   string
	Status        int
	UserAgent     string
	RemoteIP      string
	ServerIP      string
	Referrer      string
	Latency       string
	Protocol      string
}

func (h *gcloudHTTPRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("requestMethod", h.RequestMethod)
	enc.AddString("requestUrl", h.RequestURL)
	enc.AddString("requestSize", h.RequestSize)
	enc.AddInt("status", h.Status)
	enc.AddString("userAgent", h.UserAgent)
	enc.AddString("remoteIp", h.RemoteIP)
	enc.AddString("serverIp", h.ServerIP)
	enc.AddString("referrer", h.Referrer)
	enc.AddString("latency", h.Latency)
	enc.AddString("protocol", h.Protocol)
	return nil
}

type gcloudFormatter struct {
	projectID string
}

// Do not provide a default instance since we need the GCP project ID for fields like the full trace ID.
var _ Formatter = &gcloudFormatter{}

// NewGoogleCloudFormatter returns a log field formatter that will log HTTP requests and traces in a Google cloud
// compatible format.
func NewGoogleCloudFormatter(projectID string) Formatter {
	return &gcloudFormatter{projectID: projectID}
}

func (f *gcloudFormatter) GetTraceFields(_ *http.Request, spanCtx trace.SpanContext) []zap.Field {
	traceID := fmt.Sprintf("projects/%s/traces/%s", f.projectID, spanCtx.TraceID().String())
	return []zap.Field{
		zap.String("trace", traceID),
		zap.String("spanId", spanCtx.SpanID().String()),
		zap.Bool("traceSampled", spanCtx.IsSampled()),
	}
}

func (f *gcloudFormatter) GetRequestFields(req *http.Request, res *ResponseInfo) []zap.Field {
	var serverIP string
	if localAddr, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
		serverIP = localAddr.String()
	}

	h := &gcloudHTTPRequest{
		RequestMethod: req.Method,
		RequestURL:    req.URL.Redacted(),
		RequestSize:   strconv.FormatInt(req.ContentLength, 10),
		Status:        res.StatusCode,
		UserAgent:     req.UserAgent(),
		RemoteIP:      req.RemoteAddr,
		ServerIP:      serverIP,
		Referrer:      req.Referer(),
		Latency:       strconv.FormatFloat(res.Latency.Seconds(), 'f', -1, 64) + "s",
		Protocol:      req.Proto,
	}

	return []zap.Field{
		zap.Object("httpRequest", h),
	}
}
