package axios

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BasicAuth holds credentials for HTTP Basic authentication.
type BasicAuth struct {
	Username string
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
	// Params are default query parameters merged into every request.
	Params url.Values
	// Timeout, when > 0, bounds each individual attempt via a context deadline.
	Timeout time.Duration
	// BasicAuth, when set, adds an Authorization: Basic header.
	BasicAuth *BasicAuth
	// BearerToken, when non-empty, adds an Authorization: Bearer header.
	BearerToken string
	// HTTPClient lets callers supply a fully custom *http.Client. If nil a
	// client is created (optionally using Transport).
	HTTPClient *http.Client
	// Transport is used to build the internal *http.Client when HTTPClient is
	// nil.
	Transport http.RoundTripper
	// Context is the default context for requests. If nil, context.Background
	// is used. A per-request context overrides this.
	Context context.Context
	// ValidateStatus decides which status codes resolve successfully. If nil,
	// 2xx codes are treated as success and everything else yields an *Error.
	ValidateStatus func(status int) bool
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
	Headers        http.Header
	Params         url.Values
	Timeout        time.Duration
	BasicAuth      *BasicAuth
	BearerToken    string
	Context        context.Context
	ValidateStatus func(status int) bool
	// ContentType, when set, overrides the automatically chosen Content-Type.
	ContentType string
}

// Client is a reusable, configured HTTP client. It is safe for concurrent use.
type Client struct {
	cfg  Config
	http *http.Client
}

// New creates a Client from the given configuration.
func New(cfg Config) *Client {
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{}
		if cfg.Transport != nil {
			hc.Transport = cfg.Transport
		}
	}
	return &Client{cfg: cfg, http: hc}
}

// default package-level client, configurable via SetDefault.
var std = New(Config{})

// Default returns the package-level default client used by the package-level
// Get/Post/... helpers.
func Default() *Client { return std }

// SetDefault replaces the package-level default client.
func SetDefault(c *Client) { std = c }

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

// Request is the low-level entry point that all helper methods build on. Body
// is encoded automatically based on its dynamic type (see EncodeBody).
func (c *Client) Request(method, rawURL string, body any, rc *RequestConfig) (*Response, error) {
	if rc == nil {
		rc = &RequestConfig{}
	}

	fullURL, err := c.buildURL(rawURL, rc)
	if err != nil {
		return nil, &Error{Message: "invalid url", Err: err}
	}

	bodyBytes, contentType, err := EncodeBody(body)
	if err != nil {
		return nil, &Error{Message: "encode body", Err: err}
	}
	if rc.ContentType != "" {
		contentType = rc.ContentType
	}

	ctx := c.resolveContext(rc)

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
		c.applyHeaders(req, rc, contentType)
		c.applyAuth(req, rc)
		for _, ri := range c.cfg.RequestInterceptors {
			if err := ri(req); err != nil {
				if cancel != nil {
					cancel()
				}
				return nil, nil, err
			}
		}
		return req, cancel, nil
	}

	resp, req, err := c.execute(newReq)
	if err != nil {
		return nil, &Error{Message: "request failed", Request: req, Err: err}
	}

	for _, ri := range c.cfg.ResponseInterceptors {
		if err := ri(resp); err != nil {
			return resp, &Error{Message: "response interceptor", Request: req, Response: resp, Err: err}
		}
	}

	if !c.validate(rc, resp.Status) {
		return resp, &Error{
			Message:  "request failed with status " + resp.StatusText,
			Request:  req,
			Response: resp,
		}
	}
	return resp, nil
}

// execute runs the request with retry handling and buffers the response body.
func (c *Client) execute(newReq func() (*http.Request, context.CancelFunc, error)) (*Response, *http.Request, error) {
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

		resp, err := c.do(req)
		if cancel != nil {
			cancel()
		}
		lastResp, lastErr = resp, err

		if rcfg == nil || attempt == retries {
			break
		}
		if !rcfg.retryOn(resp, err) {
			break
		}
		time.Sleep(rcfg.backoff(attempt + 1))
	}

	if lastErr != nil {
		return nil, lastReq, lastErr
	}
	return lastResp, lastReq, nil
}

// do performs a single HTTP round-trip and fully buffers the body.
func (c *Client) do(req *http.Request) (*Response, error) {
	raw, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = raw.Body.Close() }()
	data, err := io.ReadAll(raw.Body)
	if err != nil {
		return nil, err
	}
	return &Response{
		Status:     raw.StatusCode,
		StatusText: raw.Status,
		Headers:    raw.Header,
		Body:       data,
		Request:    req,
		Raw:        raw,
	}, nil
}

func (c *Client) resolveContext(rc *RequestConfig) context.Context {
	if rc.Context != nil {
		return rc.Context
	}
	if c.cfg.Context != nil {
		return c.cfg.Context
	}
	return context.Background()
}

func (c *Client) buildURL(rawURL string, rc *RequestConfig) (string, error) {
	base := c.cfg.BaseURL
	var u *url.URL
	var err error

	if base != "" && !hasScheme(rawURL) {
		bu, err := url.Parse(base)
		if err != nil {
			return "", err
		}
		ref, err := url.Parse(rawURL)
		if err != nil {
			return "", err
		}
		u = bu.ResolveReference(ref)
		// ResolveReference drops a base path when ref is rooted; for a
		// non-rooted ref it joins. Preserve the more intuitive "join" behavior
		// by re-joining paths when ref is relative.
		if rawURL != "" && !strings.HasPrefix(rawURL, "/") {
			u.Path = joinPath(bu.Path, ref.Path)
		}
	} else {
		u, err = url.Parse(rawURL)
		if err != nil {
			return "", err
		}
	}

	q := u.Query()
	for k, vs := range c.cfg.Params {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	for k, vs := range rc.Params {
		q[k] = append([]string(nil), vs...)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (c *Client) applyHeaders(req *http.Request, rc *RequestConfig, contentType string) {
	for k, vs := range c.cfg.Headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	for k, vs := range rc.Headers {
		req.Header[http.CanonicalHeaderKey(k)] = append([]string(nil), vs...)
	}
	if contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", contentType)
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

func (c *Client) validate(rc *RequestConfig, status int) bool {
	if rc.ValidateStatus != nil {
		return rc.ValidateStatus(status)
	}
	if c.cfg.ValidateStatus != nil {
		return c.cfg.ValidateStatus(status)
	}
	return status >= 200 && status < 300
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
		data, err := json.Marshal(body)
		if err != nil {
			return nil, "", err
		}
		return data, "application/json", nil
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
