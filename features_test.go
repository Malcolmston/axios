package axios

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---- Cancellation -----------------------------------------------------------

func TestAbortControllerCancels(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(500 * time.Millisecond)
	}))
	defer srv.Close()

	ctrl := NewAbortController()
	c := New(Config{BaseURL: srv.URL})
	go func() {
		<-started
		ctrl.Abort(nil)
	}()
	_, err := c.Get("/", &RequestConfig{Signal: ctrl.Signal()})
	if err == nil {
		t.Fatal("expected abort error")
	}
	if !IsCancel(err) {
		t.Errorf("IsCancel = false for %v", err)
	}
	var aerr *Error
	if errors.As(err, &aerr) && aerr.Code != ErrCodeCanceled {
		t.Errorf("code = %q, want %q", aerr.Code, ErrCodeCanceled)
	}
	if !ctrl.Signal().Aborted() {
		t.Error("signal should report aborted")
	}
}

func TestCancelTokenCancels(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(500 * time.Millisecond)
	}))
	defer srv.Close()

	tok, cancel := NewCancelToken()
	c := New(Config{BaseURL: srv.URL})
	go func() {
		<-started
		cancel("user navigated away")
	}()
	_, err := c.Get("/", &RequestConfig{CancelToken: tok})
	if err == nil {
		t.Fatal("expected cancel error")
	}
	if !IsCancel(err) {
		t.Errorf("IsCancel = false for %v", err)
	}
}

func TestClientLevelSignal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	ctrl := NewAbortController()
	ctrl.Abort(errors.New("boom"))
	c := New(Config{BaseURL: srv.URL, Signal: ctrl.Signal()})
	if _, err := c.Get("/"); err == nil {
		t.Fatal("expected error from pre-aborted client signal")
	}
	if ctrl.Signal().Err() == nil {
		t.Error("signal Err should be set")
	}
}

// ---- Progress ---------------------------------------------------------------

func TestUploadAndDownloadProgress(t *testing.T) {
	payload := strings.Repeat("x", 4096)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		_, _ = io.WriteString(w, payload)
	}))
	defer srv.Close()

	var up, down ProgressEvent
	var upCalls, downCalls int
	c := New(Config{BaseURL: srv.URL})
	resp, err := c.Post("/", []byte(payload), &RequestConfig{
		OnUploadProgress:   func(e ProgressEvent) { up = e; upCalls++ },
		OnDownloadProgress: func(e ProgressEvent) { down = e; downCalls++ },
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if upCalls == 0 || up.Loaded != int64(len(payload)) {
		t.Errorf("upload progress: calls=%d loaded=%d want %d", upCalls, up.Loaded, len(payload))
	}
	if !up.LengthComputable || up.Progress() != 1 {
		t.Errorf("upload progress not complete: %+v", up)
	}
	if downCalls == 0 || down.Loaded != int64(len(payload)) {
		t.Errorf("download progress: calls=%d loaded=%d want %d", downCalls, down.Loaded, len(payload))
	}
	if down.Total != int64(len(payload)) || down.Progress() != 1 {
		t.Errorf("download progress not complete: %+v", down)
	}
	if resp.Text() != payload {
		t.Errorf("body mismatch")
	}
}

func TestProgressEventUnknownTotal(t *testing.T) {
	e := ProgressEvent{Loaded: 5, Total: -1}
	if e.LengthComputable || e.Progress() != 0 {
		t.Errorf("unknown total should give progress 0, computable false: %+v", e)
	}
}

// ---- Transforms -------------------------------------------------------------

func TestTransformRequestAndResponse(t *testing.T) {
	var gotBody, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotCT = r.Header.Get("Content-Type")
		_, _ = io.WriteString(w, "world")
	}))
	defer srv.Close()

	c := New(Config{
		BaseURL: srv.URL,
		TransformRequest: []RequestTransform{
			func(body []byte, h http.Header) ([]byte, error) {
				h.Set("Content-Type", "text/x-upper")
				return bytes.ToUpper(body), nil
			},
		},
		TransformResponse: []ResponseTransform{
			func(body []byte, h http.Header) ([]byte, error) {
				return []byte("[" + string(body) + "]"), nil
			},
		},
	})
	resp, err := c.Post("/", "hello")
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if gotBody != "HELLO" {
		t.Errorf("transformed request body = %q", gotBody)
	}
	if gotCT != "text/x-upper" {
		t.Errorf("transform-set content-type = %q", gotCT)
	}
	if resp.Text() != "[world]" {
		t.Errorf("transformed response = %q", resp.Text())
	}
}

func TestTransformRequestError(t *testing.T) {
	c := New(Config{TransformRequest: []RequestTransform{
		func(body []byte, h http.Header) ([]byte, error) { return nil, errors.New("nope") },
	}})
	_, err := c.Post("http://example.invalid/", "x")
	if err == nil || !strings.Contains(err.Error(), "transform request") {
		t.Fatalf("err = %v", err)
	}
}

