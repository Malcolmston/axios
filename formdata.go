package axios

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
)

// fdDefaultBoundary is the fixed multipart boundary used by NewFormData so that
// encoding is deterministic. Callers who need a random or specific boundary can
// override it with FormData.SetBoundary.
const fdDefaultBoundary = "axiosFormDataBoundary7MA4YWxkTrZu0gW"

// fdPart is a single multipart entry: a plain field when filename is empty, or
// a file part otherwise.
type fdPart struct {
	field    string
	filename string
	ctype    string
	data     []byte
}

// FormData builds a multipart/form-data request body, mirroring the browser
// FormData object that axios accepts as a request body. Add plain fields with
// AddField and file parts with AddFile / AddFileBytes, then obtain the encoded
// body with Reader, Bytes or WriteTo and its matching Content-Type with
// ContentType.
//
// Parts are emitted in the order they were added, and the default boundary is
// fixed, so a given sequence of calls always produces byte-identical output.
//
// Typical use with a client:
//
//	fd := axios.NewFormData()
//	fd.AddField("name", "Ada")
//	fd.AddFileBytes("avatar", "a.png", pngBytes)
//	body, _ := fd.Bytes()
//	client.Post("/upload", body, &axios.RequestConfig{ContentType: fd.ContentType()})
type FormData struct {
	boundary string
	parts    []fdPart
}

// NewFormData returns an empty FormData that uses a fixed default boundary.
func NewFormData() *FormData {
	return &FormData{boundary: fdDefaultBoundary}
}

// AddField appends a plain text field with the given name and value.
func (f *FormData) AddField(name, value string) {
	f.parts = append(f.parts, fdPart{field: name, data: []byte(value)})
}

// AddFile appends a file part read in full from r. The field is the form field
// name and filename is the reported client filename. The part is sent with
// Content-Type application/octet-stream. It returns any error encountered while
// reading r.
func (f *FormData) AddFile(field, filename string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.parts = append(f.parts, fdPart{field: field, filename: filename, data: data})
	return nil
}

// AddFileBytes appends a file part whose contents are data. It never fails and
// is a convenience wrapper over AddFile for in-memory payloads.
func (f *FormData) AddFileBytes(field, filename string, data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)
	f.parts = append(f.parts, fdPart{field: field, filename: filename, data: cp})
}

// AddFilePart appends a file part with an explicit Content-Type, overriding the
// default application/octet-stream. An empty contentType falls back to the
// default.
func (f *FormData) AddFilePart(field, filename, contentType string, data []byte) {
	cp := make([]byte, len(data))
	copy(cp, data)
	f.parts = append(f.parts, fdPart{field: field, filename: filename, ctype: contentType, data: cp})
}

// SetBoundary overrides the multipart boundary. The value must be a valid,
// non-empty multipart boundary (1-70 of the permitted characters); otherwise an
// error is returned and the boundary is left unchanged.
func (f *FormData) SetBoundary(boundary string) error {
	// Validate by attempting to set it on a throwaway writer.
	if err := multipart.NewWriter(io.Discard).SetBoundary(boundary); err != nil {
		return err
	}
	f.boundary = boundary
	return nil
}

// Boundary returns the multipart boundary that will be used when encoding.
func (f *FormData) Boundary() string {
	if f.boundary == "" {
		return fdDefaultBoundary
	}
	return f.boundary
}

// ContentType returns the full multipart/form-data Content-Type header value,
// including the boundary parameter, suitable for RequestConfig.ContentType.
func (f *FormData) ContentType() string {
	return "multipart/form-data; boundary=" + f.Boundary()
}

// Len returns the number of parts (fields and files) added so far.
func (f *FormData) Len() int {
	return len(f.parts)
}

// WriteTo encodes the multipart body into w, implementing io.WriterTo. It
// returns the number of bytes written and any error.
func (f *FormData) WriteTo(w io.Writer) (int64, error) {
	data, err := f.Bytes()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}

// Bytes encodes and returns the full multipart/form-data body. The result is
// deterministic for a given sequence of Add calls and boundary.
func (f *FormData) Bytes() ([]byte, error) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if f.boundary != "" {
		if err := mw.SetBoundary(f.boundary); err != nil {
			return nil, err
		}
	}
	for _, p := range f.parts {
		if p.filename == "" {
			pw, err := mw.CreateFormField(p.field)
			if err != nil {
				return nil, err
			}
			if _, err := pw.Write(p.data); err != nil {
				return nil, err
			}
			continue
		}
		hdr := make(textproto.MIMEHeader)
		hdr.Set("Content-Disposition",
			`form-data; name="`+fdEscapeQuotes(p.field)+`"; filename="`+fdEscapeQuotes(p.filename)+`"`)
		ct := p.ctype
		if ct == "" {
			ct = "application/octet-stream"
		}
		hdr.Set("Content-Type", ct)
		pw, err := mw.CreatePart(hdr)
		if err != nil {
			return nil, err
		}
		if _, err := pw.Write(p.data); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Reader encodes the body and returns it as an io.Reader positioned at the
// start, ready to be used as a request body.
func (f *FormData) Reader() (io.Reader, error) {
	data, err := f.Bytes()
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

// fdEscapeQuotes escapes backslashes and double quotes for use inside a
// Content-Disposition parameter value, matching the mime/multipart package.
func fdEscapeQuotes(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\', '"':
			b = append(b, '\\', s[i])
		default:
			b = append(b, s[i])
		}
	}
	return string(b)
}
