package axios

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Error codes classify an *Error, mirroring the axios error.code values so
// callers can branch on the kind of failure.
const (
	// ErrCodeInvalidURL indicates the request URL could not be built or parsed.
	ErrCodeInvalidURL = "ERR_INVALID_URL"
	// ErrCodeBadRequest indicates the request could not be prepared (body
	// encoding, request transform, size guard).
	ErrCodeBadRequest = "ERR_BAD_REQUEST"
	// ErrCodeNetwork indicates a transport/connection failure.
	ErrCodeNetwork = "ERR_NETWORK"
	// ErrCodeCanceled indicates the request was canceled or its context was
	// done (timeout, abort).
	ErrCodeCanceled = "ERR_CANCELED"
	// ErrCodeBadResponse indicates the response was received but rejected
	// (status validation, response transform/interceptor, decode).
	ErrCodeBadResponse = "ERR_BAD_RESPONSE"
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
	// Code classifies the error (one of the ErrCode* constants).
	Code string
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

// ToJSON returns a JSON-serializable view of the error, mirroring axios
// error.toJSON. It includes the message, code, HTTP method and URL (when a
// request is available) and the status (when a response is available).
func (e *Error) ToJSON() map[string]any {
	m := map[string]any{
		"message": e.Message,
		"name":    "AxiosError",
	}
	if e.Code != "" {
		m["code"] = e.Code
	}
	if e.Err != nil {
		m["cause"] = e.Err.Error()
	}
	if e.Request != nil {
		m["method"] = e.Request.Method
		if e.Request.URL != nil {
			m["url"] = e.Request.URL.String()
		}
	}
	if e.Response != nil {
		m["status"] = e.Response.Status
	}
	return m
}

// MarshalJSON implements json.Marshaler using ToJSON, so encoding an *Error
// yields the same shape as axios error.toJSON.
func (e *Error) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.ToJSON())
}

// IsAxiosError reports whether err is (or wraps) an *Error produced by this
// package, mirroring axios.isAxiosError.
func IsAxiosError(err error) bool {
	return AsError(err) != nil
}

// AsError returns err as an *Error if it is one (or wraps one), else nil. It is
// a convenience wrapper over errors.As.
func AsError(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return nil
}
