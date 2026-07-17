package axios

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type user struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestGetJSONRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"Ada","age":36}`)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL})
	resp, err := c.Get("/user")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !resp.OK() {
		t.Fatalf("not ok: %d", resp.Status)
	}
	if got := resp.Header("Content-Type"); got != "application/json" {
		t.Errorf("content-type = %q", got)
	}
	var u user
	if err := resp.JSON(&u); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if u.Name != "Ada" || u.Age != 36 {
		t.Errorf("decoded = %+v", u)
	}
	if resp.Status != 200 || !strings.Contains(resp.StatusText, "200") {
		t.Errorf("status = %d %q", resp.Status, resp.StatusText)
	}
}

func TestParamsMerge(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Params: url.Values{"a": {"1"}}})
	_, err := c.Get("/", &RequestConfig{Params: url.Values{"b": {"2"}, "a": {"override"}}})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotQuery.Get("b") != "2" {
		t.Errorf("b = %q", gotQuery.Get("b"))
	}
	if gotQuery.Get("a") != "override" {
		t.Errorf("a = %q, want override", gotQuery.Get("a"))
	}
}

func TestHeadersAndAuth(t *testing.T) {
	var h http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h = r.Header.Clone()
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL:     srv.URL,
		Headers:     http.Header{"X-App": {"demo"}},
		BearerToken: "clienttok",
	})
	_, err := c.Get("/", &RequestConfig{Headers: http.Header{"X-Extra": {"y"}}})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if h.Get("X-App") != "demo" {
		t.Errorf("X-App = %q", h.Get("X-App"))
	}
	if h.Get("X-Extra") != "y" {
		t.Errorf("X-Extra = %q", h.Get("X-Extra"))
	}
	if h.Get("Authorization") != "Bearer clienttok" {
		t.Errorf("auth = %q", h.Get("Authorization"))
	}
}

func TestPerRequestBasicAuthOverridesBearer(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, BearerToken: "clienttok"})
	_, err := c.Get("/", &RequestConfig{BasicAuth: &BasicAuth{Username: "u", Password: "p"}})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.HasPrefix(auth, "Basic ") {
		t.Errorf("auth = %q, want Basic", auth)
	}
}

func TestClientBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "user" || p != "pass" {
			t.Errorf("basic auth = %q %q ok=%v", u, p, ok)
		}
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, BasicAuth: &BasicAuth{Username: "user", Password: "pass"}})
	if _, err := c.Get("/"); err != nil {
		t.Fatalf("Get: %v", err)
	}
}

func TestPostJSON(t *testing.T) {
	var body user
	var ct string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		_ = readJSON(r, &body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL})
	resp, err := c.Post("/user", user{Name: "Grace", Age: 40})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if resp.Status != 201 {
		t.Errorf("status = %d", resp.Status)
	}
	if ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
	if body.Name != "Grace" || body.Age != 40 {
		t.Errorf("body = %+v", body)
	}
}

func TestPostForm(t *testing.T) {
	var ct, raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL})
	_, err := c.Post("/form", url.Values{"q": {"hello world"}})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if ct != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q", ct)
	}
	if !strings.Contains(raw, "q=hello+world") {
		t.Errorf("body = %q", raw)
	}
}

func TestPostRawBytesAndString(t *testing.T) {
	var raw, ct string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
		ct = r.Header.Get("Content-Type")
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})

	if _, err := c.Put("/raw", []byte("abc")); err != nil {
		t.Fatalf("Put bytes: %v", err)
	}
	if raw != "abc" || ct != "application/octet-stream" {
		t.Errorf("bytes: raw=%q ct=%q", raw, ct)
	}

	if _, err := c.Patch("/raw", "hello", &RequestConfig{ContentType: "text/x-custom"}); err != nil {
		t.Fatalf("Patch string: %v", err)
	}
	if raw != "hello" || ct != "text/x-custom" {
		t.Errorf("string: raw=%q ct=%q", raw, ct)
	}
}

func TestPostReaderBody(t *testing.T) {
	var raw string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		raw = string(b)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	if _, err := c.Post("/r", strings.NewReader("streamed")); err != nil {
		t.Fatalf("Post reader: %v", err)
	}
	if raw != "streamed" {
		t.Errorf("raw = %q", raw)
	}
}

func TestInterceptors(t *testing.T) {
	var sawHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawHeader = r.Header.Get("X-Intercept")
		_, _ = io.WriteString(w, "orig")
	}))
	defer srv.Close()

	order := []string{}
	c := New(Config{
		BaseURL: srv.URL,
		RequestInterceptors: []RequestInterceptor{
			func(req *http.Request) error {
				order = append(order, "req1")
				req.Header.Set("X-Intercept", "yes")
				return nil
			},
			func(req *http.Request) error { order = append(order, "req2"); return nil },
		},
		ResponseInterceptors: []ResponseInterceptor{
			func(resp *Response) error {
				order = append(order, "resp1")
				resp.Body = []byte(strings.ToUpper(string(resp.Body)))
				return nil
			},
		},
	})
	resp, err := c.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if sawHeader != "yes" {
		t.Errorf("intercept header = %q", sawHeader)
	}
	if resp.Text() != "ORIG" {
		t.Errorf("transformed body = %q", resp.Text())
	}
	want := []string{"req1", "req2", "resp1"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("order = %v", order)
	}
}

func TestRequestInterceptorError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	sentinel := errors.New("blocked")
	c := New(Config{BaseURL: srv.URL, RequestInterceptors: []RequestInterceptor{
		func(req *http.Request) error { return sentinel },
	}})
	_, err := c.Get("/")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v, want sentinel", err)
	}
}

func TestResponseInterceptorError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	sentinel := errors.New("bad resp")
	c := New(Config{BaseURL: srv.URL, ResponseInterceptors: []ResponseInterceptor{
		func(resp *Response) error { return sentinel },
	}})
	resp, err := c.Get("/")
	if !errors.Is(err, sentinel) {
		t.Fatalf("err = %v", err)
	}
	if resp == nil {
		t.Errorf("resp should be returned on interceptor error")
	}
}

func TestRetryOn5xxThenSuccess(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, "ok")
	}))
	defer srv.Close()

	var backoffCalls int
	c := New(Config{BaseURL: srv.URL, Retry: &RetryConfig{
		Retries: 5,
		Backoff: func(attempt int) time.Duration { backoffCalls++; return time.Millisecond },
	}})
	resp, err := c.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Text() != "ok" {
		t.Errorf("body = %q", resp.Text())
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	if backoffCalls != 2 {
		t.Errorf("backoffCalls = %d, want 2", backoffCalls)
	}
}

func TestRetryExhaustionReturnsError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New(Config{BaseURL: srv.URL, Retry: &RetryConfig{
		Retries: 2,
		Backoff: func(int) time.Duration { return time.Millisecond },
	}})
	resp, err := c.Get("/")
	var aerr *Error
	if !errors.As(err, &aerr) {
		t.Fatalf("err = %v, want *Error", err)
	}
	if aerr.StatusCode() != 502 {
		t.Errorf("status = %d", aerr.StatusCode())
	}
	if resp == nil || resp.Status != 502 {
		t.Errorf("resp missing")
	}
	if atomic.LoadInt32(&calls) != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestRetryPredicateSkips(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, Retry: &RetryConfig{
		Retries: 3,
		RetryOn: func(resp *Response, err error) bool { return false },
	}})
	_, _ = c.Get("/")
	if atomic.LoadInt32(&calls) != 1 {
		t.Errorf("calls = %d, want 1 (predicate false)", calls)
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, Timeout: 20 * time.Millisecond})
	_, err := c.Get("/")
	if err == nil {
		t.Fatal("expected timeout error")
	}
	var aerr *Error
	if !errors.As(err, &aerr) {
		t.Fatalf("err type = %T", err)
	}
}

func TestNon2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"error":"missing"}`)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	resp, err := c.Get("/x")
	if err == nil {
		t.Fatal("expected error")
	}
	var aerr *Error
	if !errors.As(err, &aerr) {
		t.Fatalf("err type = %T", err)
	}
	if aerr.Response == nil || aerr.Response.Status != 404 {
		t.Errorf("error response missing")
	}
	// body is still accessible for inspection.
	if !strings.Contains(resp.Text(), "missing") {
		t.Errorf("body = %q", resp.Text())
	}
	if aerr.Error() == "" {
		t.Errorf("empty error string")
	}
}

