package axios

import (
	"net/http"
	"testing"
	"time"
)

func TestMergeConfig(t *testing.T) {
	base := Config{
		BaseURL: "https://base.example",
		Timeout: 5 * time.Second,
		Headers: http.Header{"X-Base": {"1"}, "X-Common": {"base"}},
	}
	override := Config{
		BaseURL: "https://override.example",
		Headers: http.Header{"X-Over": {"2"}, "X-Common": {"over"}},
	}
	got := MergeConfig(base, override)

	if got.BaseURL != "https://override.example" {
		t.Fatalf("BaseURL = %q", got.BaseURL)
	}
	if got.Timeout != 5*time.Second {
		t.Fatalf("Timeout = %v, want inherited from base", got.Timeout)
	}
	if got.Headers.Get("X-Base") != "1" {
		t.Fatalf("X-Base = %q, want inherited", got.Headers.Get("X-Base"))
	}
	if got.Headers.Get("X-Over") != "2" {
		t.Fatalf("X-Over = %q", got.Headers.Get("X-Over"))
	}
	if got.Headers.Get("X-Common") != "over" {
		t.Fatalf("X-Common = %q, want override to win", got.Headers.Get("X-Common"))
	}
}
