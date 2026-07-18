package axios

// PostJSON issues a POST request with the given body and decodes a successful
// JSON response into a value of type T. It complements GetJSON, mirroring the
// typed axios.post<T> call. On a transport error or rejected status the zero
// value of T and the error are returned.
func PostJSON[T any](c *Client, rawURL string, body any, cfg ...*RequestConfig) (T, error) {
	return decodeJSONResp[T](c.Post(rawURL, body, cfg...))
}

// PutJSON issues a PUT request with the given body and decodes a successful
// JSON response into a value of type T.
func PutJSON[T any](c *Client, rawURL string, body any, cfg ...*RequestConfig) (T, error) {
	return decodeJSONResp[T](c.Put(rawURL, body, cfg...))
}

// PatchJSON issues a PATCH request with the given body and decodes a successful
// JSON response into a value of type T.
func PatchJSON[T any](c *Client, rawURL string, body any, cfg ...*RequestConfig) (T, error) {
	return decodeJSONResp[T](c.Patch(rawURL, body, cfg...))
}

// DeleteJSON issues a DELETE request and decodes a successful JSON response
// into a value of type T.
func DeleteJSON[T any](c *Client, rawURL string, cfg ...*RequestConfig) (T, error) {
	return decodeJSONResp[T](c.Delete(rawURL, cfg...))
}

// RequestJSON issues an arbitrary request via Client.Request and decodes a
// successful JSON response into a value of type T. It is the generic,
// method-agnostic counterpart to the verb-specific *JSON helpers.
func RequestJSON[T any](c *Client, method, rawURL string, body any, rc *RequestConfig) (T, error) {
	return decodeJSONResp[T](c.Request(method, rawURL, body, rc))
}

// decodeJSONResp decodes resp into T, forwarding any request error and wrapping
// decode failures as an *Error with the ErrCodeBadResponse classification.
func decodeJSONResp[T any](resp *Response, err error) (T, error) {
	var out T
	if err != nil {
		return out, err
	}
	if err := resp.JSON(&out); err != nil {
		return out, &Error{Message: "decode json", Request: resp.Request, Response: resp, Err: err, Code: ErrCodeBadResponse}
	}
	return out, nil
}
