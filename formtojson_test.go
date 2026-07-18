package axios

import (
	"net/url"
	"reflect"
	"testing"
)

func TestFormToJSON(t *testing.T) {
	tests := []struct {
		name string
		in   url.Values
		want map[string]any
	}{
		{
			name: "flat",
			in:   url.Values{"a": {"1"}, "b": {"2"}},
			want: map[string]any{"a": "1", "b": "2"},
		},
		{
			name: "nested object",
			in:   url.Values{"user[name]": {"Ada"}, "user[age]": {"36"}},
			want: map[string]any{"user": map[string]any{"name": "Ada", "age": "36"}},
		},
		{
			name: "deep nested",
			in:   url.Values{"a[b][c]": {"x"}},
			want: map[string]any{"a": map[string]any{"b": map[string]any{"c": "x"}}},
		},
		{
			name: "append array",
			in:   url.Values{"items[]": {"a", "b", "c"}},
			want: map[string]any{"items": []any{"a", "b", "c"}},
		},
		{
			name: "indexed array",
			in:   url.Values{"n[0]": {"x"}, "n[1]": {"y"}},
			want: map[string]any{"n": []any{"x", "y"}},
		},
		{
			name: "repeated key becomes slice",
			in:   url.Values{"tag": {"go", "http"}},
			want: map[string]any{"tag": []any{"go", "http"}},
		},
		{
			name: "empty",
			in:   url.Values{},
			want: map[string]any{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := FormToJSON(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("FormToJSON(%v) = %#v, want %#v", tc.in, got, tc.want)
			}
		})
	}
}

func TestFormToJSONParseKey(t *testing.T) {
	tests := []struct {
		key  string
		want []string
	}{
		{"a", []string{"a"}},
		{"a[b]", []string{"a", "b"}},
		{"a[b][c]", []string{"a", "b", "c"}},
		{"a[]", []string{"a", ""}},
		{"a[0][b]", []string{"a", "0", "b"}},
	}
	for _, tc := range tests {
		got := ftjParseKey(tc.key)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("ftjParseKey(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}
