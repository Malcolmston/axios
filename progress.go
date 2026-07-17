package axios

import "io"

// ProgressEvent describes the progress of a request or response body transfer.
// It mirrors the axios progress event passed to onUploadProgress and
// onDownloadProgress.
type ProgressEvent struct {
	// Loaded is the number of bytes transferred so far.
	Loaded int64
	// Total is the total number of bytes expected, or -1 when the length is
	// unknown (for example a chunked response with no Content-Length).
	Total int64
	// LengthComputable reports whether Total is known (Total >= 0).
	LengthComputable bool
}

// Progress returns the fraction transferred in the range [0,1], or 0 when the
// total length is unknown.
func (e ProgressEvent) Progress() float64 {
	if !e.LengthComputable || e.Total <= 0 {
		return 0
	}
	p := float64(e.Loaded) / float64(e.Total)
	if p > 1 {
		return 1
	}
	return p
}

// ProgressFunc is the callback invoked as bytes are transferred. It is called
// with cumulative progress; the final call reports Loaded == Total when the
// total is known.
type ProgressFunc func(ProgressEvent)

// progressReader wraps an io.Reader and reports cumulative bytes read to a
// ProgressFunc. It is used to drive OnUploadProgress (wrapping the request
// body) and OnDownloadProgress (wrapping the response body).
type progressReader struct {
	r      io.Reader
	total  int64
	loaded int64
	fn     ProgressFunc
}

// newProgressReader wraps r so that every read reports progress to fn. total is
// the expected number of bytes, or -1 when unknown. If fn is nil the original
// reader is returned unchanged.
func newProgressReader(r io.Reader, total int64, fn ProgressFunc) io.Reader {
	if fn == nil {
		return r
	}
	return &progressReader{r: r, total: total, fn: fn}
}

// Read implements io.Reader, forwarding to the wrapped reader and emitting a
// ProgressEvent after each read (including the terminal read that returns EOF).
func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.loaded += int64(n)
		p.fn(ProgressEvent{
			Loaded:           p.loaded,
			Total:            p.total,
			LengthComputable: p.total >= 0,
		})
	}
	return n, err
}
