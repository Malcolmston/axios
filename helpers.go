package axios

import (
	"net"
	"net/http"
	"net/url"
	"strconv"
	"sync"
)

// HeaderDefaults holds default headers grouped the way axios groups them under
// defaults.headers: Common headers apply to every request, and the per-method
// groups apply only to requests using that method. Method-specific headers take
// precedence over Common, and both are overridden by Config.Headers and
// per-request headers.
type HeaderDefaults struct {
	// Common headers are applied to every request regardless of method.
	Common http.Header
	// Get headers are applied only to GET requests.
	Get http.Header
	// Post headers are applied only to POST requests.
	Post http.Header
	// Put headers are applied only to PUT requests.
	Put http.Header
	// Patch headers are applied only to PATCH requests.
	Patch http.Header
	// Delete headers are applied only to DELETE requests.
	Delete http.Header
	// Head headers are applied only to HEAD requests.
	Head http.Header
	// Options headers are applied only to OPTIONS requests.
	Options http.Header
}

// forMethod returns the ordered header groups that apply to method: Common
// first, then the method-specific group (so method headers win on conflict).
func (h HeaderDefaults) forMethod(method string) []http.Header {
	groups := []http.Header{h.Common}
	switch method {
	case http.MethodGet:
		groups = append(groups, h.Get)
	case http.MethodPost:
		groups = append(groups, h.Post)
	case http.MethodPut:
		groups = append(groups, h.Put)
	case http.MethodPatch:
		groups = append(groups, h.Patch)
	case http.MethodDelete:
		groups = append(groups, h.Delete)
	case http.MethodHead:
		groups = append(groups, h.Head)
	case http.MethodOptions:
		groups = append(groups, h.Options)
	}
	return groups
}

// DefaultHeaderGroups returns a HeaderDefaults pre-populated the way axios does:
// a common Accept of application/json. Callers can extend the returned value.
func DefaultHeaderGroups() HeaderDefaults {
	return HeaderDefaults{
		Common: http.Header{"Accept": {"application/json, text/plain, */*"}},
	}
}

// ProxyConfig configures an outbound HTTP proxy, mirroring the axios proxy
// option. It is used only when the client builds its own *http.Client (that is,
// when Config.HTTPClient is nil).
type ProxyConfig struct {
	// Host is the proxy host name or IP (required).
	Host string
	// Port is the proxy port. When 0 the URL is built without an explicit port.
	Port int
	// Protocol is the proxy URL scheme; defaults to "http" when empty.
	Protocol string
	// Username, when set, adds proxy Basic credentials.
	Username string
	// Password is the proxy Basic password.
	Password string
}

// URL renders the proxy as a *url.URL suitable for http.Transport.Proxy, or nil
// when Host is empty.
func (p *ProxyConfig) URL() *url.URL {
	if p == nil || p.Host == "" {
		return nil
	}
	scheme := p.Protocol
	if scheme == "" {
		scheme = "http"
	}
	host := p.Host
	if p.Port != 0 {
		host = net.JoinHostPort(p.Host, strconv.Itoa(p.Port))
	}
	u := &url.URL{Scheme: scheme, Host: host}
	if p.Username != "" {
		u.User = url.UserPassword(p.Username, p.Password)
	}
	return u
}

// All runs the given request-producing functions concurrently and returns their
// responses in the same order, mirroring axios.all. If any call returns an
// error, All returns the responses gathered so far (with nil in the failed
// slots) and the first error encountered.
func All(calls ...func() (*Response, error)) ([]*Response, error) {
	results := make([]*Response, len(calls))
	errs := make([]error, len(calls))
	var wg sync.WaitGroup
	for i, call := range calls {
		wg.Add(1)
		go func(i int, call func() (*Response, error)) {
			defer wg.Done()
			results[i], errs[i] = call()
		}(i, call)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

// Spread adapts a variadic callback so it can consume the slice returned by All,
// mirroring axios.spread. For example:
//
//	resps, err := axios.All(a, b)
//	names := axios.Spread(func(rs ...*axios.Response) []int {
//		return []int{rs[0].Status, rs[1].Status}
//	})(resps)
func Spread[T any](fn func(...*Response) T) func([]*Response) T {
	return func(resps []*Response) T {
		return fn(resps...)
	}
}
