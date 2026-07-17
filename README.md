# axios

Ergonomic, axios-style HTTP client for Go, built entirely on the standard
library (`net/http`). No third-party dependencies.

Features:

- Client instance (`axios.New`) plus package-level `Get`/`Post`/... helpers, and
  `Create` for instances with deep-merged defaults
- Verb methods: `Get`, `Delete`, `Head`, `Options`, `Post`, `Put`, `Patch`
- Config: `BaseURL`, default headers (plus `HeaderGroups` common/get/post),
  query params, timeout, Basic/Bearer auth, proxy, custom
  `*http.Client`/transport, and a default context
- Automatic body encoding: JSON for structs/maps, form-urlencoded for
  `url.Values`, raw for `[]byte`/`string`/`io.Reader`
- Rich `Response`: `Status`, `StatusText`, `Headers`, raw `Body`, plus `JSON`,
  `Text`, `OK` helpers; configurable `ResponseType` including true streaming
- Cancellation: context, `AbortController`/`Signal`, and legacy `CancelToken`
  (`IsCancel`)
- Upload/download progress callbacks (`OnUploadProgress`/`OnDownloadProgress`)
- `TransformRequest`/`TransformResponse` pipelines alongside interceptors
- Query serialization: `ArrayFormat` (repeat/brackets/indices/comma), nested
  `ParamsMap`, and a `ParamsSerializer` hook; `GetUri` to preview a URL
- Transparent gzip/deflate decompression with `Accept-Encoding` negotiation
- Redirect control (`MaxRedirects`, `RedirectPolicy`) and body-size guards
  (`MaxContentLength`, `MaxBodyLength`)
- XSRF/CSRF token cookie-to-header (`XSRFCookieName`/`XSRFHeaderName`)
- Request and response interceptors, applied in order
- Configurable automatic retries (count, backoff, retry predicate)
- Typed `*Error` that carries the `Response` on non-2xx statuses
  (`ValidateStatus`, like axios `validateStatus`), with `IsAxiosError`,
  `Error.ToJSON`, and `code` classification
- Concurrency helpers `axios.All`/`axios.Spread`
- Generic `axios.GetJSON[T]` decode helper

## Install

```sh
go get github.com/malcolmston/axios
```

Requires Go 1.24+.

## Quick start

```go
package main

import (
	"fmt"
	"net/url"
	"time"

	"github.com/malcolmston/axios"
)

type User struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func main() {
	client := axios.New(axios.Config{
		BaseURL:     "https://api.example.com",
		Timeout:     5 * time.Second,
		BearerToken: "secret-token",
		Retry: &axios.RetryConfig{Retries: 3}, // retries 5xx/network errors
	})

	// GET with query params, decoded into a struct.
	resp, err := client.Get("/users/1", &axios.RequestConfig{
		Params: url.Values{"expand": {"profile"}},
	})
	if err != nil {
		panic(err)
	}
	var u User
	_ = resp.JSON(&u)
	fmt.Println(resp.Status, u.Name)

	// POST a JSON body (structs/maps are encoded automatically).
	_, _ = client.Post("/users", User{Name: "Ada", Age: 36})

	// Generic one-liner fetch + decode.
	got, _ := axios.GetJSON[User](client, "/users/2")
	fmt.Println(got.Name)
}
```

### Interceptors

```go
client := axios.New(axios.Config{
	RequestInterceptors: []axios.RequestInterceptor{
		func(req *http.Request) error {
			req.Header.Set("X-Trace", "abc")
			return nil
		},
	},
	ResponseInterceptors: []axios.ResponseInterceptor{
		func(resp *axios.Response) error {
			log.Printf("got %d", resp.Status)
			return nil
		},
	},
})
```

### Error handling

Non-2xx responses return an `*axios.Error` whose `Response` field is still
populated, so you can inspect the body:

```go
resp, err := client.Get("/missing")
var aerr *axios.Error
if errors.As(err, &aerr) {
	fmt.Println(aerr.StatusCode(), aerr.Response.Text())
}
```

Override which statuses are considered successful with `ValidateStatus` on the
client `Config` or a per-request `RequestConfig`. Use `axios.IsAxiosError` and
`aerr.ToJSON()` for axios-style handling, and branch on `aerr.Code` (e.g.
`axios.ErrCodeCanceled`).

### Cancellation

```go
ctrl := axios.NewAbortController()
go func() { time.Sleep(time.Second); ctrl.Abort(nil) }()

_, err := client.Get("/slow", &axios.RequestConfig{Signal: ctrl.Signal()})
if axios.IsCancel(err) {
	fmt.Println("canceled")
}
```

### Progress

```go
_, _ = client.Post("/upload", bigPayload, &axios.RequestConfig{
	OnUploadProgress:   func(e axios.ProgressEvent) { fmt.Printf("up %.0f%%\n", e.Progress()*100) },
	OnDownloadProgress: func(e axios.ProgressEvent) { fmt.Printf("down %d/%d\n", e.Loaded, e.Total) },
})
```

### Streaming responses

```go
st := axios.ResponseStream
resp, _ := client.Get("/large", &axios.RequestConfig{ResponseType: &st})
defer resp.Close()
_, _ = io.Copy(os.Stdout, resp.Stream)
```

### Query serialization

```go
brackets := axios.ArrayFormatBrackets
_, _ = client.Get("/search", &axios.RequestConfig{
	Params:      url.Values{"id": {"1", "2"}}, // id[]=1&id[]=2
	ArrayFormat: &brackets,
})
```

### Concurrent requests

```go
resps, err := axios.All(
	func() (*axios.Response, error) { return client.Get("/a") },
	func() (*axios.Response, error) { return client.Get("/b") },
)
_ = err
combined := axios.Spread(func(rs ...*axios.Response) string {
	return rs[0].Text() + rs[1].Text()
})(resps)
```

### Instances

```go
api := client.Create(axios.Config{
	Headers: http.Header{"X-Env": {"prod"}}, // deep-merged over client defaults
})
```

## License

See repository.
