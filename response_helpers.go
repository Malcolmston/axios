package axios

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// IsInformational reports whether the status code is in the 1xx range.
func (r *Response) IsInformational() bool {
	return r.Status >= 100 && r.Status < 200
}

// IsRedirect reports whether the status code is in the 3xx range.
func (r *Response) IsRedirect() bool {
	return r.Status >= 300 && r.Status < 400
}

// IsClientError reports whether the status code is in the 4xx range.
func (r *Response) IsClientError() bool {
	return r.Status >= 400 && r.Status < 500
}

// IsServerError reports whether the status code is in the 5xx range.
func (r *Response) IsServerError() bool {
	return r.Status >= 500 && r.Status < 600
}

// ContentType returns the raw Content-Type response header value, including any
// parameters (e.g. "application/json; charset=utf-8"), or "" if absent.
func (r *Response) ContentType() string {
	return r.Headers.Get("Content-Type")
}

// ContentLength returns the value of the Content-Length response header parsed
// as an integer, or -1 when the header is missing or not a valid number. Note
// this reflects the declared length, which may differ from len(Body) for
// decompressed or chunked responses.
func (r *Response) ContentLength() int64 {
	v := r.Headers.Get("Content-Length")
	if v == "" {
		return -1
	}
	n, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
	if err != nil {
		return -1
	}
	return n
}

// Location returns the URL from the Location response header, resolved relative
// to the request URL when possible. It returns an error if the header is
// absent or unparseable, mirroring http.Response.Location.
func (r *Response) Location() (*url.URL, error) {
	if r.Raw != nil {
		return r.Raw.Location()
	}
	loc := r.Headers.Get("Location")
	if loc == "" {
		return nil, http.ErrNoLocation
	}
	u, err := url.Parse(loc)
	if err != nil {
		return nil, err
	}
	if r.Request != nil && r.Request.URL != nil {
		return r.Request.URL.ResolveReference(u), nil
	}
	return u, nil
}

// Cookies parses and returns the cookies set by the response via Set-Cookie
// headers.
func (r *Response) Cookies() []*http.Cookie {
	if r.Raw != nil {
		return r.Raw.Cookies()
	}
	return (&http.Response{Header: r.Headers}).Cookies()
}

// RetryAfter interprets the Retry-After response header, returning the delay a
// client should wait before retrying and whether the header was present and
// valid. Both forms are supported: a number of seconds, and an HTTP-date. For
// the date form the delay is measured from the response's Date header when
// present, otherwise from the current time; a date in the past yields a zero
// duration.
func (r *Response) RetryAfter() (time.Duration, bool) {
	v := strings.TrimSpace(r.Headers.Get("Retry-After"))
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			secs = 0
		}
		return time.Duration(secs) * time.Second, true
	}
	t, err := http.ParseTime(v)
	if err != nil {
		return 0, false
	}
	base := time.Now()
	if d := r.Headers.Get("Date"); d != "" {
		if dt, err := http.ParseTime(d); err == nil {
			base = dt
		}
	}
	dur := t.Sub(base)
	if dur < 0 {
		dur = 0
	}
	return dur, true
}
