package axios

import "net/http"

// RequestTransform transforms an outgoing request body before it is sent. It
// receives the encoded body and the header set that will be applied to the
// request, and returns the (possibly rewritten) body. Transforms may mutate
// headers in place, for example to set Content-Type. They mirror the axios
// transformRequest pipeline and run in registration order after EncodeBody but
// before request interceptors. Returning an error aborts the request.
type RequestTransform func(body []byte, headers http.Header) ([]byte, error)

// ResponseTransform transforms a received response body after it has been
// buffered (and decompressed) but before response interceptors and status
// validation. It receives the body and response headers and returns the
// rewritten body. It mirrors the axios transformResponse pipeline and runs in
// registration order. Returning an error aborts processing. Response transforms
// are skipped for streaming responses (ResponseStream), which are never
// buffered.
type ResponseTransform func(body []byte, headers http.Header) ([]byte, error)

// applyRequestTransforms runs the transform pipeline over body, threading the
// output of each stage into the next.
func applyRequestTransforms(ts []RequestTransform, body []byte, headers http.Header) ([]byte, error) {
	for _, t := range ts {
		if t == nil {
			continue
		}
		out, err := t(body, headers)
		if err != nil {
			return nil, err
		}
		body = out
	}
	return body, nil
}

// applyResponseTransforms runs the transform pipeline over body.
func applyResponseTransforms(ts []ResponseTransform, body []byte, headers http.Header) ([]byte, error) {
	for _, t := range ts {
		if t == nil {
			continue
		}
		out, err := t(body, headers)
		if err != nil {
			return nil, err
		}
		body = out
	}
	return body, nil
}
