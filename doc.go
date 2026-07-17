// Package axios is an ergonomic, axios-style HTTP client built entirely on the
// Go standard library (net/http). It offers a configurable client instance,
// convenience verb methods, automatic body encoding/decoding, request and
// response interceptors, configurable retries with backoff, and a typed error
// that carries the response for non-2xx statuses.
//
// # Clients and configuration
//
// Create a client with New and a Config. Every Config field is optional:
//
//	client := axios.New(axios.Config{
//		BaseURL:     "https://api.example.com",
//		Headers:     http.Header{"X-App": {"demo"}},
//		Params:      url.Values{"v": {"1"}},
//		Timeout:     5 * time.Second,
//		BearerToken: "secret",
//	})
//
// A zero-value client is also ready to use, and the package exposes
// package-level Get/Post/... helpers backed by a default client (see Default and
// SetDefault).
//
// # Verb methods
//
// Get, Delete, Head and Options take a URL and an optional *RequestConfig. Post,
// Put and Patch additionally take a body:
//
//	resp, err := client.Get("/users", &axios.RequestConfig{
//		Params: url.Values{"page": {"2"}},
//	})
//	resp, err = client.Post("/users", map[string]any{"name": "Ada"})
//
// The lower-level Request method underlies all of them.
//
// # Bodies
//
// Request bodies are encoded automatically by their dynamic type (see
// EncodeBody): structs and maps become JSON (application/json), url.Values
// becomes form-urlencoded, and []byte/string/io.Reader are sent raw. Set
// RequestConfig.ContentType to override the chosen Content-Type.
//
// # Responses
//
// A Response buffers the whole body into Body and exposes helpers: JSON decodes
// into a value, Text returns a string, OK reports a 2xx status, and Status,
// StatusText and Headers expose the metadata. The generic helpers GetJSON and
// GetJSONDefault fetch and decode in one call.
//
// # Status validation and errors
//
// By default any non-2xx response resolves to an *Error whose Response field
// still holds the parsed body, letting callers inspect it. Override this per
// client or per request with ValidateStatus (analogous to axios validateStatus).
// Transport failures also return an *Error, with the underlying error available
// via errors.Unwrap.
//
// # Interceptors
//
// RequestInterceptors run in order after the outgoing request is built but
// before it is sent, and may mutate it. ResponseInterceptors run in order after
// the response is buffered but before status validation, and may transform it.
//
// # Retries
//
// Set Config.Retry to enable automatic retries. RetryConfig controls the number
// of attempts, the Backoff schedule, and a RetryOn predicate. The defaults
// (DefaultBackoff, DefaultRetryOn) retry on transport errors and 5xx responses
// with exponential backoff.
package axios
