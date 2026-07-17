package axios

import "net/http"

// RequestInterceptor is invoked, in registration order, after the outgoing
// *http.Request has been fully built (URL, headers, auth and body applied) but
// before it is sent. It may mutate the request in place. Returning an error
// aborts the request and the error is propagated to the caller.
type RequestInterceptor func(req *http.Request) error

// ResponseInterceptor is invoked, in registration order, after a response has
// been received and buffered but before status validation. It may inspect or
// transform the Response in place. Returning an error aborts processing and the
// error is propagated to the caller.
type ResponseInterceptor func(resp *Response) error
