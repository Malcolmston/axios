package axios

import (
	"encoding/json"
	"net/http"
)

// Response is the result of a completed HTTP request. The body has already been
// fully read into Body, so there is no stream to close and the value is safe to
// reuse and inspect repeatedly.
type Response struct {
	// Status is the numeric HTTP status code, e.g. 200.
	Status int
	// StatusText is the textual status, e.g. "200 OK".
	StatusText string
	// Headers holds the response headers.
	Headers http.Header
	// Body is the raw, fully-buffered response body.
	Body []byte
	// Request is the final *http.Request that produced this response.
	Request *http.Request
	// Raw is the underlying *http.Response with its Body already drained and
	// closed. Provided for advanced use (TLS info, cookies, ...).
	Raw *http.Response
}

// JSON decodes the response body into v using encoding/json.
func (r *Response) JSON(v any) error {
	return json.Unmarshal(r.Body, v)
}

// Text returns the response body as a string.
func (r *Response) Text() string {
	return string(r.Body)
}

// Bytes returns the raw response body bytes.
func (r *Response) Bytes() []byte {
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