// ---- Param serialization ----------------------------------------------------

func TestSerializeParamsFormats(t *testing.T) {
	params := url.Values{"a": {"1", "2"}, "b": {"x"}}
	cases := map[ArrayFormat]string{
		ArrayFormatRepeat:   "a=1&a=2&b=x",
		ArrayFormatBrackets: "a%5B%5D=1&a%5B%5D=2&b=x",
		ArrayFormatIndices:  "a%5B0%5D=1&a%5B1%5D=2&b=x",
		ArrayFormatComma:    "a=1%2C2&b=x",
	}
	for format, want := range cases {
		if got := SerializeParams(params, format); got != want {
			t.Errorf("format %d = %q, want %q", format, got, want)
		}
	}
	if SerializeParams(nil, ArrayFormatRepeat) != "" {
		t.Error("empty params should serialize to empty string")
	}
}

func TestBuildURLArrayFormatAndSerializer(t *testing.T) {
	c := New(Config{BaseURL: "https://x.test"})
	bf := ArrayFormatBrackets
	got, err := c.GetUri("/p", &RequestConfig{Params: url.Values{"id": {"1", "2"}}, ArrayFormat: &bf})
	if err != nil {
		t.Fatalf("GetUri: %v", err)
	}
	if !strings.Contains(got, "id%5B%5D=1") || !strings.Contains(got, "id%5B%5D=2") {
		t.Errorf("brackets url = %q", got)
	}

	custom, err := c.GetUri("/p", &RequestConfig{
		Params:           url.Values{"q": {"a"}},
		ParamsSerializer: func(v url.Values) string { return "raw=" + v.Get("q") },
	})
	if err != nil {
		t.Fatalf("GetUri: %v", err)
	}
	if !strings.HasSuffix(custom, "raw=a") {
		t.Errorf("custom serializer url = %q", custom)
	}
}

func TestFlattenParams(t *testing.T) {
	m := map[string]any{
		"filter": map[string]any{"name": "ada", "active": true},
		"ids":    []any{1, 2},
		"skip":   nil,
	}
	v := FlattenParams(m)
	if v.Get("filter[name]") != "ada" {
		t.Errorf("filter[name] = %q", v.Get("filter[name]"))
	}
	if v.Get("filter[active]") != "true" {
		t.Errorf("filter[active] = %q", v.Get("filter[active]"))
	}
	if got := v["ids"]; len(got) != 2 || got[0] != "1" || got[1] != "2" {
		t.Errorf("ids = %v", got)
	}
	if _, ok := v["skip"]; ok {
		t.Error("nil param should be skipped")
	}
}

func TestParamsMapInRequest(t *testing.T) {
	c := New(Config{BaseURL: "https://x.test"})
	got, err := c.GetUri("/p", &RequestConfig{ParamsMap: map[string]any{"page": 3}})
	if err != nil {
		t.Fatalf("GetUri: %v", err)
	}
	if !strings.Contains(got, "page=3") {
		t.Errorf("url = %q", got)
	}
}

// ---- Decompression ----------------------------------------------------------

func TestGzipDecompression(t *testing.T) {
	original := "the quick brown fox jumps over the lazy dog"
	var gotAE string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAE = r.Header.Get("Accept-Encoding")
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte(original))
		_ = gz.Close()
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	resp, err := c.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Text() != original {
		t.Errorf("decompressed = %q", resp.Text())
	}
	if !strings.Contains(gotAE, "gzip") || !strings.Contains(gotAE, "deflate") {
		t.Errorf("accept-encoding = %q", gotAE)
	}
	if resp.Header("Content-Encoding") != "" {
		t.Errorf("content-encoding should be stripped, got %q", resp.Header("Content-Encoding"))
	}
}

func TestDeflateDecompressionZlib(t *testing.T) {
	original := "deflate via zlib framing"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "deflate")
		zw := zlib.NewWriter(w)
		_, _ = zw.Write([]byte(original))
		_ = zw.Close()
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	resp, err := c.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Text() != original {
		t.Errorf("zlib deflate = %q", resp.Text())
	}
}

func TestDeflateDecompressionRaw(t *testing.T) {
	original := "raw deflate rfc1951"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "deflate")
		fw, _ := flate.NewWriter(w, flate.DefaultCompression)
		_, _ = fw.Write([]byte(original))
		_ = fw.Close()
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	resp, err := c.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Text() != original {
		t.Errorf("raw deflate = %q", resp.Text())
	}
}

