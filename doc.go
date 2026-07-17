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
//
// # Cancellation
//
// Requests honor a context (Config.Context / RequestConfig.Context) and, for an
// axios-style API, an AbortController: create one with NewAbortController, pass
// its Signal via RequestConfig.Signal, and call Abort to cancel. The legacy
// CancelToken (NewCancelToken) is also supported. IsCancel reports whether an
// error came from a cancellation.
//
// # Progress
//
// Config/RequestConfig OnUploadProgress and OnDownloadProgress receive
// ProgressEvent values as the request and response bodies are transferred.
//
// # Transforms
//
// TransformRequest and TransformResponse are pipelines that rewrite the raw
// body bytes (and may mutate headers) around encoding/decoding, complementing
// the request/response interceptors.
//
// # Response types and streaming
//
// ResponseType selects how the body is delivered. The default buffers into
// Response.Body; ResponseStream leaves it unread on Response.Stream for
// incremental consumption (call Response.Close when done).
//
// # Query parameters
//
// Params (url.Values) and ParamsMap (nested map, flattened with FlattenParams)
// are merged into the URL. ArrayFormat controls how repeated values are encoded
// (repeat/brackets/indices/comma), or supply a ParamsSerializer to take over
// entirely. GetUri returns the resolved URL without sending a request.
//
// # Decompression
//
// Responses are transparently decompressed (gzip and deflate) and an
// Accept-Encoding header is advertised, unless Config.Decompress is disabled.
//
// # Redirects and size guards
//
// MaxRedirects caps or disables redirect following (or supply a RedirectPolicy).
// MaxContentLength and MaxBodyLength bound response and request body sizes.
//
// # XSRF/CSRF
//
// With XSRFCookieName and XSRFHeaderName set, a token is copied from the
// client's cookie jar into the outgoing request header.
//
// # Instances and helpers
//
// Create returns a new client whose config is deep-merged over an existing one
// (request > instance > defaults precedence throughout). All and Spread run and
// combine several requests; IsAxiosError, AsError and Error.ToJSON aid error
// handling; HeaderDefaults provides common/get/post default header groups.
package axios
