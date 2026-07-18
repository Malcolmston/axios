package axios

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

type g2Payload struct {
	Method string `json:"method"`
	Name   string `json:"name"`
}

func g2Server(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Name string `json:"name"`
		}
		if r.Body != nil {
			body, _ := io.ReadAll(r.Body)
			if len(body) > 0 {
				_ = json.Unmarshal(body, &in)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(g2Payload{Method: r.Method, Name: in.Name})
	}))
}

func TestTypedJSONHelpers(t *testing.T) {
	srv := g2Server(t)
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})

	got, err := PostJSON[g2Payload](c, "/", map[string]any{"name": "Ada"})
	if err != nil {
		t.Fatalf("PostJSON: %v", err)
	}
	if got.Method != http.MethodPost || got.Name != "Ada" {
		t.Fatalf("PostJSON = %+v", got)
	}

	got, err = PutJSON[g2Payload](c, "/", map[string]any{"name": "Bob"})
	if err != nil {
		t.Fatalf("PutJSON: %v", err)
	}
	if got.Method != http.MethodPut || got.Name != "Bob" {
		t.Fatalf("PutJSON = %+v", got)
	}

	got, err = PatchJSON[g2Payload](c, "/", map[string]any{"name": "Cy"})
	if err != nil {
		t.Fatalf("PatchJSON: %v", err)
	}
	if got.Method != http.MethodPatch {
		t.Fatalf("PatchJSON method = %s", got.Method)
	}

	got, err = DeleteJSON[g2Payload](c, "/")
	if err != nil {
		t.Fatalf("DeleteJSON: %v", err)
	}
	if got.Method != http.MethodDelete {
		t.Fatalf("DeleteJSON method = %s", got.Method)
	}

	got, err = RequestJSON[g2Payload](c, http.MethodPost, "/", map[string]any{"name": "Zed"}, nil)
	if err != nil {
		t.Fatalf("RequestJSON: %v", err)
	}
	if got.Method != http.MethodPost || got.Name != "Zed" {
		t.Fatalf("RequestJSON = %+v", got)
	}
}

func TestTypedJSONHelperError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := New(Config{BaseURL: srv.URL})

	if _, err := PostJSON[g2Payload](c, "/", nil); err == nil {
		t.Fatal("expected error for 500 response")
	}
}
