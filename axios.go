package axios

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// BasicAuth holds credentials for HTTP Basic authentication.
type BasicAuth struct {
	// Username is the Basic auth user name.
	Username string
	// Password is the Basic auth password.
	Password string
}

// Config configures a Client instance. Every field is optional; the zero value
// yields a usable client backed by http.DefaultClient semantics.
type Config struct {
	// BaseURL is prepended to request URLs that are not absolute.
	BaseURL string
	// Headers are sent with every request. Per-request headers take precedence
	// on a per-key basis.
	Headers http.Header
	// HeaderGroups holds method-scoped default headers (common/get/post/...),
	// mirroring axios defaults.headers. They are applied beneath Headers.
	HeaderGroups HeaderDefaults
	// Params are default query parameters merged into every request.
	Params url.Values
	// ParamsMap holds nested/typed default query parameters that are flattened
	// with bracket notation (see FlattenParams) and merged into every request.
	ParamsMap map[string]any
	// ParamsSerializer, when set, fully controls how query parameters are
	// encoded (axios paramsSerializer).
	ParamsSerializer ParamsSerializer
	// ArrayFormat selects how repeated params are serialized when
	// ParamsSerializer is nil. The zero value repeats the key.
	ArrayFormat ArrayFormat
	// Timeout, when > 0, bounds each individual attempt via a context deadline.
	Timeout time.Duration
	// BasicAuth, when set, adds an Authorization: Basic header.
	BasicAuth *BasicAuth
	// BearerToken, when non-empty, adds an Authorization: Bearer header.
	BearerToken string
	// Proxy configures an outbound HTTP proxy. It is honored only when
	// HTTPClient is nil (the client builds its own transport).
	Proxy *ProxyConfig
	// HTTPClient lets callers supply a fully custom *http.Client. If nil a
	// client is created (optionally using Transport/Proxy/redirect policy).
	HTTPClient *http.Client
	// Transport is used to build the internal *http.Client when HTTPClient is
	// nil.
	Transport http.RoundTripper
	// Context is the default context for requests. If nil, context.Background
	// is used. A per-request context overrides this.
	Context context.Context
	// Signal cancels every request made with this client when its controller
	// aborts (axios signal). A per-request Signal overrides it.
	Signal *AbortSignal
	// ValidateStatus decides which status codes resolve successfully. If nil,
	// 2xx codes are treated as success and everything else yields an *Error.
	ValidateStatus func(status int) bool
	// MaxRedirects controls redirect following, mirroring axios' maxRedirects.
	// nil leaves the transport's default policy in place (redirects are
	// followed). A non-nil pointer to 0 means "do not follow": the 3xx response
	// is returned as-is. A positive value caps the chain and fails with
	// ERR_FR_TOO_MANY_REDIRECTS when exceeded. A negative value fails on the
	// first redirect. Honored only when HTTPClient is nil.
	MaxRedirects *int
	// RedirectPolicy fully overrides redirect handling (http.Client
	// CheckRedirect) when set. Honored only when HTTPClient is nil.
	RedirectPolicy func(req *http.Request, via []*http.Request) error
	// MaxContentLength, when > 0, rejects response bodies larger than this many
	// bytes (axios maxContentLength).
	MaxContentLength int64
	// MaxBodyLength, when > 0, rejects request bodies larger than this many
	// bytes (axios maxBodyLength).
	MaxBodyLength int64
	// Decompress controls transparent gzip/deflate decompression of responses.
	// Nil (the default) enables it; set to a pointer to false to disable.
	Decompress *bool
	// ResponseType selects how response bodies are delivered (buffered vs.
	// streamed). See ResponseType.
	ResponseType ResponseType
	// XSRFCookieName and XSRFHeaderName enable copying an anti-CSRF token from a
	// response cookie (in the client's cookie jar) into a request header. Both
	// must be set for the feature to activate.
	XSRFCookieName string
	// XSRFHeaderName is the request header that receives the XSRF cookie value.
	XSRFHeaderName string
	// OnUploadProgress is called as the request body is written.
	OnUploadProgress ProgressFunc
	// OnDownloadProgress is called as the response body is read.
	OnDownloadProgress ProgressFunc
	// TransformRequest transforms the encoded request body before sending.
	TransformRequest []RequestTransform
	// TransformResponse transforms the response body before interceptors and
	// status validation.
	TransformResponse []ResponseTransform
	// Retry configures automatic retries. If nil, retrying is disabled.
	Retry *RetryConfig
	// RequestInterceptors run, in order, before each request is sent.
	RequestInterceptors []RequestInterceptor
	// ResponseInterceptors run, in order, after each response is received.
	ResponseInterceptors []ResponseInterceptor
}

