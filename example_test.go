package axios_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/malcolmston/axios"
)

// Example demonstrates a basic GET request against a test server, decoding the
// JSON response into a struct.
func Example() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"name":"Ada Lovelace","age":36}`)
	}))
	defer srv.Close()

	client := axios.New(axios.Config{BaseURL: srv.URL})

	var person struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	resp, err := client.Get("/user")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	if err := resp.JSON(&person); err != nil {
		fmt.Println("decode:", err)
		return
	}

	fmt.Println("status:", resp.Status)
	fmt.Println("name:", person.Name)

	// Output:
	// status: 200
	// name: Ada Lovelace
}

// ExampleSerializeParams shows the array serialization formats used by the
// query-parameter encoder (and the paramsSerializer hook).
func ExampleSerializeParams() {
	params := url.Values{"id": {"1", "2"}}
	fmt.Println(axios.SerializeParams(params, axios.ArrayFormatRepeat))
	fmt.Println(axios.SerializeParams(params, axios.ArrayFormatBrackets))
	fmt.Println(axios.SerializeParams(params, axios.ArrayFormatComma))
	// Output:
	// id=1&id=2
	// id%5B%5D=1&id%5B%5D=2
	// id=1%2C2
}

// ExampleAll runs several requests concurrently and combines their results with
// Spread, mirroring axios.all/axios.spread.
func ExampleAll() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.Path)
	}))
	defer srv.Close()
	client := axios.New(axios.Config{BaseURL: srv.URL})

	resps, err := axios.All(
		func() (*axios.Response, error) { return client.Get("/first") },
		func() (*axios.Response, error) { return client.Get("/second") },
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(axios.Spread(func(rs ...*axios.Response) string {
		return rs[0].Text() + " " + rs[1].Text()
	})(resps))
	// Output:
	// /first /second
}
