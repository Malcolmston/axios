package axios

// GetJSON issues a GET request with the given client and decodes a successful
// JSON response into a value of type T, which it returns. If the request fails
// (transport error or rejected status) the zero value of T and the error are
// returned.
func GetJSON[T any](c *Client, rawURL string, cfg ...*RequestConfig) (T, error) {
	var out T
	resp, err := c.Get(rawURL, cfg...)
	if err != nil {
		return out, err
	}
	if err := resp.JSON(&out); err != nil {
		return out, &Error{Message: "decode json", Request: resp.Request, Response: resp, Err: err, Code: ErrCodeBadResponse}
	}
	return out, nil
}

// GetJSONDefault is GetJSON using the package-level default client.
func GetJSONDefault[T any](rawURL string, cfg ...*RequestConfig) (T, error) {
	return GetJSON[T](std, rawURL, cfg...)
}