// RequestConfig overrides Client defaults for a single request. All fields are
// optional. Header and Param maps are merged with the client defaults (the
// request wins on conflicting keys); other non-zero fields replace the client
// value for that request.
type RequestConfig struct {
	// Headers are merged over the client's default headers for this request.
	Headers http.Header
	// Params are query parameters merged over the client's defaults (request
	// wins per key).
	Params url.Values
	// ParamsMap holds nested/typed query parameters flattened with bracket
	// notation (see FlattenParams) for this request.
	ParamsMap map[string]any
	// Timeout, when > 0, overrides the client's per-attempt timeout.
	Timeout time.Duration
	// BasicAuth, when set, overrides the client's auth with Basic credentials.
	BasicAuth *BasicAuth
	// BearerToken, when non-empty, overrides the client's auth with a bearer
	// token.
	BearerToken string
	// Context, when set, overrides the client's default context for this
	// request.
	Context context.Context
	// ValidateStatus, when set, overrides the client's status predicate for
	// this request.
	ValidateStatus func(status int) bool
	// ContentType, when set, overrides the automatically chosen Content-Type.
	ContentType string
	// Signal cancels this request when its controller aborts.
	Signal *AbortSignal
	// CancelToken cancels this request (legacy axios cancellation).
	CancelToken *CancelToken
	// ParamsSerializer overrides the client's query serialization for this
	// request.
	ParamsSerializer ParamsSerializer
	// ArrayFormat overrides the client's repeated-param format for this
	// request.
	ArrayFormat *ArrayFormat
	// ResponseType overrides the client's response delivery for this request.
	ResponseType *ResponseType
	// MaxContentLength overrides the client's response size guard.
	MaxContentLength int64
	// MaxBodyLength overrides the client's request size guard.
	MaxBodyLength int64
	// OnUploadProgress overrides the client's upload progress callback.
	OnUploadProgress ProgressFunc
	// OnDownloadProgress overrides the client's download progress callback.
	OnDownloadProgress ProgressFunc
	// TransformRequest overrides the client's request transform pipeline.
	TransformRequest []RequestTransform
	// TransformResponse overrides the client's response transform pipeline.
	TransformResponse []ResponseTransform
}

// Client is a reusable, configured HTTP client. It is safe for concurrent use.
type Client struct {
	cfg  Config
	http *http.Client
}

// New creates a Client from the given configuration.
func New(cfg Config) *Client {
	c := &Client{cfg: cfg}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{}
		transport := cfg.Transport
		needsHTTPTransport := (cfg.Proxy != nil && cfg.Proxy.URL() != nil) ||
			(cfg.Decompress != nil && !*cfg.Decompress)
		if needsHTTPTransport {
			base, _ := transport.(*http.Transport)
			if base == nil {
				base = http.DefaultTransport.(*http.Transport).Clone()
			}
			if cfg.Proxy != nil && cfg.Proxy.URL() != nil {
				base.Proxy = http.ProxyURL(cfg.Proxy.URL())
			}
			if cfg.Decompress != nil && !*cfg.Decompress {
				// Prevent the transport from transparently negotiating and
				// decoding gzip so Decompress:false is fully honored.
				base.DisableCompression = true
			}
			transport = base
		}
		if transport != nil {
			hc.Transport = transport
		}
		if cfg.MaxRedirects != nil || cfg.RedirectPolicy != nil {
			hc.CheckRedirect = c.checkRedirect
		}
	}
	c.http = hc
	return c
}

