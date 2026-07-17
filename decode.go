package axios

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"strings"
)

// ResponseType selects how a response body is delivered, mirroring the axios
// responseType option. The default (ResponseDefault) buffers the body into
// Response.Body, exactly like the JSON/text/bytes types; the distinction only
// matters for ResponseStream, which leaves the body unread for the caller to
// consume from Response.Stream.
type ResponseType int

const (
	// ResponseDefault buffers the body and leaves interpretation to the caller
	// (Response.JSON, Response.Text, Response.Bytes).
	ResponseDefault ResponseType = iota
	// ResponseJSON buffers the body; callers typically decode it with
	// Response.JSON. Included for parity with axios responseType: "json".
	ResponseJSON
	// ResponseText buffers the body as text; use Response.Text.
	ResponseText
	// ResponseBytes buffers the raw bytes; use Response.Bytes.
	ResponseBytes
	// ResponseStream leaves the response body unbuffered and exposes it via
	// Response.Stream for incremental reading. The caller MUST close the
	// response (Response.Close) to release the connection.
	ResponseStream
)

// decompress wraps r with a decompressing reader selected by the
// Content-Encoding value (gzip, deflate/zlib, or identity). For "deflate" it
// tries zlib (RFC 1950) first and falls back to raw flate (RFC 1951), matching
// the leniency of browsers and axios. Unknown encodings are returned
// unchanged. The returned closer, when non-nil, must be closed by the caller in
// addition to the underlying body. Brotli ("br") is not supported because the
// standard library has no brotli implementation.
func decompress(encoding string, r io.Reader) (io.Reader, io.Closer, error) {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "gzip", "x-gzip":
		zr, err := gzip.NewReader(r)
		if err != nil {
			return nil, nil, err
		}
		return zr, zr, nil
	case "deflate":
		// Peek so we can retry as raw flate if the stream is not zlib-framed.
		buf, err := io.ReadAll(r)
		if err != nil {
			return nil, nil, err
		}
		if zr, zerr := zlib.NewReader(bytes.NewReader(buf)); zerr == nil {
			return zr, zr, nil
		}
		fr := flate.NewReader(bytes.NewReader(buf))
		return fr, fr, nil
	default:
		return r, nil, nil
	}
}
