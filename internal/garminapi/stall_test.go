package garminapi

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type protoMode string

const (
	modeH1      protoMode = "h1"
	modeH2      protoMode = "h2"
	modeH2Proxy protoMode = "h2-via-https-proxy"
)

var bothProtos = []protoMode{modeH1, modeH2}

const testStallTimeout = 300 * time.Millisecond

func stallServer(t *testing.T, mode protoMode, h http.HandlerFunc) (*Client, *atomic.Int32) {
	t.Helper()
	orig := stallTimeout
	t.Cleanup(func() { stallTimeout = orig })
	stallTimeout = testStallTimeout

	stop := make(chan struct{})
	var stopOnce sync.Once
	var conns atomic.Int32
	wrapped := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if mode != modeH1 && r.ProtoMajor != 2 {
			t.Errorf("server saw %s, want HTTP/2", r.Proto)
		}
		h(w, r)
	})
	srv := httptest.NewUnstartedServer(wrapped)
	srv.Config.BaseContext = func(net.Listener) context.Context {
		return context.WithValue(context.Background(), stopKey{}, stop)
	}
	srv.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Listener = smallBufListener{srv.Listener}
	if mode == modeH1 {
		srv.Start()
	} else {
		srv.EnableHTTP2 = true
		srv.StartTLS()
	}
	t.Cleanup(srv.Close)
	t.Cleanup(func() { stopOnce.Do(func() { close(stop) }) })

	c := NewClient(testTokens(), WithBaseURL(srv.URL))
	base := baseTransport(c)
	if mode != modeH1 {
		base.TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	}
	base.DialContext = (&smallBufDialer{net.Dialer{Timeout: 30 * time.Second, KeepAlive: 30 * time.Second}}).DialContext
	if mode == modeH2Proxy {
		proxy := httptest.NewTLSServer(http.HandlerFunc(connectProxy))
		t.Cleanup(proxy.Close)
		u, _ := url.Parse(proxy.URL)
		base.Proxy = http.ProxyURL(u)
	}
	return c, &conns
}

func stallLayer(c *Client) *stallTransport {
	return c.httpClient.Transport.(*RetryTransport).Base.(*stallTransport)
}

func baseTransport(c *Client) *http.Transport {
	return stallLayer(c).base.(*http.Transport)
}

const (
	serverReadBuffer  = 32 << 10
	clientWriteBuffer = 64 << 10
)

type smallBufListener struct{ net.Listener }

func (l smallBufListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.SetReadBuffer(serverReadBuffer)
	}
	return c, err
}

type smallBufDialer struct{ net.Dialer }

func (d *smallBufDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	c, err := d.Dialer.DialContext(ctx, network, addr)
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.SetWriteBuffer(clientWriteBuffer)
	}
	return c, err
}

type stopKey struct{}

func block(r *http.Request) {
	select {
	case <-r.Context().Done():
	case <-r.Context().Value(stopKey{}).(chan struct{}):
	}
}

func connectProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodConnect {
		http.Error(w, "CONNECT only", http.StatusMethodNotAllowed)
		return
	}
	upstream, err := net.Dial("tcp", r.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	client, buf, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return
	}
	_, _ = client.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	go func() { _, _ = io.Copy(upstream, buf); _ = upstream.Close() }()
	_, _ = io.Copy(client, upstream)
	_ = client.Close()
}

func callWithin(t *testing.T, limit time.Duration, call func() error) (time.Duration, error) {
	t.Helper()
	errc := make(chan error, 1)
	start := time.Now()
	go func() { errc <- call() }()
	select {
	case err := <-errc:
		return time.Since(start), err
	case <-time.After(limit):
		t.Fatalf("still blocked after %s", limit)
		return 0, nil
	}
}

func wantStall(t *testing.T, err error, elapsed time.Duration) {
	t.Helper()
	var stall *StallError
	if !errors.As(err, &stall) {
		t.Fatalf("err = %v (%T) after %s, want *StallError", err, err, elapsed.Round(time.Millisecond))
	}
	if stall.Idle != testStallTimeout {
		t.Fatalf("Idle = %s, want %s", stall.Idle, testStallTimeout)
	}
	t.Logf("failed after %s: %v", elapsed.Round(time.Millisecond), err)
}

func partialBodyThenBlock(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Length", "1000")
	_, _ = w.Write([]byte(`{"partial":`))
	w.(http.Flusher).Flush()
	block(r)
}