// errTooManyRedirects is returned by checkRedirect when the redirect chain
// exceeds the configured maximum. It maps to axios' ERR_FR_TOO_MANY_REDIRECTS.
var errTooManyRedirects = errors.New("axios: maximum number of redirects exceeded")

// checkRedirect implements the client's redirect policy (see Config.MaxRedirects
// and Config.RedirectPolicy). It is installed only when one of those is set.
func (c *Client) checkRedirect(req *http.Request, via []*http.Request) error {
	if c.cfg.RedirectPolicy != nil {
		return c.cfg.RedirectPolicy(req, via)
	}
	if c.cfg.MaxRedirects == nil {
		return nil
	}
	max := *c.cfg.MaxRedirects
	if max == 0 {
		// axios maxRedirects:0 means "do not follow"; return the 3xx response.
		return http.ErrUseLastResponse
	}
	if len(via) > max {
		return errTooManyRedirects
	}
	return nil
}

// default package-level client, configurable via SetDefault.
var std = New(Config{})

// Default returns the package-level default client used by the package-level
// Get/Post/... helpers.
func Default() *Client { return std }

// SetDefault replaces the package-level default client.
func SetDefault(c *Client) { std = c }

// Create returns a new Client whose configuration is cfg deep-merged over this
// client's configuration (request-time values still take ultimate precedence).
// It mirrors axios.create. Headers, header groups and params are merged
// key-by-key; other set fields in cfg override the base.
func (c *Client) Create(cfg Config) *Client {
	return New(mergeConfig(c.cfg, cfg))
}

// Create returns a new Client whose configuration is cfg deep-merged over the
// package-level default client's configuration. It mirrors axios.create.
func Create(cfg Config) *Client {
	return std.Create(cfg)
}

// Get issues a GET request.
func (c *Client) Get(rawURL string, cfg ...*RequestConfig) (*Response, error) {
	return c.Request(http.MethodGet, rawURL, nil, opt(cfg))
}

// Delete issues a DELETE request.
func (c *Client) Delete(rawURL string, cfg ...*RequestConfig) (*Response, error) {
	return c.Request(http.MethodDelete, rawURL, nil, opt(cfg))
}

// Head issues a HEAD request.
func (c *Client) Head(rawURL string, cfg ...*RequestConfig) (*Response, error) {
	return c.Request(http.MethodHead, rawURL, nil, opt(cfg))
}

// Options issues an OPTIONS request.
func (c *Client) Options(rawURL string, cfg ...*RequestConfig) (*Response, error) {
	return c.Request(http.MethodOptions, rawURL, nil, opt(cfg))
}

// Post issues a POST request with the given body.
func (c *Client) Post(rawURL string, body any, cfg ...*RequestConfig) (*Response, error) {
	return c.Request(http.MethodPost, rawURL, body, opt(cfg))
}

// Put issues a PUT request with the given body.
func (c *Client) Put(rawURL string, body any, cfg ...*RequestConfig) (*Response, error) {
	return c.Request(http.MethodPut, rawURL, body, opt(cfg))
}

// Patch issues a PATCH request with the given body.
func (c *Client) Patch(rawURL string, body any, cfg ...*RequestConfig) (*Response, error) {
	return c.Request(http.MethodPatch, rawURL, body, opt(cfg))
}

func opt(cfg []*RequestConfig) *RequestConfig {
	if len(cfg) > 0 {
		return cfg[0]
	}
	return nil
}

// GetUri returns the fully-resolved URL (base + path + serialized params) that
// a request with the given config would target, without sending it. It mirrors
// axios getUri.
func (c *Client) GetUri(rawURL string, rc *RequestConfig) (string, error) {
	if rc == nil {
		rc = &RequestConfig{}
	}
	return c.buildURL(rawURL, rc)
}