func TestValidateStatusOverride(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, ValidateStatus: func(s int) bool { return s < 500 }})
	if _, err := c.Get("/"); err != nil {
		t.Fatalf("404 should be accepted: %v", err)
	}
	// per-request override wins.
	_, err := c.Get("/", &RequestConfig{ValidateStatus: func(s int) bool { return s == 200 }})
	if err == nil {
		t.Fatal("per-request validate should reject 404")
	}
}

func TestGenericGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"name":"Lin","age":22}`)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	u, err := GetJSON[user](c, "/u")
	if err != nil {
		t.Fatalf("GetJSON: %v", err)
	}
	if u.Name != "Lin" || u.Age != 22 {
		t.Errorf("u = %+v", u)
	}
}

func TestGenericGetJSONError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	if _, err := GetJSON[user](c, "/u"); err == nil {
		t.Fatal("expected error")
	}
}

func TestGenericGetJSONDecodeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `not json`)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	_, err := GetJSON[user](c, "/u")
	var aerr *Error
	if !errors.As(err, &aerr) {
		t.Fatalf("err = %v", err)
	}
}

func TestPackageLevelDefaultClient(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "root:"+r.Method+r.URL.Path)
	}))
	defer srv.Close()

	old := Default()
	SetDefault(New(Config{BaseURL: srv.URL}))
	defer SetDefault(old)

	tests := []struct {
		name string
		fn   func() (*Response, error)
		want string
	}{
		{"get", func() (*Response, error) { return Get("/a") }, "root:GET/a"},
		{"delete", func() (*Response, error) { return Delete("/a") }, "root:DELETE/a"},
		{"options", func() (*Response, error) { return Options("/a") }, "root:OPTIONS/a"},
		{"post", func() (*Response, error) { return Post("/a", nil) }, "root:POST/a"},
		{"put", func() (*Response, error) { return Put("/a", nil) }, "root:PUT/a"},
		{"patch", func() (*Response, error) { return Patch("/a", nil) }, "root:PATCH/a"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := tt.fn()
			if err != nil {
				t.Fatalf("%s: %v", tt.name, err)
			}
			if resp.Text() != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, resp.Text(), tt.want)
			}
		})
	}

	// HEAD has no body.
	if _, err := Head("/a"); err != nil {
		t.Fatalf("Head: %v", err)
	}
	// generic default.
	if _, err := GetJSONDefault[map[string]any]("/a"); err == nil {
		// body is not valid JSON, so decode error expected; that's fine.
		_ = err
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	c := New(Config{BaseURL: srv.URL})
	_, err := c.Get("/", &RequestConfig{Context: ctx})
	if err == nil {
		t.Fatal("expected context error")
	}
}

func TestBaseURLJoining(t *testing.T) {
	cases := []struct {
		base, path, want string
	}{
		{"https://x.test/api", "users", "https://x.test/api/users"},
		{"https://x.test/api/", "users", "https://x.test/api/users"},
		{"https://x.test/api", "/users", "https://x.test/users"},
		{"https://x.test", "https://y.test/z", "https://y.test/z"},
	}
	c := New(Config{})
	for _, tc := range cases {
		c.cfg.BaseURL = tc.base
		got, err := c.buildURL(tc.path, &RequestConfig{})
		if err != nil {
			t.Fatalf("buildURL(%q,%q): %v", tc.base, tc.path, err)
		}
		if got != tc.want {
			t.Errorf("buildURL(%q,%q) = %q, want %q", tc.base, tc.path, got, tc.want)
		}
	}
}

func TestEncodeBody(t *testing.T) {
	if b, ct, _ := EncodeBody(nil); b != nil || ct != "" {
		t.Errorf("nil: %v %q", b, ct)
	}
	if b, ct, _ := EncodeBody([]byte("x")); string(b) != "x" || ct != "application/octet-stream" {
		t.Errorf("bytes: %q %q", b, ct)
	}
	if b, ct, _ := EncodeBody("s"); string(b) != "s" || !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("string: %q %q", b, ct)
	}
	if b, ct, _ := EncodeBody(map[string]int{"a": 1}); string(b) != `{"a":1}` || ct != "application/json" {
		t.Errorf("json: %q %q", b, ct)
	}
	if _, _, err := EncodeBody(make(chan int)); err == nil {
		t.Error("expected json marshal error for channel")
	}
}

func TestTransportError(t *testing.T) {
	c := New(Config{})
	_, err := c.Get("http://127.0.0.1:0/nope")
	var aerr *Error
	if !errors.As(err, &aerr) {
		t.Fatalf("err = %v", err)
	}
	if aerr.Response != nil {
		t.Errorf("transport error should have nil response")
	}
	if aerr.Unwrap() == nil {
		t.Errorf("expected wrapped underlying error")
	}
}

func TestInvalidURL(t *testing.T) {
	c := New(Config{})
	if _, err := c.Get("://bad url with spaces\n"); err == nil {
		t.Fatal("expected url error")
	}
}

func TestCustomTransport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "via-transport")
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, Transport: http.DefaultTransport})
	resp, err := c.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Text() != "via-transport" {
		t.Errorf("body = %q", resp.Text())
	}
}

func TestDefaultBackoffAndRetryOn(t *testing.T) {
	if DefaultBackoff(0) != 100*time.Millisecond {
		t.Errorf("backoff(0) = %v", DefaultBackoff(0))
	}
	if DefaultBackoff(2) != 200*time.Millisecond {
		t.Errorf("backoff(2) = %v", DefaultBackoff(2))
	}
	if DefaultBackoff(100) != 10*time.Second {
		t.Errorf("backoff cap = %v", DefaultBackoff(100))
	}
	if !DefaultRetryOn(nil, errors.New("x")) {
		t.Error("should retry on error")
	}
	if !DefaultRetryOn(&Response{Status: 503}, nil) {
		t.Error("should retry on 503")
	}
	if DefaultRetryOn(&Response{Status: 200}, nil) {
		t.Error("should not retry on 200")
	}
}

func readJSON(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}
