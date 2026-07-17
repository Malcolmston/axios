package axios

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

// Response is the result of a completed HTTP request. For buffered response
// types the body has already been fully read into Body, so there is no stream
// to close and the value is safe to reuse and inspect repeatedly. For the
// ResponseStream type the body is instead available on Stream and the caller
// MUST call Close to release the underlying connection.
type Response struct {
	// Status is the numeric HTTP status code, e.g. 200.
	Status int
	// StatusText is the textual status, e.g. "200 OK".
	StatusText string
	// Headers holds the response headers.
	Headers http.Header
	// Body is the raw, fully-buffered response body. It is nil for streaming
	// responses (see Stream).
	Body []byte
	// Stream is the unbuffered response body for ResponseStream requests. It is
	// nil for buffered responses. Read from it and then Close the response.
	Stream io.ReadCloser
	// Request is the final *http.Request that produced this response.
	Request *http.Request
	// Raw is the underlying *http.Response. For buffered responses its Body has
	// already been drained and closed. Provided for advanced use (TLS info,
	// cookies, ...).
	Raw *http.Response

	// cancel releases the request context; set for streaming responses so Close
	// can tear it down.
	cancel context.CancelFunc
}

// JSON decodes the response body into v using encoding/json. For a streaming
// response it reads and decodes the stream.
func (r *Response) JSON(v any) error {
	if r.Body == nil && r.Stream != nil {
		return json.NewDecoder(r.Stream).Decode(v)
	}
	return json.Unmarshal(r.Body, v)
}

// Text returns the response body as a string. For a streaming response it reads
// the stream to completion.
func (r *Response) Text() string {
	return string(r.Bytes())
}

// Bytes returns the raw response body bytes. For a streaming response it reads
// the stream to completion (and caches the result in Body).
func (r *Response) Bytes() []byte {
	if r.Body == nil && r.Stream != nil {
		data, _ := io.ReadAll(r.Stream)
		r.Body = data
	}
	return r.Body
}

// OK reports whether the status code is in the 2xx range.
func (r *Response) OK() bool {
	return r.Status >= 200 && r.Status < 300
}

// Header returns the first value associated with the given response header key.
func (r *Response) Header(key string) string {
	return r.Headers.Get(key)
}

// Close releases the resources held by a streaming response (the network body
// and request context). It is a no-op for buffered responses and is safe to
// call multiple times.
func (r *Response) Close() error {
	var err error
	if r.Stream != nil {
		err = r.Stream.Close()
		r.Stream = nil
	}
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
	return err
}

// multiReadCloser presents an io.Reader together with a set of closers (the
// decompressor and the raw network body) as a single io.ReadCloser.
type multiReadCloser struct {
	r       io.Reader
	closers []io.Closer
}

// Read implements io.Reader.
func (m *multiReadCloser) Read(b []byte) (int, error) { return m.r.Read(b) }

// Close closes every underlying closer, returning the first error.
func (m *multiReadCloser) Close() error {
	var first error
	for _, c := range m.closers {
		if c == nil {
			continue
		}
		if err := c.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}