func TestDecompressDisabledNoAcceptEncoding(t *testing.T) {
	var gotAE string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAE = r.Header.Get("Accept-Encoding")
		_, _ = io.WriteString(w, "plain")
	}))
	defer srv.Close()
	no := false
	c := New(Config{BaseURL: srv.URL, Decompress: &no})
	resp, err := c.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Text() != "plain" {
		t.Errorf("body = %q", resp.Text())
	}
	if gotAE != "" {
		t.Errorf("accept-encoding should be unset when decompress disabled, got %q", gotAE)
	}
}

// ---- Redirects --------------------------------------------------------------

func redirectChainServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
		if err != nil || n <= 0 {
			_, _ = io.WriteString(w, "done")
			return
		}
		http.Redirect(w, r, "/"+strconv.Itoa(n-1), http.StatusFound)
	}))
}

func TestRedirectCapExceeded(t *testing.T) {
	srv := redirectChainServer()
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, MaxRedirects: 2})
	if _, err := c.Get("/3"); err == nil {
		t.Fatal("expected redirect cap error")
	}
}

func TestRedirectWithinCap(t *testing.T) {
	srv := redirectChainServer()
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, MaxRedirects: 10})
	resp, err := c.Get("/3")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Text() != "done" {
		t.Errorf("body = %q", resp.Text())
	}
}

func TestRedirectDisabled(t *testing.T) {
	srv := redirectChainServer()
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, MaxRedirects: -1, ValidateStatus: func(int) bool { return true }})
	resp, err := c.Get("/3")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Status != http.StatusFound {
		t.Errorf("status = %d, want 302", resp.Status)
	}
	if resp.Header("Location") == "" {
		t.Error("expected Location header on unfollowed redirect")
	}
}

func TestRedirectPolicyOverride(t *testing.T) {
	srv := redirectChainServer()
	defer srv.Close()
	sentinel := errors.New("policy stop")
	c := New(Config{BaseURL: srv.URL, RedirectPolicy: func(req *http.Request, via []*http.Request) error {
		return sentinel
	}})
	if _, err := c.Get("/2"); err == nil {
		t.Fatal("expected policy error")
	}
}

// ---- Size guards ------------------------------------------------------------

func TestMaxContentLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, strings.Repeat("y", 200))
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL, MaxContentLength: 50})
	if _, err := c.Get("/"); err == nil {
		t.Fatal("expected content length error")
	}
}

func TestMaxBodyLength(t *testing.T) {
	c := New(Config{MaxBodyLength: 4})
	_, err := c.Post("http://example.invalid/", []byte("too big body"))
	var aerr *Error
	if !errors.As(err, &aerr) || aerr.Code != ErrCodeBadRequest {
		t.Fatalf("err = %v", err)
	}
}

// ---- XSRF -------------------------------------------------------------------

func TestXSRFCookieToHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-CSRF-Token")
	}))
	defer srv.Close()

	jar, _ := cookiejar.New(nil)
	u, _ := url.Parse(srv.URL)
	jar.SetCookies(u, []*http.Cookie{{Name: "csrftoken", Value: "secret123"}})

	c := New(Config{
		BaseURL:        srv.URL,
		HTTPClient:     &http.Client{Jar: jar},
		XSRFCookieName: "csrftoken",
		XSRFHeaderName: "X-CSRF-Token",
	})
	if _, err := c.Get("/"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if gotHeader != "secret123" {
		t.Errorf("xsrf header = %q", gotHeader)
	}
}

// ---- All / Spread -----------------------------------------------------------

func TestAllAndSpread(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "path:"+r.URL.Path)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})

	resps, err := All(
		func() (*Response, error) { return c.Get("/a") },
		func() (*Response, error) { return c.Get("/b") },
	)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	joined := Spread(func(rs ...*Response) string {
		return rs[0].Text() + "," + rs[1].Text()
	})(resps)
	if joined != "path:/a,path:/b" {
		t.Errorf("spread = %q", joined)
	}
}

func TestAllReturnsFirstError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bad" {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	_, err := All(
		func() (*Response, error) { return c.Get("/ok") },
		func() (*Response, error) { return c.Get("/bad") },
	)
	if err == nil {
		t.Fatal("expected error from All")
	}
}

// ---- Create / merge ---------------------------------------------------------

func TestCreateDeepMerge(t *testing.T) {
	var h http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h = r.Header.Clone()
	}))
	defer srv.Close()

	base := New(Config{
		BaseURL: srv.URL,
		Headers: http.Header{"X-Base": {"1"}, "X-Override": {"base"}},
		Params:  url.Values{"shared": {"base"}},
	})
	child := base.Create(Config{
		Headers: http.Header{"X-Child": {"2"}, "X-Override": {"child"}},
	})
	if _, err := child.Get("/"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if h.Get("X-Base") != "1" {
		t.Errorf("base header lost: %q", h.Get("X-Base"))
	}
	if h.Get("X-Child") != "2" {
		t.Errorf("child header missing: %q", h.Get("X-Child"))
	}
	if h.Get("X-Override") != "child" {
		t.Errorf("override header = %q, want child", h.Get("X-Override"))
	}
}

