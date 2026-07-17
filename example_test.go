package axios_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

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
