package axios

import (
	"fmt"
	"net/http"
)

// Error is the typed error returned by the client when a request fails, either
// because of a transport/network problem or because the response status was
// rejected by the configured ValidateStatus predicate.
//
// When the failure is a rejected status, Response is populated so callers can
// still inspect the body, headers and status code. When the failure is a
// transport error (DNS, connection refused, timeout, ...), Err holds the
// underlying error and Response is nil.
//
// Error implements the error interface and supports errors.Is/errors.As via
// Unwrap.
type Error struct {
	// Message is a human readable description of what went wrong.
	Message string
	// Request is the outgoing *http.Request that produced the error, when
	// available.
	Request *http.Request
	// Response is the parsed response. It is non-nil for status rejections and
	// nil for transport errors.
	Response *Response
	// Err is the underlying error, if any (transport errors, encode errors...).
	Err error
}

// Error implements the error interface.
func (e *Error) Error() string {
	if e.Response != nil {
		return fmt.Sprintf("axios: %s (status %d %s)", e.Message, e.Response.Status, e.Response.StatusText)
	}
	if e.Err != nil {
		return fmt.Sprintf("axios: %s: %v", e.Message, e.Err)
	}
	return "axios: " + e.Message
}

// Unwrap returns the underlying error so that errors.Is and errors.As work.
func (e *Error) Unwrap() error { return e.Err }

// StatusCode is a convenience accessor that returns the HTTP status code
// associated with the error, or 0 when there is no response.
func (e *Error) StatusCode() int {
	if e.Response == nil {
		return 0
	}
	return e.Response.Status
}