func TestStall_ResponseBody(t *testing.T) {
	for _, mode := range []protoMode{modeH1, modeH2, modeH2Proxy} {
		t.Run(string(mode), func(t *testing.T) {
			c, _ := stallServer(t, mode, partialBodyThenBlock)
			d, err := callWithin(t, 3*time.Second, func() error {
				_, err := c.ConnectAPI(context.Background(), http.MethodGet, "/x", nil)
				return err
			})
			wantStall(t, err, d)
		})
	}
}

func TestStall_Download(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			c, _ := stallServer(t, mode, partialBodyThenBlock)
			d, err := callWithin(t, 3*time.Second, func() error {
				_, err := c.Download(context.Background(), "/x")
				return err
			})
			wantStall(t, err, d)
		})
	}
}

func TestStall_RequestBodyWrite(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			c, _ := stallServer(t, mode, func(_ http.ResponseWriter, r *http.Request) { block(r) })
			d, err := callWithin(t, 5*time.Second, func() error {
				_, err := c.ConnectAPI(context.Background(), http.MethodPost, "/x", bytes.NewReader(make([]byte, 8<<20)))
				return err
			})
			wantStall(t, err, d)
		})
	}
}

func TestStall_HeaderWait(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			c, _ := stallServer(t, mode, func(_ http.ResponseWriter, r *http.Request) { block(r) })
			d, err := callWithin(t, 3*time.Second, func() error {
				_, err := c.ConnectAPI(context.Background(), http.MethodGet, "/x", nil)
				return err
			})
			wantStall(t, err, d)
		})
	}
}

// Over HTTP/2 the response body ends with END_STREAM, not at Content-Length.
func TestStall_H2StreamNeverEnds(t *testing.T) {
	c, _ := stallServer(t, modeH2, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "6")
		_, _ = w.Write([]byte("xxxxxx"))
		w.(http.Flusher).Flush()
		block(r)
	})
	d, err := callWithin(t, 3*time.Second, func() error {
		_, err := c.ConnectAPI(context.Background(), http.MethodGet, "/x", nil)
		return err
	})
	wantStall(t, err, d)
}

func TestStall_DrainBeforeRetryThenRetrySucceeds(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			var calls atomic.Int32
			c, _ := stallServer(t, mode, func(w http.ResponseWriter, r *http.Request) {
				if calls.Add(1) == 1 {
					w.Header().Set("Content-Length", "1000")
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte("busy"))
					w.(http.Flusher).Flush()
					block(r)
					return
				}
				_, _ = io.WriteString(w, `{"ok":true}`)
			})
			d, err := callWithin(t, 5*time.Second, func() error {
				_, err := c.ConnectAPI(context.Background(), http.MethodGet, "/x", nil)
				return err
			})
			if err != nil {
				t.Fatalf("err = %v, want success on the retry", err)
			}
			if n := calls.Load(); n != 2 {
				t.Fatalf("calls = %d, want 2", n)
			}
			t.Logf("drain stalled, retry succeeded after %s", d.Round(time.Millisecond))
		})
	}
}

func dialOutsideStallWatch(t *testing.T, c *Client, path string) {
	t.Helper()
	stallLayer(c).idle = time.Minute
	if _, err := c.ConnectAPI(context.Background(), http.MethodGet, path, nil); err != nil {
		t.Fatalf("warm-up request: %v", err)
	}
	stallLayer(c).idle = testStallTimeout
}

func TestStall_HeadersThenSlowBodySucceeds(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			const overHalfLimit = testStallTimeout/2 + testStallTimeout/20
			c, conns := stallServer(t, mode, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/warm" {
					_, _ = io.WriteString(w, `{"ok":true}`)
					return
				}
				time.Sleep(overHalfLimit)
				w.WriteHeader(http.StatusOK)
				w.(http.Flusher).Flush()
				time.Sleep(overHalfLimit)
				_, _ = io.WriteString(w, `{"ok":true}`)
			})
			dialOutsideStallWatch(t, c, "/warm")
			var data []byte
			d, err := callWithin(t, 3*time.Second, func() error {
				var err error
				data, err = c.ConnectAPI(context.Background(), http.MethodGet, "/x", nil)
				return err
			})
			if err != nil {
				t.Fatalf("err = %v after %s, want success", err, d.Round(time.Millisecond))
			}
			if string(data) != `{"ok":true}` {
				t.Fatalf("data = %q, want the full body", data)
			}
			if n := conns.Load(); n != 1 {
				t.Fatalf("server conns = %d, want the timed request on the warm connection", n)
			}
		})
	}
}

