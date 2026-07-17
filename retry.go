package axios

import "time"

// RetryConfig controls automatic retries. A request is attempted at most
// Retries+1 times. Between attempts the client sleeps for the duration returned
// by Backoff, and only retries when RetryOn reports true.
type RetryConfig struct {
	// Retries is the number of additional attempts after the first. A value of
	// 0 disables retrying.
	Retries int
	// Backoff returns how long to wait before the given attempt (1-based: the
	// argument is the number of the attempt about to be made, i.e. 1 means the
	// first retry). If nil, DefaultBackoff is used.
	Backoff func(attempt int) time.Duration
	// RetryOn decides whether an attempt should be retried given its result.
	// Either resp or err will be set. If nil, DefaultRetryOn is used.
	RetryOn func(resp *Response, err error) bool
}

// DefaultBackoff returns an exponential backoff: 100ms, 200ms, 400ms, ...
// capped at 10s.
func DefaultBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := 100 * time.Millisecond
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= 10*time.Second {
			return 10 * time.Second
		}
	}
	return d
}

// DefaultRetryOn retries on any transport error or on a 5xx status code.
func DefaultRetryOn(resp *Response, err error) bool {
	if err != nil {
		return true
	}
	if resp != nil && resp.Status >= 500 && resp.Status <= 599 {
		return true
	}
	return false
}

func (rc *RetryConfig) backoff(attempt int) time.Duration {
	if rc.Backoff != nil {
		return rc.Backoff(attempt)
	}
	return DefaultBackoff(attempt)
}

func (rc *RetryConfig) retryOn(resp *Response, err error) bool {
	if rc.RetryOn != nil {
		return rc.RetryOn(resp, err)
	}
	return DefaultRetryOn(resp, err)
}