// Request is the low-level entry point that all helper methods build on. Body
// is encoded automatically based on its dynamic type (see EncodeBody).
func (c *Client) Request(method, rawURL string, body any, rc *RequestConfig) (*Response, error) {
	if rc == nil {
		rc = &RequestConfig{}
	}

	fullURL, err := c.buildURL(rawURL, rc)
	if err != nil {
		// A malformed URL is a client-side (config) error; axios surfaces the
		// platform's TypeError here, which is *not* an AxiosError. Mirror that
		// by returning a distinct, non-*Error value carrying ERR_INVALID_URL.
		return nil, &invalidURLError{err: err}
	}
	// Reject protocols the transport cannot speak before dialing, matching
	// axios' "Unsupported protocol" ERR_BAD_REQUEST.
	if u, perr := url.Parse(fullURL); perr == nil {
		if sc := u.Scheme; sc != "" && sc != "http" && sc != "https" {
			return nil, &Error{Message: "unsupported protocol " + sc + ":", Code: ErrCodeBadRequest}
		}
	}

	bodyBytes, _, err := EncodeBody(body)
	if err != nil {
		return nil, &Error{Message: "encode body", Err: err, Code: ErrCodeBadRequest}
	}

	// Choose the request Content-Type the way axios does: JSON for structured
	// values, url-encoded (with charset) for URLSearchParams, and the request
	// method's default (x-www-form-urlencoded for POST/PUT/PATCH) for raw
	// string/binary/empty bodies. A user transformRequest replaces the default
	// serializer, so the body-derived type is dropped in favor of the method
	// default. An explicit RequestConfig.ContentType always wins.
	hasTransforms := len(c.requestTransforms(rc)) > 0
	contentType := requestContentType(method, body, hasTransforms)
	if rc.ContentType != "" {
		contentType = rc.ContentType
	}

	// TransformRequest pipeline: may rewrite the body and set headers.
	transformHeaders := http.Header{}
	if contentType != "" {
		transformHeaders.Set("Content-Type", contentType)
	}
	if ts := c.requestTransforms(rc); len(ts) > 0 {
		bodyBytes, err = applyRequestTransforms(ts, bodyBytes, transformHeaders)
		if err != nil {
			// A throwing transform propagates verbatim in axios (a plain Error,
			// isAxiosError === false); do the same rather than wrapping it.
			return nil, err
		}
	}

	// MaxBodyLength guard.
	if limit := c.maxBodyLength(rc); limit > 0 && int64(len(bodyBytes)) > limit {
		return nil, &Error{
			Message: fmt.Sprintf("request body of %d bytes exceeds MaxBodyLength %d", len(bodyBytes), limit),
			Code:    ErrCodeBadRequest,
		}
	}

	ctx := c.resolveContext(rc)
	uploadFn := c.uploadProgress(rc)

	// build a fresh request per attempt so bodies can be replayed on retry.
	newReq := func() (*http.Request, context.CancelFunc, error) {
		reqCtx := ctx
		var cancel context.CancelFunc
		timeout := c.cfg.Timeout
		if rc.Timeout > 0 {
			timeout = rc.Timeout
		}
		if timeout > 0 {
			reqCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		var rdr io.Reader
		if bodyBytes != nil {
			rdr = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(reqCtx, method, fullURL, rdr)
		if err != nil {
			if cancel != nil {
				cancel()
			}
			return nil, nil, err
		}
		if uploadFn != nil && len(bodyBytes) > 0 {
			req.Body = io.NopCloser(newProgressReader(bytes.NewReader(bodyBytes), int64(len(bodyBytes)), uploadFn))
		}
		c.applyHeaders(req, rc, method, transformHeaders)
		c.applyAuth(req, rc)
		c.applyXSRF(req)
		c.applyAcceptEncoding(req)
		c.applyDefaultHeaders(req)
		for _, ri := range c.cfg.RequestInterceptors {
			if err := ri(req); err != nil {
				if cancel != nil {
					cancel()
				}
				return nil, nil, &userError{err: err}
			}
		}
		return req, cancel, nil
	}

	resp, req, err := c.execute(newReq, rc)
	if err != nil {
		// A user callback (request interceptor) that failed propagates verbatim,
		// matching axios which does not wrap interceptor errors.
		var ue *userError
		if errors.As(err, &ue) {
			return nil, ue.err
		}
		code := ErrCodeNetwork
		switch {
		case errors.Is(err, context.DeadlineExceeded):
			code = ErrCodeTimeout // axios: ECONNABORTED
		case IsCancel(err):
			code = ErrCodeCanceled
		case errors.Is(err, errTooManyRedirects):
			code = ErrCodeTooManyRedirects
		case errors.Is(err, errMaxContentLength):
			code = ErrCodeBadResponse
		case errors.Is(err, syscall.ECONNREFUSED):
			code = ErrCodeConnRefused
		}
		return nil, &Error{Message: "request failed", Request: req, Err: err, Code: code}
	}

	// TransformResponse pipeline (buffered responses only).
	if resp.Stream == nil {
		if ts := c.responseTransforms(rc); len(ts) > 0 {
			out, terr := applyResponseTransforms(ts, resp.Body, resp.Headers)
			if terr != nil {
				return resp, &Error{Message: "transform response", Request: req, Response: resp, Err: terr, Code: ErrCodeBadResponse}
			}
			resp.Body = out
		}
	}

	for _, ri := range c.cfg.ResponseInterceptors {
		if err := ri(resp); err != nil {
			// A throwing response interceptor propagates verbatim in axios
			// (a plain Error, isAxiosError === false). The response is still
			// handed back for inspection; because the error is not an *Error,
			// IsAxiosError/ErrorCode report it as a non-axios failure.
			return resp, err
		}
	}

	if !c.validate(rc, resp.Status) {
		// axios settles a rejected status with ERR_BAD_REQUEST for 4xx and
		// ERR_BAD_RESPONSE for 5xx; any other rejected status (e.g. a 2xx or 3xx
		// turned down by a custom validateStatus) carries no code.
		code := ""
		switch resp.Status / 100 {
		case 4:
			code = ErrCodeBadRequest
		case 5:
			code = ErrCodeBadResponse
		}
		return resp, &Error{
			Message:  "request failed with status " + resp.StatusText,
			Request:  req,
			Response: resp,
			Code:     code,
		}
	}
	return resp, nil
}

// execute runs the request with retry handling and buffers the response body.
func (c *Client) execute(newReq func() (*http.Request, context.CancelFunc, error), rc *RequestConfig) (*Response, *http.Request, error) {
	retries := 0
	var rcfg *RetryConfig
	if c.cfg.Retry != nil {
		rcfg = c.cfg.Retry
		retries = rcfg.Retries
	}

	var lastResp *Response
	var lastReq *http.Request
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		req, cancel, err := newReq()
		if err != nil {
			return nil, req, err
		}
		lastReq = req

		resp, err := c.do(req, rc)
		if cancel != nil {
			// A streamed body still needs the context alive while it is read,
			// so hand the cancel to the response; buffered responses cancel now.
			if resp == nil || resp.Stream == nil {
				cancel()
			} else {
				resp.cancel = cancel
			}
		}
		lastResp, lastErr = resp, err

		if rcfg == nil || attempt == retries {
			break
		}
		if !rcfg.retryOn(resp, err) {
			break
		}
		// Discard an unread streamed body before retrying to avoid leaks.
		if resp != nil && resp.Stream != nil {
			_ = resp.Close()
		}
		time.Sleep(rcfg.backoff(attempt + 1))
	}

	if lastErr != nil {
		return nil, lastReq, lastErr
	}
	return lastResp, lastReq, nil
}

// do performs a single HTTP round-trip. For streaming responses it leaves the
// body open on Response.Stream; otherwise it buffers (decompressing and
// enforcing MaxContentLength) into Response.Body.
func (c *Client) do(req *http.Request, rc *RequestConfig) (*Response, error) {
	raw, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	// axios' res.statusText is the HTTP reason phrase only ("OK"), whereas
	// net/http's raw.Status includes the code ("200 OK"); strip the code.
	statusText := raw.Status
	if _, rest, ok := strings.Cut(raw.Status, " "); ok {
		statusText = rest
	}
	resp := &Response{
		Status:     raw.StatusCode,
		StatusText: statusText,
		Headers:    raw.Header,
		Request:    req,
		Raw:        raw,
	}

	var bodyReader io.Reader = raw.Body
	var extraCloser io.Closer
	if enc := raw.Header.Get("Content-Encoding"); enc != "" && c.decompressEnabled() {
		dr, closer, derr := decompress(enc, bodyReader)
		if derr != nil {
			_ = raw.Body.Close()
			return nil, derr
		}
		if closer != nil {
			bodyReader = dr
			extraCloser = closer
			// Body is now decoded: drop the encoding/length metadata.
			raw.Header.Del("Content-Encoding")
			raw.Header.Del("Content-Length")
		}
	}

	total := raw.ContentLength
	bodyReader = newProgressReader(bodyReader, total, c.downloadProgress(rc))

	if c.responseType(rc) == ResponseStream {
		resp.Stream = &multiReadCloser{r: bodyReader, closers: []io.Closer{extraCloser, raw.Body}}
		return resp, nil
	}

	defer func() { _ = raw.Body.Close() }()
	if extraCloser != nil {
		defer func() { _ = extraCloser.Close() }()
	}

	data, err := readAllLimited(bodyReader, c.maxContentLength(rc))
	if err != nil {
		return nil, err
	}
	resp.Body = data
	return resp, nil
}

// errMaxContentLength is returned when a response body exceeds MaxContentLength.
// It maps to axios' ERR_BAD_RESPONSE ("maxContentLength size ... exceeded").
var errMaxContentLength = errors.New("axios: response body exceeds MaxContentLength")

// readAllLimited reads all of r, returning an error if limit > 0 and the body
// exceeds it.
func readAllLimited(r io.Reader, limit int64) ([]byte, error) {
	if limit <= 0 {
		return io.ReadAll(r)
	}
	data, err := io.ReadAll(io.LimitReader(r, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%w %d", errMaxContentLength, limit)
	}
	return data, nil
}

func (c *Client) resolveContext(rc *RequestConfig) context.Context {
	switch {
	case rc.Signal != nil:
		return rc.Signal.Context()
	case rc.CancelToken != nil:
		return rc.CancelToken.Context()
	case rc.Context != nil:
		return rc.Context
	case c.cfg.Signal != nil:
		return c.cfg.Signal.Context()
	case c.cfg.Context != nil:
		return c.cfg.Context
	default:
		return context.Background()
	}
}

func (c *Client) buildURL(rawURL string, rc *RequestConfig) (string, error) {
	// Combine the configured base with the requested URL using axios' semantics
	// (helpers/buildFullPath + combineURLs): a base path is preserved and joined
	// with the request path, while an absolute request URL overrides the base.
	full := rawURL
	if c.cfg.BaseURL != "" {
		full = BuildFullPath(c.cfg.BaseURL, rawURL, true)
	}
	// Validate that the combined URL parses so malformed URLs surface as errors.
	if _, err := url.Parse(full); err != nil {
		return "", err
	}

	// Gather all params into a single url.Values, request winning per key.
	q := url.Values{}
	for k, vs := range c.cfg.Params {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	for k, vs := range FlattenParams(c.cfg.ParamsMap) {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	for k, vs := range rc.Params {
		q[k] = append([]string(nil), vs...)
	}
	for k, vs := range FlattenParams(rc.ParamsMap) {
		q[k] = append([]string(nil), vs...)
	}

	if len(q) == 0 {
		return full, nil
	}

	var serialized string
	if ser := c.paramsSerializer(rc); ser != nil {
		serialized = ser(q)
	} else {
		serialized = serializeParamsAxios(q, c.arrayFormat(rc))
	}
	if serialized == "" {
		return full, nil
	}
	// Append after any existing query, discarding a fragment, exactly like
	// axios' helpers/buildURL: "?" when the URL has no query, "&" otherwise.
	if i := strings.IndexByte(full, '#'); i != -1 {
		full = full[:i]
	}
	sep := "?"
	if strings.IndexByte(full, '?') != -1 {
		sep = "&"
	}
	return full + sep + serialized, nil
}

func (c *Client) applyHeaders(req *http.Request, rc *RequestConfig, method string, transformHeaders http.Header) {
	// Header groups (common + method) form the base layer.
	for _, group := range c.cfg.HeaderGroups.forMethod(method) {
		for k, vs := range group {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
	// Client default headers.
	for k, vs := range c.cfg.Headers {
		req.Header[http.CanonicalHeaderKey(k)] = append([]string(nil), vs...)
	}
	// Per-request headers win.
	for k, vs := range rc.Headers {
		req.Header[http.CanonicalHeaderKey(k)] = append([]string(nil), vs...)
	}
	// Content-Type / other headers set by the transform pipeline, unless the
	// caller already set the key explicitly.
	for k, vs := range transformHeaders {
		if req.Header.Get(k) == "" {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
}

func (c *Client) applyAuth(req *http.Request, rc *RequestConfig) {
	// per-request auth wins over client defaults.
	switch {
	case rc.BearerToken != "":
		req.Header.Set("Authorization", "Bearer "+rc.BearerToken)
	case rc.BasicAuth != nil:
		req.SetBasicAuth(rc.BasicAuth.Username, rc.BasicAuth.Password)
	case c.cfg.BearerToken != "":
		req.Header.Set("Authorization", "Bearer "+c.cfg.BearerToken)
	case c.cfg.BasicAuth != nil:
		req.SetBasicAuth(c.cfg.BasicAuth.Username, c.cfg.BasicAuth.Password)
	}
}

// applyXSRF copies an anti-CSRF token from the client's cookie jar into the
// configured request header, mirroring axios xsrfCookieName/xsrfHeaderName.
func (c *Client) applyXSRF(req *http.Request) {
	if c.cfg.XSRFCookieName == "" || c.cfg.XSRFHeaderName == "" {
		return
	}
	if req.Header.Get(c.cfg.XSRFHeaderName) != "" {
		return
	}
	if c.http.Jar == nil {
		return
	}
	for _, ck := range c.http.Jar.Cookies(req.URL) {
		if ck.Name == c.cfg.XSRFCookieName {
			req.Header.Set(c.cfg.XSRFHeaderName, ck.Value)
			return
		}
	}
}

// applyAcceptEncoding advertises the codec list axios always sends, regardless
// of whether decompression is enabled, unless the caller set Accept-Encoding
// explicitly. Setting it here also stops net/http from negotiating its own
// transparent gzip, so responses are decompressed by do() instead.
func (c *Client) applyAcceptEncoding(req *http.Request) {
	if req.Header.Get("Accept-Encoding") == "" {
		req.Header.Set("Accept-Encoding", "gzip, compress, deflate, br")
	}
}

// applyDefaultHeaders sets the default Accept and User-Agent axios ships, unless
// the caller supplied their own. Accept mirrors defaults.headers.common.Accept;
// User-Agent identifies the port in axios' "axios/<version>" form.
func (c *Client) applyDefaultHeaders(req *http.Request) {
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "axios/"+Version)
	}
}

func (c *Client) validate(rc *RequestConfig, status int) bool {
	if rc.ValidateStatus != nil {
		return rc.ValidateStatus(status)
	}
	if c.cfg.ValidateStatus != nil {
		return c.cfg.ValidateStatus(status)
	}
	return status >= 200 && status < 300
}

// ---- per-request/config resolvers (request > instance > default) ----

func (c *Client) requestTransforms(rc *RequestConfig) []RequestTransform {
	if rc.TransformRequest != nil {
		return rc.TransformRequest
	}
	return c.cfg.TransformRequest
}

func (c *Client) responseTransforms(rc *RequestConfig) []ResponseTransform {
	if rc.TransformResponse != nil {
		return rc.TransformResponse
	}
	return c.cfg.TransformResponse
}

func (c *Client) paramsSerializer(rc *RequestConfig) ParamsSerializer {
	if rc.ParamsSerializer != nil {
		return rc.ParamsSerializer
	}
	return c.cfg.ParamsSerializer
}

func (c *Client) arrayFormat(rc *RequestConfig) ArrayFormat {
	if rc.ArrayFormat != nil {
		return *rc.ArrayFormat
	}
	if c.cfg.ArrayFormat != 0 {
		return c.cfg.ArrayFormat
	}
	// axios' default array serialization is bracket notation (a[]=1&a[]=2). The
	// zero value of ArrayFormat is ArrayFormatRepeat, so callers who explicitly
	// want repeated keys pass ArrayFormatRepeat via a non-nil pointer (which the
	// checks above honor); an unset format defaults to brackets to match axios.
	return ArrayFormatBrackets
}

func (c *Client) responseType(rc *RequestConfig) ResponseType {
	if rc.ResponseType != nil {
		return *rc.ResponseType
	}
	return c.cfg.ResponseType
}

func (c *Client) uploadProgress(rc *RequestConfig) ProgressFunc {
	if rc.OnUploadProgress != nil {
		return rc.OnUploadProgress
	}
	return c.cfg.OnUploadProgress
}

func (c *Client) downloadProgress(rc *RequestConfig) ProgressFunc {
	if rc.OnDownloadProgress != nil {
		return rc.OnDownloadProgress
	}
	return c.cfg.OnDownloadProgress
}

func (c *Client) maxContentLength(rc *RequestConfig) int64 {
	if rc.MaxContentLength > 0 {
		return rc.MaxContentLength
	}
	return c.cfg.MaxContentLength
}

func (c *Client) maxBodyLength(rc *RequestConfig) int64 {
	if rc.MaxBodyLength > 0 {
		return rc.MaxBodyLength
	}
	return c.cfg.MaxBodyLength
}

func (c *Client) decompressEnabled() bool {
	return c.cfg.Decompress == nil || *c.cfg.Decompress
}

// EncodeBody converts a request body value into bytes and a default
// Content-Type. The rules are:
//
//   - nil                        -> no body, no content type
//   - []byte                     -> raw bytes, application/octet-stream
//   - string                     -> raw bytes, text/plain; charset=utf-8
//   - io.Reader                  -> drained to bytes, application/octet-stream
//   - url.Values                 -> form encoded, application/x-www-form-urlencoded
//   - anything else              -> JSON encoded, application/json
func EncodeBody(body any) ([]byte, string, error) {
	switch b := body.(type) {
	case nil:
		return nil, "", nil
	case []byte:
		return b, "application/octet-stream", nil
	case string:
		return []byte(b), "text/plain; charset=utf-8", nil
	case url.Values:
		return []byte(b.Encode()), "application/x-www-form-urlencoded", nil
	case io.Reader:
		data, err := io.ReadAll(b)
		if err != nil {
			return nil, "", err
		}
		return data, "application/octet-stream", nil
	default:
		// Encode without HTML escaping so <, > and & survive verbatim, matching
		// JavaScript's JSON.stringify (Go's json.Marshal would emit < etc.).
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(body); err != nil {
			return nil, "", err
		}
		return bytes.TrimRight(buf.Bytes(), "\n"), "application/json", nil
	}
}

// methodDefaultContentType returns the Content-Type axios inherits from
// defaults.headers[method] for a body that carries no type of its own. POST,
// PUT and PATCH default to application/x-www-form-urlencoded; other methods
// carry no default.
func methodDefaultContentType(method string) string {
	switch strings.ToUpper(method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return "application/x-www-form-urlencoded"
	}
	return ""
}

// requestContentType chooses the outgoing Content-Type the way axios' default
// transformRequest does. Structured values become application/json;
// URLSearchParams become application/x-www-form-urlencoded;charset=utf-8; raw
// string/binary/empty bodies fall back to the request method's default header.
// When a user transformRequest is present it replaces the default serializer,
// so the body-derived type is discarded and only the method default applies.
func requestContentType(method string, body any, hasTransforms bool) string {
	if hasTransforms {
		return methodDefaultContentType(method)
	}
	switch body.(type) {
	case nil, string, []byte, io.Reader:
		return methodDefaultContentType(method)
	case url.Values:
		return "application/x-www-form-urlencoded;charset=utf-8"
	default:
		return "application/json"
	}
}

func hasScheme(s string) bool {
	i := strings.Index(s, "://")
	return i > 0
}

func joinPath(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	case strings.HasSuffix(a, "/") && strings.HasPrefix(b, "/"):
		return a + b[1:]
	case !strings.HasSuffix(a, "/") && !strings.HasPrefix(b, "/"):
		return a + "/" + b
	default:
		return a + b
	}
}