func TestHeaderGroups(t *testing.T) {
	var accept, xget string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		xget = r.Header.Get("X-Get-Only")
	}))
	defer srv.Close()
	groups := DefaultHeaderGroups()
	groups.Get = http.Header{"X-Get-Only": {"yes"}}
	c := New(Config{BaseURL: srv.URL, HeaderGroups: groups})
	if _, err := c.Get("/"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !strings.Contains(accept, "application/json") {
		t.Errorf("common accept = %q", accept)
	}
	if xget != "yes" {
		t.Errorf("method group header = %q", xget)
	}
	// POST should not receive the GET-only header.
	if _, err := c.Post("/", nil); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if xget != "" {
		t.Errorf("GET-only header leaked to POST: %q", xget)
	}
}

// ---- Error helpers ----------------------------------------------------------

func TestIsAxiosErrorAndToJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	_, err := c.Get("/x")
	if !IsAxiosError(err) {
		t.Fatal("IsAxiosError = false")
	}
	if IsAxiosError(errors.New("plain")) {
		t.Error("plain error should not be axios error")
	}
	aerr := AsError(err)
	if aerr == nil {
		t.Fatal("AsError = nil")
	}
	j := aerr.ToJSON()
	if j["name"] != "AxiosError" || j["status"] != 400 {
		t.Errorf("toJSON = %v", j)
	}
	if j["code"] != ErrCodeBadResponse {
		t.Errorf("code = %v", j["code"])
	}
	// MarshalJSON should round-trip.
	b, merr := json.Marshal(aerr)
	if merr != nil || !strings.Contains(string(b), "AxiosError") {
		t.Errorf("marshal = %s err=%v", b, merr)
	}
}

// ---- ResponseType stream ----------------------------------------------------

func TestResponseTypeStream(t *testing.T) {
	body := strings.Repeat("stream-chunk;", 500)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	st := ResponseStream
	resp, err := c.Get("/", &RequestConfig{ResponseType: &st})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Stream == nil {
		t.Fatal("expected Stream to be set")
	}
	data, err := io.ReadAll(resp.Stream)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}
	if string(data) != body {
		t.Errorf("stream body mismatch: %d vs %d bytes", len(data), len(body))
	}
	if err := resp.Close(); err != nil {
		t.Errorf("close: %v", err)
	}
	if err := resp.Close(); err != nil {
		t.Errorf("second close should be no-op: %v", err)
	}
}

func TestStreamGzipDecompression(t *testing.T) {
	original := "streamed and gzipped"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		_, _ = gz.Write([]byte(original))
		_ = gz.Close()
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	st := ResponseStream
	resp, err := c.Get("/", &RequestConfig{ResponseType: &st})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Close() }()
	if resp.Text() != original {
		t.Errorf("stream gzip = %q", resp.Text())
	}
}

// ---- Proxy ------------------------------------------------------------------

func TestProxyConfigWiring(t *testing.T) {
	var proxiedURI string
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxiedURI = r.RequestURI
		_, _ = io.WriteString(w, "PROXIED")
	}))
	defer proxy.Close()

	pu, _ := url.Parse(proxy.URL)
	host, portStr, _ := splitHostPort(pu.Host)
	port, _ := strconv.Atoi(portStr)
	c := New(Config{Proxy: &ProxyConfig{Host: host, Port: port}})
	resp, err := c.Get("http://example.com/widget")
	if err != nil {
		t.Fatalf("Get via proxy: %v", err)
	}
	if resp.Text() != "PROXIED" {
		t.Errorf("body = %q", resp.Text())
	}
	if !strings.Contains(proxiedURI, "example.com") {
		t.Errorf("proxy did not receive absolute URI: %q", proxiedURI)
	}
}

func TestProxyURL(t *testing.T) {
	p := &ProxyConfig{Host: "127.0.0.1", Port: 8080, Username: "u", Password: "p"}
	u := p.URL()
	if u.Scheme != "http" || u.Host != "127.0.0.1:8080" {
		t.Errorf("proxy url = %v", u)
	}
	if pw, _ := u.User.Password(); u.User.Username() != "u" || pw != "p" {
		t.Errorf("proxy creds = %v", u.User)
	}
	if (&ProxyConfig{}).URL() != nil {
		t.Error("empty proxy should yield nil URL")
	}
}

func splitHostPort(hostport string) (host, port string, err error) {
	i := strings.LastIndex(hostport, ":")
	if i < 0 {
		return hostport, "", fmt.Errorf("no port")
	}
	return hostport[:i], hostport[i+1:], nil
}
