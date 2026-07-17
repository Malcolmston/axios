package axios

import (
	"context"
	"errors"
)

// ErrCanceled is the cause reported when a request is aborted via an
// AbortController or CancelToken without an explicit reason.
var ErrCanceled = errors.New("axios: request canceled")

// AbortSignal is the read side of an AbortController. A request configured with
// a signal (RequestConfig.Signal or Config.Signal) is canceled when the
// controlling AbortController's Abort method is called. It is the standard-
// library analogue of the DOM AbortSignal that axios accepts.
type AbortSignal struct {
	ctx context.Context
}

// Context returns the context that is canceled when the signal is aborted. It
// is never nil.
func (s *AbortSignal) Context() context.Context {
	if s == nil || s.ctx == nil {
		return context.Background()
	}
	return s.ctx
}

// Aborted reports whether the signal has been aborted.
func (s *AbortSignal) Aborted() bool {
	if s == nil || s.ctx == nil {
		return false
	}
	return s.ctx.Err() != nil
}

// Err returns the cause the signal was aborted with, or nil if it has not been
// aborted.
func (s *AbortSignal) Err() error {
	if s == nil || s.ctx == nil {
		return nil
	}
	if s.ctx.Err() == nil {
		return nil
	}
	return context.Cause(s.ctx)
}

// AbortController creates AbortSignals and aborts them, mirroring the DOM
// AbortController API. Share one controller's Signal across several requests to
// cancel them all at once.
type AbortController struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
}

// NewAbortController returns a ready-to-use AbortController derived from
// context.Background.
func NewAbortController() *AbortController {
	return NewAbortControllerWithContext(context.Background())
}

// NewAbortControllerWithContext returns an AbortController whose signal is also
// canceled when the parent context is canceled. If parent is nil,
// context.Background is used.
func NewAbortControllerWithContext(parent context.Context) *AbortController {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithCancelCause(parent)
	return &AbortController{ctx: ctx, cancel: cancel}
}

// Signal returns the AbortSignal controlled by this controller. Pass it to
// RequestConfig.Signal or Config.Signal.
func (a *AbortController) Signal() *AbortSignal {
	return &AbortSignal{ctx: a.ctx}
}

// Abort cancels every request using this controller's signal. The optional
// cause is reported via AbortSignal.Err and context.Cause; when nil,
// ErrCanceled is used.
func (a *AbortController) Abort(cause error) {
	if cause == nil {
		cause = ErrCanceled
	}
	a.cancel(cause)
}

// CancelToken is the legacy axios cancellation primitive. Obtain one, together
// with its cancel function, from NewCancelToken, and pass it via
// RequestConfig.CancelToken. New code should prefer AbortController.
type CancelToken struct {
	signal *AbortSignal
}

// Context returns the context canceled when the token is canceled.
func (t *CancelToken) Context() context.Context {
	if t == nil {
		return context.Background()
	}
	return t.signal.Context()
}

// CancelFunc cancels a CancelToken with an optional human-readable message.
type CancelFunc func(message string)

// NewCancelToken returns a CancelToken and the function that cancels it,
// mirroring axios.CancelToken.source(). Calling the returned CancelFunc aborts
// any in-flight request configured with the token; the message becomes the
// cancellation cause.
func NewCancelToken() (*CancelToken, CancelFunc) {
	ctrl := NewAbortController()
	tok := &CancelToken{signal: ctrl.Signal()}
	return tok, func(message string) {
		if message == "" {
			ctrl.Abort(ErrCanceled)
			return
		}
		ctrl.Abort(&CanceledError{Message: message})
	}
}

// CanceledError is the cause carried when a CancelToken is canceled with an
// explicit message.
type CanceledError struct {
	// Message is the reason supplied to the cancel function.
	Message string
}

// Error implements the error interface.
func (e *CanceledError) Error() string {
	if e.Message == "" {
		return ErrCanceled.Error()
	}
	return "axios: canceled: " + e.Message
}

// IsCancel reports whether err was produced by aborting a request through an
// AbortController or CancelToken. It mirrors axios.isCancel.
func IsCancel(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrCanceled) || errors.Is(err, context.Canceled) {
		return true
	}
	var ce *CanceledError
	return errors.As(err, &ce)
}
