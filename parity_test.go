package axios

import (
	"net/url"
	"reflect"
	"testing"
	"time"
)

// TestParityIsAbsoluteURL mirrors axios test/specs/helpers/isAbsoluteURL.spec.js.
func TestParityIsAbsoluteURL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		// begins with valid scheme name
		{"https://api.github.com/users", true},
		{"custom-scheme-v1.0://example.com/", true},
		{"HTTP://example.com/", true},
		// invalid scheme name
		{"123://example.com/", false},
		{"!valid://example.com/", false},
		// protocol-relative
		{"//example.com/", true},
		// relative
		{"/foo", false},
		{"foo", false},
	}
	for _, tc := range cases {
		if got := IsAbsoluteURL(tc.in); got != tc.want {
			t.Errorf("IsAbsoluteURL(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

// TestParityCombineURLs mirrors axios test/specs/helpers/combineURLs.spec.js.
func TestParityCombineURLs(t *testing.T) {
	cases := []struct {
		base, rel, want string
	}{
		{"https://api.github.com", "/users", "https://api.github.com/users"},
		{"https://api.github.com/", "/users", "https://api.github.com/users"},
		{"https://api.github.com", "users", "https://api.github.com/users"},
		{"https://api.github.com/users", "", "https://api.github.com/users"},
		{"https://api.github.com/users", "/", "https://api.github.com/users/"},
	}
	for _, tc := range cases {
		if got := CombineURLs(tc.base, tc.rel); got != tc.want {
			t.Errorf("CombineURLs(%q, %q) = %q, want %q", tc.base, tc.rel, got, tc.want)
		}
	}
}

// TestParityBuildFullPath mirrors axios test/specs/core/buildFullPath.spec.js.
func TestParityBuildFullPath(t *testing.T) {
	cases := []struct {
		base, req     string
		allowAbsolute bool
		want          string
	}{
		{"https://api.github.com", "/users", true, "https://api.github.com/users"},
		{"https://api.github.com", "https://api.example.com/users", true, "https://api.example.com/users"},
		{"https://api.github.com", "https://api.example.com/users", false, "https://api.github.com/https://api.example.com/users"},
		{"", "https://api.example.com/users", false, "https://api.example.com/users"},
		{"", "/users", true, "/users"},
		{"/api", "/users", true, "/api/users"},
	}
	for _, tc := range cases {
		if got := BuildFullPath(tc.base, tc.req, tc.allowAbsolute); got != tc.want {
			t.Errorf("BuildFullPath(%q, %q, %v) = %q, want %q", tc.base, tc.req, tc.allowAbsolute, got, tc.want)
		}
	}
}

// TestParityBuildURL mirrors axios test/specs/helpers/buildURL.spec.js. Because
// the upstream serializer walks a JavaScript object's own key order (which Go
// maps cannot preserve), BuildURL emits keys in sorted order; the single
// order-sensitive vector is asserted order-independently below.
func TestParityBuildURL(t *testing.T) {
	// null params
	if got := BuildURL("/foo", nil, ArrayFormatBrackets); got != "/foo" {
		t.Errorf("null params: got %q", got)
	}
	// support params, undefined/null omitted (Go cannot represent them, so we
	// pass only the surviving key).
	if got := BuildURL("/foo", url.Values{"foo": {"bar"}}, ArrayFormatBrackets); got != "/foo?foo=bar" {
		t.Errorf("params: got %q", got)
	}
	// object params -> foo[bar]=baz (brackets percent-encoded).
	if got := BuildURL("/foo", url.Values{"foo[bar]": {"baz"}}, ArrayFormatBrackets); got != "/foo?foo%5Bbar%5D=baz" {
		t.Errorf("object params: got %q", got)
	}
	// array params with brackets encoding.
	if got := BuildURL("/foo", url.Values{"foo": {"bar", "baz"}}, ArrayFormatBrackets); got != "/foo?foo%5B%5D=bar&foo%5B%5D=baz" {
		t.Errorf("array params: got %q", got)
	}
	// special char params: ':' '$' ',' kept literal, space -> '+'.
	if got := BuildURL("/foo", url.Values{"foo": {":$, "}}, ArrayFormatBrackets); got != "/foo?foo=:$,+" {
		t.Errorf("special char: got %q", got)
	}
	// existing params: append with '&'.
	if got := BuildURL("/foo?foo=bar", url.Values{"bar": {"baz"}}, ArrayFormatBrackets); got != "/foo?foo=bar&bar=baz" {
		t.Errorf("existing params: got %q", got)
	}
	// discard url hash mark.
	if got := BuildURL("/foo?foo=bar#hash", url.Values{"query": {"baz"}}, ArrayFormatBrackets); got != "/foo?foo=bar&query=baz" {
		t.Errorf("hash discard: got %q", got)
	}
	// URLSearchParams-style single pair.
	if got := BuildURL("/foo", url.Values{"bar": {"baz"}}, ArrayFormatBrackets); got != "/foo?bar=baz" {
		t.Errorf("urlsearchparams: got %q", got)
	}
	// date params: ISO-8601 with ':' kept unescaped.
	date := time.Date(2026, 7, 18, 12, 34, 56, 0, time.UTC)
	iso := date.Format("2006-01-02T15:04:05.000Z07:00")
	if got := BuildURL("/foo", url.Values{"date": {iso}}, ArrayFormatBrackets); got != "/foo?date="+iso {
		t.Errorf("date params: got %q, want %q", got, "/foo?date="+iso)
	}
	// "length" parameter with numbers; order-independent check.
	got := BuildURL("/foo", url.Values{"query": {"bar"}, "start": {"0"}, "length": {"5"}}, ArrayFormatBrackets)
	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse %q: %v", got, err)
	}
	if u.Path != "/foo" {
		t.Errorf("length params path = %q", u.Path)
	}
	q := u.Query()
	for k, want := range map[string]string{"query": "bar", "start": "0", "length": "5"} {
		if q.Get(k) != want {
			t.Errorf("length params %s = %q, want %q", k, q.Get(k), want)
		}
	}
}

// TestParityEncodeURIComponent pins the axios buildURL encode character rules.
func TestParityEncodeURIComponent(t *testing.T) {
	cases := []struct{ in, want string }{
		{":$, ", ":$,+"},
		{"foo[bar]", "foo%5Bbar%5D"},
		{"a b", "a+b"},
		{"a+b", "a%2Bb"},
	}
	for _, tc := range cases {
		if got := EncodeURIComponent(tc.in); got != tc.want {
			t.Errorf("EncodeURIComponent(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestParityFormDataToJSON mirrors axios test/specs/helpers/formDataToJSON.spec.js.
func TestParityFormDataToJSON(t *testing.T) {
	t.Run("nested object", func(t *testing.T) {
		got := FormToJSON(url.Values{"foo[bar][baz]": {"123"}})
		want := map[string]any{"foo": map[string]any{"bar": map[string]any{"baz": "123"}}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("repeatable values", func(t *testing.T) {
		got := FormToJSON(url.Values{"foo": {"1", "2"}})
		want := map[string]any{"foo": []any{"1", "2"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("empty brackets to arrays", func(t *testing.T) {
		got := FormToJSON(url.Values{"foo[]": {"1", "2"}})
		want := map[string]any{"foo": []any{"1", "2"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("indexed arrays", func(t *testing.T) {
		got := FormToJSON(url.Values{"foo[0]": {"1"}, "foo[1]": {"2"}})
		want := map[string]any{"foo": []any{"1", "2"}}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v", got)
		}
	})
	t.Run("resist prototype pollution", func(t *testing.T) {
		got := FormToJSON(url.Values{
			"foo[0]":                  {"1"},
			"foo[1]":                  {"2"},
			"__proto__.x":             {"hack"},
			"constructor.prototype.y": {"value"},
		})
		want := map[string]any{
			"foo":         []any{"1", "2"},
			"constructor": map[string]any{"prototype": map[string]any{"y": "value"}},
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v", got)
		}
	})
}
