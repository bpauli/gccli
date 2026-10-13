package garminapi

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// These tests drive the production client built by newHTTPClient against
// servers that stop moving bytes at one point of the exchange. Each stalled
// call must fail within a short deadline; a hang fails the test through the
// deadline in callWithin.

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
	orig := responseHeaderTimeout
	t.Cleanup(func() { responseHeaderTimeout = orig })
	responseHeaderTimeout = testStallTimeout

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

// baseTransport returns the http.Transport under the client's retry layer.
func baseTransport(c *Client) *http.Transport {
	return c.httpClient.Transport.(*RetryTransport).Base.(*http.Transport)
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
	if err == nil {
		t.Fatalf("err = nil after %s, want an error", elapsed.Round(time.Millisecond))
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
