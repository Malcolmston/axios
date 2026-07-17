package axios

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPackageLevelCreate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Header.Get("X-Tag"))
	}))
	defer srv.Close()

	old := Default()
	SetDefault(New(Config{Headers: http.Header{"X-Tag": {"root"}}}))
	defer SetDefault(old)

	child := Create(Config{BaseURL: srv.URL})
	resp, err := child.Get("/")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Text() != "root" {
		t.Errorf("inherited header body = %q", resp.Text())
	}
}

func TestCanceledError(t *testing.T) {
	e := &CanceledError{Message: "stop"}
	if !strings.Contains(e.Error(), "stop") {
		t.Errorf("error = %q", e.Error())
	}
	empty := &CanceledError{}
	if empty.Error() == "" {
		t.Error("empty canceled error should still describe itself")
	}
	if !IsCancel(e) {
		t.Error("CanceledError should be a cancel")
	}
	if IsCancel(nil) {
		t.Error("nil is not a cancel")
	}
}

func TestCancelTokenEmptyMessage(t *testing.T) {
	tok, cancel := NewCancelToken()
	cancel("")
	if tok.Context().Err() == nil {
		t.Error("empty-message cancel should still cancel")
	}
}

func TestNilAbortSignalMethods(t *testing.T) {
	var s *AbortSignal
	if s.Aborted() || s.Err() != nil {
		t.Error("nil signal should be inert")
	}
	if s.Context() == nil {
		t.Error("nil signal context should be non-nil")
	}
}

func TestMergeConfigManyFields(t *testing.T) {
	base := Config{
		BaseURL:      "https://base.test",
		BearerToken:  "basetok",
		Timeout:      time.Second,
		ArrayFormat:  ArrayFormatRepeat,
		ResponseType: ResponseDefault,
		Params:       url.Values{"a": {"1"}},
		ParamsMap:    map[string]any{"m": 1},
		HeaderGroups: DefaultHeaderGroups(),
		RequestInterceptors: []RequestInterceptor{
			func(*http.Request) error { return nil },
		},
	}
	over := Config{
		BaseURL:          "https://over.test",
		Timeout:          2 * time.Second,
		ArrayFormat:      ArrayFormatBrackets,
		ResponseType:     ResponseStream,
		MaxContentLength: 100,
		MaxBodyLength:    50,
		XSRFCookieName:   "c",
		XSRFHeaderName:   "H",
		Params:           url.Values{"b": {"2"}},
		ParamsMap:        map[string]any{"n": 2},
		ResponseInterceptors: []ResponseInterceptor{
			func(*Response) error { return nil },
		},
	}
	m := mergeConfig(base, over)
	if m.BaseURL != "https://over.test" {
		t.Errorf("baseurl = %q", m.BaseURL)
	}
	if m.BearerToken != "basetok" {
		t.Errorf("bearer lost: %q", m.BearerToken)
	}
	if m.Timeout != 2*time.Second || m.ArrayFormat != ArrayFormatBrackets || m.ResponseType != ResponseStream {
		t.Errorf("scalar overrides wrong: %+v", m)
	}
	if m.MaxContentLength != 100 || m.MaxBodyLength != 50 {
		t.Errorf("size guards not merged")
	}
	if m.Params.Get("a") != "1" || m.Params.Get("b") != "2" {
		t.Errorf("params merge = %v", m.Params)
	}
	if m.ParamsMap["m"] != 1 || m.ParamsMap["n"] != 2 {
		t.Errorf("paramsmap merge = %v", m.ParamsMap)
	}
	if len(m.RequestInterceptors) != 1 || len(m.ResponseInterceptors) != 1 {
		t.Errorf("interceptors merge = %d/%d", len(m.RequestInterceptors), len(m.ResponseInterceptors))
	}
}

func TestStringifyParamVariants(t *testing.T) {
	m := map[string]any{
		"i":   42,
		"i64": int64(9000000000),
		"f":   3.5,
		"b":   false,
		"x":   struct{ A int }{1},
	}
	v := FlattenParams(m)
	if v.Get("i") != "42" || v.Get("i64") != "9000000000" || v.Get("f") != "3.5" || v.Get("b") != "false" {
		t.Errorf("stringify = %v", v)
	}
	if v.Get("x") == "" {
		t.Error("struct param should stringify via fmt")
	}
}

func TestStreamingJSONDecode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"name":"Ada","age":36}`)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})
	st := ResponseStream
	resp, err := c.Get("/", &RequestConfig{ResponseType: &st})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	defer func() { _ = resp.Close() }()
	var u user
	if err := resp.JSON(&u); err != nil {
		t.Fatalf("JSON: %v", err)
	}
	if u.Name != "Ada" || u.Age != 36 {
		t.Errorf("decoded = %+v", u)
	}
}

func TestNewAbortControllerWithContextNilParent(t *testing.T) {
	var noParent context.Context // exercise the documented nil-parent path
	ctrl := NewAbortControllerWithContext(noParent)
	if ctrl.Signal().Aborted() {
		t.Error("fresh controller should not be aborted")
	}
	parent, cancel := context.WithCancel(context.Background())
	child := NewAbortControllerWithContext(parent)
	cancel()
	select {
	case <-child.Signal().Context().Done():
	case <-time.After(time.Second):
		t.Error("child signal should cancel when parent cancels")
	}
}

func TestParentContextErr(t *testing.T) {
	e := &Error{Message: "x"}
	if e.StatusCode() != 0 {
		t.Errorf("no-response status = %d", e.StatusCode())
	}
	if !strings.HasPrefix(e.Error(), "axios:") {
		t.Errorf("error string = %q", e.Error())
	}
	wrapped := &Error{Message: "y", Err: errors.New("inner")}
	if !strings.Contains(wrapped.Error(), "inner") {
		t.Errorf("wrapped error = %q", wrapped.Error())
	}
}