func TestStall_SlowDownloadSucceeds(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			const chunks = 10
			c, _ := stallServer(t, mode, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "10")
				for range chunks {
					time.Sleep(100 * time.Millisecond)
					_, _ = w.Write([]byte("x"))
					w.(http.Flusher).Flush()
				}
			})
			var data []byte
			d, err := callWithin(t, 5*time.Second, func() error {
				var err error
				data, err = c.Download(context.Background(), "/x")
				return err
			})
			if err != nil {
				t.Fatalf("slow download failed: %v", err)
			}
			if string(data) != "xxxxxxxxxx" {
				t.Fatalf("data = %q, want 10 bytes", data)
			}
			if d < 3*testStallTimeout {
				t.Fatalf("download took %s, want longer than %s to prove the limit is not total time", d, 3*testStallTimeout)
			}
		})
	}
}

func TestStall_SlowUploadSucceeds(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			const (
				size            = 8 << 20
				readStep        = 64 << 10
				h2StreamWindow  = 1 << 20
				idleTail        = h2StreamWindow + clientWriteBuffer + serverReadBuffer
				tailDrainBudget = testStallTimeout / 2
				pausePerStep    = tailDrainBudget * readStep / idleTail
			)
			var total atomic.Int64
			c, _ := stallServer(t, mode, func(w http.ResponseWriter, r *http.Request) {
				buf := make([]byte, readStep)
				sinceSleep := 0
				for {
					n, err := r.Body.Read(buf)
					total.Add(int64(n))
					sinceSleep += n
					for sinceSleep >= readStep {
						time.Sleep(pausePerStep)
						sinceSleep -= readStep
					}
					if err != nil {
						break
					}
				}
				_, _ = io.WriteString(w, `{"read":true}`)
			})
			d, err := callWithin(t, 10*time.Second, func() error {
				_, err := c.ConnectAPI(context.Background(), http.MethodPost, "/x", bytes.NewReader(make([]byte, size)))
				return err
			})
			if err != nil {
				t.Fatalf("slow upload failed: %v", err)
			}
			if got := total.Load(); got != size {
				t.Fatalf("server read %d bytes, want %d", got, size)
			}
			if d < 3*testStallTimeout {
				t.Fatalf("upload took %s, want longer than %s", d, 3*testStallTimeout)
			}
		})
	}
}

func TestStall_RetryAfterWaitNotIdle(t *testing.T) {
	var calls atomic.Int32
	c, _ := stallServer(t, modeH1, func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = io.WriteString(w, `{"ok":true}`)
	})
	d, err := callWithin(t, 5*time.Second, func() error {
		_, err := c.ConnectAPI(context.Background(), http.MethodGet, "/x", nil)
		return err
	})
	if err != nil {
		t.Fatalf("err = %v, want success after the Retry-After wait", err)
	}
	if n := calls.Load(); n != 2 {
		t.Fatalf("calls = %d, want 2", n)
	}
	if d < time.Second {
		t.Fatalf("succeeded after %s, want the 1 s Retry-After wait", d)
	}
}

func TestStall_CancelDuringStall(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			c, _ := stallServer(t, mode, partialBodyThenBlock)
			stallLayer(c).idle = time.Minute
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(100*time.Millisecond, cancel)
			_, err := callWithin(t, 3*time.Second, func() error {
				_, err := c.ConnectAPI(ctx, http.MethodGet, "/x", nil)
				return err
			})
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("err = %v, want context.Canceled", err)
			}
			var stall *StallError
			if errors.As(err, &stall) {
				t.Fatalf("err = %v, want no StallError", err)
			}
		})
	}
}

func TestStall_ConnectionReuse(t *testing.T) {
	for _, mode := range bothProtos {
		t.Run(string(mode), func(t *testing.T) {
			c, conns := stallServer(t, mode, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = io.WriteString(w, `{"ok":true}`)
			})
			var reused []bool
			get := func(step string) {
				ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
					GotConn: func(info httptrace.GotConnInfo) { reused = append(reused, info.Reused) },
				})
				if _, err := c.ConnectAPI(ctx, http.MethodGet, "/x", nil); err != nil {
					t.Fatalf("%s: %v", step, err)
				}
			}
			get("first request")
			get("request after the watch ended at EOF")
			time.Sleep(2 * testStallTimeout)
			get("request after the pooled connection idled past the limit")
			if len(reused) != 3 || reused[0] || !reused[1] || !reused[2] || conns.Load() != 1 {
				t.Fatalf("reused = %v, server conns = %d, want [false true true] on one connection", reused, conns.Load())
			}
		})
	}
}
