package axios

// The following package-level helpers delegate to the default client returned
// by Default. Replace it with SetDefault to change global behaviour.

// Get issues a GET request using the default client.
func Get(rawURL string, cfg ...*RequestConfig) (*Response, error) { return std.Get(rawURL, cfg...) }

// Delete issues a DELETE request using the default client.
func Delete(rawURL string, cfg ...*RequestConfig) (*Response, error) {
	return std.Delete(rawURL, cfg...)
}

// Head issues a HEAD request using the default client.
func Head(rawURL string, cfg ...*RequestConfig) (*Response, error) { return std.Head(rawURL, cfg...) }

// Options issues an OPTIONS request using the default client.
func Options(rawURL string, cfg ...*RequestConfig) (*Response, error) {
	return std.Options(rawURL, cfg...)
}

// Post issues a POST request using the default client.
func Post(rawURL string, body any, cfg ...*RequestConfig) (*Response, error) {
	return std.Post(rawURL, body, cfg...)
}

// Put issues a PUT request using the default client.
func Put(rawURL string, body any, cfg ...*RequestConfig) (*Response, error) {
	return std.Put(rawURL, body, cfg...)
}

// Patch issues a PATCH request using the default client.
func Patch(rawURL string, body any, cfg ...*RequestConfig) (*Response, error) {
	return std.Patch(rawURL, body, cfg...)
}
