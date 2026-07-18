package axios

import (
	"bytes"
	"mime"
	"mime/multipart"
	"strings"
	"testing"
)

func TestFormDataDeterministicEncode(t *testing.T) {
	fd := NewFormData()
	fd.AddField("name", "Ada")
	fd.AddField("age", "36")
	fd.AddFileBytes("avatar", "a.txt", []byte("hello"))

	b1, err := fd.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	b2, err := fd.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}
	if !bytes.Equal(b1, b2) {
		t.Fatal("encoding not deterministic")
	}

	want := "multipart/form-data; boundary=" + fdDefaultBoundary
	if got := fd.ContentType(); got != want {
		t.Fatalf("ContentType = %q, want %q", got, want)
	}
	if fd.Len() != 3 {
		t.Fatalf("Len = %d, want 3", fd.Len())
	}

	// The body must contain the fixed boundary and the field content.
	s := string(b1)
	if !strings.Contains(s, "--"+fdDefaultBoundary) {
		t.Fatal("boundary missing from body")
	}
	for _, sub := range []string{`name="name"`, "Ada", `name="age"`, "36", `filename="a.txt"`, "hello", "application/octet-stream"} {
		if !strings.Contains(s, sub) {
			t.Fatalf("body missing %q", sub)
		}
	}
}

func TestFormDataRoundTripParse(t *testing.T) {
	fd := NewFormData()
	fd.AddField("user", "Ada")
	fd.AddFileBytes("file", "data.bin", []byte("payload-bytes"))
	fd.AddFilePart("doc", "d.json", "application/json", []byte(`{"a":1}`))

	body, err := fd.Bytes()
	if err != nil {
		t.Fatalf("Bytes: %v", err)
	}

	_, params, err := mime.ParseMediaType(fd.ContentType())
	if err != nil {
		t.Fatalf("ParseMediaType: %v", err)
	}
	mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
	form, err := mr.ReadForm(1 << 20)
	if err != nil {
		t.Fatalf("ReadForm: %v", err)
	}
	if got := form.Value["user"]; len(got) != 1 || got[0] != "Ada" {
		t.Fatalf("user value = %v", got)
	}
	fh := form.File["file"]
	if len(fh) != 1 || fh[0].Filename != "data.bin" {
		t.Fatalf("file header = %v", fh)
	}
	if ct := form.File["doc"][0].Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("doc content-type = %q", ct)
	}
}

func TestFormDataSetBoundary(t *testing.T) {
	fd := NewFormData()
	if err := fd.SetBoundary("my-boundary-123"); err != nil {
		t.Fatalf("SetBoundary: %v", err)
	}
	if fd.Boundary() != "my-boundary-123" {
		t.Fatalf("Boundary = %q", fd.Boundary())
	}
	if err := fd.SetBoundary(""); err == nil {
		t.Fatal("expected error for empty boundary")
	}
	// Boundary unchanged after invalid set.
	if fd.Boundary() != "my-boundary-123" {
		t.Fatalf("Boundary changed after invalid set: %q", fd.Boundary())
	}
}

func TestFormDataReaderAndWriteTo(t *testing.T) {
	fd := NewFormData()
	fd.AddField("k", "v")

	want, _ := fd.Bytes()

	r, err := fd.Reader()
	if err != nil {
		t.Fatalf("Reader: %v", err)
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("ReadFrom: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatal("Reader output mismatch")
	}

	var wbuf bytes.Buffer
	n, err := fd.WriteTo(&wbuf)
	if err != nil {
		t.Fatalf("WriteTo: %v", err)
	}
	if int(n) != len(want) || !bytes.Equal(wbuf.Bytes(), want) {
		t.Fatalf("WriteTo output mismatch: n=%d", n)
	}
}

func TestFormDataAddFileReader(t *testing.T) {
	fd := NewFormData()
	if err := fd.AddFile("f", "r.txt", strings.NewReader("streamed")); err != nil {
		t.Fatalf("AddFile: %v", err)
	}
	body, _ := fd.Bytes()
	if !strings.Contains(string(body), "streamed") {
		t.Fatal("reader content missing")
	}
}

func BenchmarkFormDataBytes(b *testing.B) {
	fd := NewFormData()
	fd.AddField("name", "Ada")
	fd.AddFileBytes("avatar", "a.bin", bytes.Repeat([]byte("x"), 1024))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := fd.Bytes(); err != nil {
			b.Fatal(err)
		}
	}
}
