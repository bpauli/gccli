package conformance

import (
	"bufio"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type request struct {
	Method string
	Host   string
	URI    string
	Bearer string
}

func (r request) String() string {
	return r.Method + " " + r.Host + r.URI
}

type authority struct {
	PEM  []byte
	cert tls.Certificate
}

func newAuthority() (*authority, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "gccli conformance mock"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		DNSNames:     []string{"*.garmin.com", "garmin.com", "*.garmin.cn", "garmin.cn"},
		IsCA:         true, BasicConstraintsValid: true,
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}
	kb, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, err
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	cert, err := tls.X509KeyPair(certPEM, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb}))
	if err != nil {
		return nil, err
	}
	return &authority{PEM: certPEM, cert: cert}, nil
}

type proxy struct {
	URL     string
	handler http.Handler
	ln      net.Listener
	tunnels chan net.Conn
	server  *http.Server

	mu   sync.Mutex
	reqs []request
}

func (a *authority) serve(handler http.Handler) (*proxy, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	p := &proxy{
		URL:     "http://" + ln.Addr().String(),
		handler: handler,
		ln:      ln,
		tunnels: make(chan net.Conn),
	}
	p.server = &http.Server{
		Handler:  http.HandlerFunc(p.record),
		ErrorLog: log.New(io.Discard, "", 0),
	}
	go func() {
		_ = p.server.Serve(tls.NewListener(tunnelListener{p}, &tls.Config{Certificates: []tls.Certificate{a.cert}}))
	}()
	go p.accept()
	return p, nil
}

func (p *proxy) accept() {
	for {
		c, err := p.ln.Accept()
		if err != nil {
			close(p.tunnels)
			return
		}
		go func() {
			req, err := http.ReadRequest(bufio.NewReader(c))
			if err != nil || req.Method != http.MethodConnect {
				_, _ = c.Write([]byte("HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n"))
				_ = c.Close()
				return
			}
			_, _ = c.Write([]byte("HTTP/1.1 200 OK\r\n\r\n"))
			p.tunnels <- c
		}()
	}
}

func (p *proxy) record(w http.ResponseWriter, r *http.Request) {
	p.mu.Lock()
	p.reqs = append(p.reqs, request{
		Method: r.Method,
		Host:   r.Host,
		URI:    r.URL.RequestURI(),
		Bearer: strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "),
	})
	p.mu.Unlock()
	p.handler.ServeHTTP(w, r)
}

func (p *proxy) requests() []request {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]request(nil), p.reqs...)
}

func (p *proxy) close() {
	_ = p.ln.Close()
	_ = p.server.Close()
}

type tunnelListener struct{ p *proxy }

func (l tunnelListener) Accept() (net.Conn, error) {
	c, ok := <-l.p.tunnels
	if !ok {
		return nil, net.ErrClosed
	}
	return c, nil
}

func (l tunnelListener) Close() error   { return nil }
func (l tunnelListener) Addr() net.Addr { return l.p.ln.Addr() }

type garmin struct {
	diDelay       time.Duration
	rejectBearers map[string]bool

	mu     sync.Mutex
	used   map[string]bool
	issued int
}

func newGarmin(diDelay time.Duration, bearers ...string) *garmin {
	g := &garmin{diDelay: diDelay, rejectBearers: map[string]bool{}, used: map[string]bool{}}
	for _, t := range bearers {
		g.rejectBearers[t] = true
	}
	return g
}

func (g *garmin) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/di-oauth2-service/oauth/token" {
		g.token(w, r)
		return
	}
	if g.rejectBearers[strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")] {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	_, _ = fmt.Fprintf(w, `{"path":%q}`, r.URL.Path)
}

func (g *garmin) token(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	rt := r.PostForm.Get("refresh_token")
	g.mu.Lock()
	reused := g.used[rt]
	g.used[rt] = true
	g.issued++
	n := g.issued
	g.mu.Unlock()
	time.Sleep(g.diDelay)
	if reused {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":"invalid_grant"}`)
		return
	}
	_, _ = fmt.Fprintf(w, `{"token_type":"Bearer","access_token":"AT-%d","refresh_token":"RT-%d","expires_in":3600,"refresh_token_expires_in":7776000}`, n, n)
}
