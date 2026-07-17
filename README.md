# axios

Ergonomic, axios-style HTTP client for Go, built entirely on the standard
library (`net/http`). No third-party dependencies.

Features:

- Client instance (`axios.New`) plus package-level `Get`/`Post`/... helpers
- Verb methods: `Get`, `Delete`, `Head`, `Options`, `Post`, `Put`, `Patch`
- Config: `BaseURL`, default headers, query params, timeout, Basic/Bearer auth,
  custom `*http.Client`/transport, and a default context
- Automatic body encoding: JSON for structs/maps, form-urlencoded for
  `url.Values`, raw for `[]byte`/`string`/`io.Reader`
- Rich `Response`: `Status`, `StatusText`, `Headers`, raw `Body`, plus `JSON`,
  `Text`, `OK` helpers
- Request and response interceptors, applied in order
- Configurable automatic retries (count, backoff, retry predicate)
- Typed `*Error` that carries the `Response` on non-2xx statuses
  (`ValidateStatus`, like axios `validateStatus`)
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
client `Config` or a per-request `RequestConfig`.

## License

See repository.
