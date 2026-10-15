package conformance

import (
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestProxyConnectAfterClose(t *testing.T) {
	for range 20 {
		if connectAfterClose(t) {
			return
		}
	}
	t.Fatal("the proxy closed before it accepted the CONNECT, 20 times")
}

func connectAfterClose(t *testing.T) (accepted bool) {
	p, err := ca.serve(http.NotFoundHandler())
	if err != nil {
		t.Fatal(err)
	}
	c, err := net.Dial("tcp", strings.TrimPrefix(p.URL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	if _, err := io.WriteString(c, "CONNECT connectapi.garmin.com:443 HTTP/1.1\r\nHost: connectapi.garmin.com:443\r\n"); err != nil {
		t.Fatal(err)
	}
	p.close()
	_, _ = io.WriteString(c, "\r\n")
	if err := c.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	resp, err := io.ReadAll(c)
	if !strings.HasPrefix(string(resp), "HTTP/1.1 200 ") {
		return false
	}
	if err != nil {
		t.Errorf("a CONNECT that ends after close got no closed connection: %v", err)
	}
	return true
}
