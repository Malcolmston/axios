package axios

import (
	"net/http"
	"testing"
	"time"
)

func TestResponseStatusClasses(t *testing.T) {
	tests := []struct {
		status                             int
		info, redirect, client, server, ok bool
	}{
		{100, true, false, false, false, false},
		{200, false, false, false, false, true},
		{301, false, true, false, false, false},
		{404, false, false, true, false, false},
		{503, false, false, false, true, false},
	}
	for _, tc := range tests {
		r := &Response{Status: tc.status}
		if r.IsInformational() != tc.info {
			t.Errorf("status %d IsInformational = %v", tc.status, r.IsInformational())
		}
		if r.IsRedirect() != tc.redirect {
			t.Errorf("status %d IsRedirect = %v", tc.status, r.IsRedirect())
		}
		if r.IsClientError() != tc.client {
			t.Errorf("status %d IsClientError = %v", tc.status, r.IsClientError())
		}
		if r.IsServerError() != tc.server {
			t.Errorf("status %d IsServerError = %v", tc.status, r.IsServerError())
		}
		if r.OK() != tc.ok {
			t.Errorf("status %d OK = %v", tc.status, r.OK())
		}
	}
}

func TestResponseContentHeaders(t *testing.T) {
	r := &Response{Headers: http.Header{
		"Content-Type":   {"application/json; charset=utf-8"},
		"Content-Length": {"42"},
	}}
	if r.ContentType() != "application/json; charset=utf-8" {
		t.Fatalf("ContentType = %q", r.ContentType())
	}
	if r.ContentLength() != 42 {
		t.Fatalf("ContentLength = %d", r.ContentLength())
	}

	empty := &Response{Headers: http.Header{}}
	if empty.ContentLength() != -1 {
		t.Fatalf("empty ContentLength = %d, want -1", empty.ContentLength())
	}
	bad := &Response{Headers: http.Header{"Content-Length": {"nope"}}}
	if bad.ContentLength() != -1 {
		t.Fatalf("bad ContentLength = %d, want -1", bad.ContentLength())
	}
}

func TestResponseLocation(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "https://ex.com/a/b", nil)
	r := &Response{
		Headers: http.Header{"Location": {"/c/d"}},
		Request: req,
	}
	loc, err := r.Location()
	if err != nil {
		t.Fatalf("Location: %v", err)
	}
	if loc.String() != "https://ex.com/c/d" {
		t.Fatalf("Location = %q", loc.String())
	}

	none := &Response{Headers: http.Header{}}
	if _, err := none.Location(); err == nil {
		t.Fatal("expected error when Location absent")
	}
}

func TestResponseCookies(t *testing.T) {
	r := &Response{Headers: http.Header{
		"Set-Cookie": {"session=abc; Path=/", "theme=dark"},
	}}
	cookies := r.Cookies()
	if len(cookies) != 2 {
		t.Fatalf("got %d cookies", len(cookies))
	}
	if cookies[0].Name != "session" || cookies[0].Value != "abc" {
		t.Fatalf("cookie[0] = %+v", cookies[0])
	}
}

func TestResponseRetryAfter(t *testing.T) {
	// Numeric seconds form.
	r := &Response{Headers: http.Header{"Retry-After": {"120"}}}
	d, ok := r.RetryAfter()
	if !ok || d != 120*time.Second {
		t.Fatalf("numeric RetryAfter = %v, ok=%v", d, ok)
	}

	// HTTP-date form measured against the Date header (deterministic).
	base := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	target := base.Add(90 * time.Second)
	r = &Response{Headers: http.Header{
		"Date":        {base.Format(http.TimeFormat)},
		"Retry-After": {target.Format(http.TimeFormat)},
	}}
	d, ok = r.RetryAfter()
	if !ok || d != 90*time.Second {
		t.Fatalf("date RetryAfter = %v, ok=%v", d, ok)
	}

	// Past date yields zero.
	pastTarget := base.Add(-30 * time.Second)
	r = &Response{Headers: http.Header{
		"Date":        {base.Format(http.TimeFormat)},
		"Retry-After": {pastTarget.Format(http.TimeFormat)},
	}}
	d, ok = r.RetryAfter()
	if !ok || d != 0 {
		t.Fatalf("past RetryAfter = %v, ok=%v", d, ok)
	}

	// Absent header.
	r = &Response{Headers: http.Header{}}
	if _, ok := r.RetryAfter(); ok {
		t.Fatal("expected ok=false when header absent")
	}
}
