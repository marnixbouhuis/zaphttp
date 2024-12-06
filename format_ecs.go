package zaphttp

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ecsTrace represents trace info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-tracing.html
type ecsTrace struct {
	// ID is the trace ID, see: https://www.elastic.co/guide/en/ecs/current/ecs-tracing.html#field-trace-id
	ID string
	// Sampled indicates if a trace has been sampled or not. It is not a standard field but still a nice to have in logs.
	Sampled bool
}

func (t *ecsTrace) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("id", t.ID)
	enc.AddBool("sampled", t.Sampled)
	return nil
}

// ecsSpan represents trace span info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-tracing.html
type ecsSpan struct {
	// ID is the span ID, see: https://www.elastic.co/guide/en/ecs/current/ecs-tracing.html#field-span-id
	ID string
}

func (s *ecsSpan) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("id", s.ID)
	return nil
}

// ecsEvent represents event info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-event.html
type ecsEvent struct {
	// Start is the start of the event, see: https://www.elastic.co/guide/en/ecs/current/ecs-event.html#field-event-start
	Start time.Time
	// Duration is the duration of the event, see: https://www.elastic.co/guide/en/ecs/current/ecs-event.html#field-event-duration
	Duration time.Duration
	// End is the start of the event, see: https://www.elastic.co/guide/en/ecs/current/ecs-event.html#field-event-end
	End time.Time
}

func (e *ecsEvent) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("start", e.Start.Format(time.RFC3339Nano))
	enc.AddInt64("duration", e.Duration.Nanoseconds())
	enc.AddString("end", e.End.Format(time.RFC3339Nano))
	return nil
}

// ecsHTTPRequestBody represents HTTP request body info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-http.html
type ecsHTTPRequestBody struct {
	// Bytes is the size of the request body, see: https://www.elastic.co/guide/en/ecs/current/ecs-http.html#field-http-request-body-bytes
	Bytes int64
}

func (b *ecsHTTPRequestBody) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt64("bytes", b.Bytes)
	return nil
}

// ecsHTTPRequest represents HTTP request info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-http.html
type ecsHTTPRequest struct {
	// Body contains information about the request body, see: https://www.elastic.co/guide/en/ecs/current/ecs-http.html
	Body *ecsHTTPRequestBody
	// Method is the HTTP method used for this request, see: https://www.elastic.co/guide/en/ecs/current/ecs-http.html#field-http-request-method
	Method string
	// MimeType is the content type sent by the client, see: https://www.elastic.co/guide/en/ecs/current/ecs-http.html#field-http-request-mime-type
	MimeType string
	// Referrer is the referrer sent by the client, see: https://www.elastic.co/guide/en/ecs/current/ecs-http.html#field-http-request-referrer
	Referrer string
}

func (r *ecsHTTPRequest) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if err := enc.AddObject("body", r.Body); err != nil {
		return err
	}
	enc.AddString("method", r.Method)
	enc.AddString("mime_type", r.MimeType)
	enc.AddString("referrer", r.Referrer)
	return nil
}

// ecsHTTPResponse represents HTTP response info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-http.html
type ecsHTTPResponse struct {
	// MimeType is the content type sent by the server, see: https://www.elastic.co/guide/en/ecs/current/ecs-http.html#field-http-response-mime-type
	MimeType string
	// StatusCode is the response code sent by the server, see: https://www.elastic.co/guide/en/ecs/current/ecs-http.html#field-http-response-status-code
	StatusCode int
}

func (r *ecsHTTPResponse) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("mime_type", r.MimeType)
	enc.AddInt("status_code", r.StatusCode)
	return nil
}

// ecsHTTPResponse represents HTTP info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-http.html
type ecsHTTP struct {
	// Request contains information about the request sent by the client
	Request *ecsHTTPRequest
	// Response contains information about the response sent by the server
	Response *ecsHTTPResponse
	// Version is the HTTP version used for this request, see: https://www.elastic.co/guide/en/ecs/current/ecs-http.html#field-http-version
	Version string
}

func (h *ecsHTTP) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	if err := enc.AddObject("request", h.Request); err != nil {
		return err
	}
	if err := enc.AddObject("response", h.Response); err != nil {
		return err
	}
	enc.AddString("version", h.Version)
	return nil
}

// ecsURL represents URL info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-url.html
type ecsURL struct {
	// URL will be mapped to all different supported fields for an ECS URL.
	URL *url.URL
}

func (u *ecsURL) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	var username string
	if u.URL.User != nil {
		username = u.URL.User.Username()
	}

	enc.AddString("original", u.URL.Redacted())
	enc.AddString("path", u.URL.Path)
	enc.AddString("query", u.URL.RawQuery)
	enc.AddString("scheme", u.URL.Scheme)
	enc.AddString("username", username)
	return nil
}

// ecsUserAgent represents user agent info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-user_agent.html
type ecsUserAgent struct {
	// Original is the unparsed user agent of the client, see: https://www.elastic.co/guide/en/ecs/current/ecs-user_agent.html#field-user-agent-original
	Original string
}

func (u *ecsUserAgent) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("original", u.Original)
	return nil
}

// ecsClient represents client info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-client.html
type ecsClient struct {
	// Address is the address the client connected from, see: https://www.elastic.co/guide/en/ecs/current/ecs-client.html#field-client-address
	Address string
}

func (c *ecsClient) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("address", c.Address)
	return nil
}

// ecsServer represents server info formatted for elastic common schema logging.
// See: https://www.elastic.co/guide/en/ecs/current/ecs-server.html
type ecsServer struct {
	// Address is the address the server is listening on, see: https://www.elastic.co/guide/en/ecs/current/ecs-server.html#field-server-address
	Address string
}

func (s *ecsServer) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("address", s.Address)
	return nil
}

type elasticCommonSchemaFormatter struct{}

var ElasticCommonSchemaFormatter Formatter = &elasticCommonSchemaFormatter{}

func (*elasticCommonSchemaFormatter) GetTraceFields(_ *http.Request, spanCtx trace.SpanContext) []zap.Field {
	return []zap.Field{
		zap.Object("trace", &ecsTrace{
			ID:      spanCtx.TraceID().String(),
			Sampled: spanCtx.IsSampled(),
		}),
		zap.Object("span", &ecsSpan{
			ID: spanCtx.SpanID().String(),
		}),
	}
}

func (*elasticCommonSchemaFormatter) GetRequestFields(req *http.Request, res *ResponseInfo) []zap.Field {
	var serverAddr string
	if localAddr, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr); ok {
		serverAddr = localAddr.String()
	}

	return []zap.Field{
		zap.Object("event", &ecsEvent{
			Start:    res.Start,
			Duration: res.Latency,
			End:      res.Start.Add(res.Latency),
		}),
		zap.Object("http", &ecsHTTP{
			Request: &ecsHTTPRequest{
				Body: &ecsHTTPRequestBody{
					Bytes: req.ContentLength,
				},
				Method:   req.Method,
				MimeType: req.Header.Get("Content-Type"),
				Referrer: req.Referer(),
			},
			Response: &ecsHTTPResponse{
				MimeType:   res.ContentType,
				StatusCode: res.StatusCode,
			},
			Version: fmt.Sprintf("%d.%d", req.ProtoMajor, req.ProtoMinor),
		}),
		zap.Object("url", &ecsURL{
			URL: req.URL,
		}),
		zap.Object("user_agent", &ecsUserAgent{
			Original: req.UserAgent(),
		}),
		zap.Object("client", &ecsClient{
			Address: req.RemoteAddr,
		}),
		zap.Object("server", &ecsServer{
			Address: serverAddr,
		}),
	}
}
