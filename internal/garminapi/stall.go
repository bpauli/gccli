package garminapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"
)

type StallError struct {
	Idle time.Duration
}

func (e *StallError) Error() string {
	return fmt.Sprintf("connection stalled: no data sent or received for %s", e.Idle)
}

type stallTransport struct {
	base http.RoundTripper
	idle time.Duration
}

func (t *stallTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	w := watchStall(req.Context(), t.idle)
	req = req.WithContext(w.ctx)
	if req.Body != nil && req.Body != http.NoBody {
		req.Body = &progressBody{ReadCloser: req.Body, w: w}
		if getBody := req.GetBody; getBody != nil {
			// The transport rewinds with GetBody when a reused connection
			// fails, so the new copy must mark progress too.
			req.GetBody = func() (io.ReadCloser, error) {
				body, err := getBody()
				if err != nil {
					return nil, err
				}
				return &progressBody{ReadCloser: body, w: w}, nil
			}
		}
	}
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		w.end()
		return nil, w.explain(err)
	}
	w.touch()
	resp.Body = &progressBody{ReadCloser: resp.Body, w: w, endsWatch: true}
	return resp, nil
}

type stallWatch struct {
	ctx    context.Context
	cancel context.CancelCauseFunc
	idle   time.Duration
	last   atomic.Int64
}

func watchStall(parent context.Context, idle time.Duration) *stallWatch {
	ctx, cancel := context.WithCancelCause(parent)
	w := &stallWatch{ctx: ctx, cancel: cancel, idle: idle}
	w.touch()
	go w.run()
	return w
}

func (w *stallWatch) touch() { w.last.Store(time.Now().UnixNano()) }

func (w *stallWatch) run() {
	timer := time.NewTimer(w.idle)
	defer timer.Stop()
	for {
		select {
		case <-w.ctx.Done():
			return
		case <-timer.C:
			quiet := time.Since(time.Unix(0, w.last.Load()))
			if quiet >= w.idle {
				w.cancel(&StallError{Idle: w.idle})
				return
			}
			timer.Reset(w.idle - quiet)
		}
	}
}

func (w *stallWatch) end() { w.cancel(nil) }

func (w *stallWatch) explain(err error) error {
	var stall *StallError
	if errors.As(context.Cause(w.ctx), &stall) {
		return stall
	}
	return err
}

type progressBody struct {
	io.ReadCloser
	w         *stallWatch
	endsWatch bool
}

func (b *progressBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.w.touch()
	}
	if err != nil {
		if b.endsWatch {
			b.w.end()
		}
		if err != io.EOF {
			err = b.w.explain(err)
		}
	}
	return n, err
}

func (b *progressBody) Close() error {
	err := b.ReadCloser.Close()
	if b.endsWatch {
		b.w.end()
	}
	return err
}
