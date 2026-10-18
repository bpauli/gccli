package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/bpauli/gccli/conformance"
)

// A Recorder writes each API response to its own file in a directory.
// Wrap gives the transport that records.
type Recorder struct {
	dir string
	seq atomic.Int64
}

func NewRecorder(dir string) (*Recorder, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(entries) > 0 {
		return nil, fmt.Errorf("%s is not empty", dir)
	}
	return &Recorder{dir: dir}, nil
}

// Wrap returns a transport that sends each request through base and writes
// the response to the recorder's directory. The record holds the method,
// the host, the URI, the status and a JSON body, never a request header or
// a request body.
func (r *Recorder) Wrap(base http.RoundTripper) http.RoundTripper {
	return &recording{rec: r, base: base}
}

type recording struct {
	rec  *Recorder
	base http.RoundTripper
}

func (t *recording) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return nil, err
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))

	rec := conformance.Record{Method: req.Method, Host: req.URL.Host, URI: req.URL.RequestURI(), Status: resp.StatusCode}
	if json.Valid(body) {
		rec.Body = body
	} else {
		rec.NonJSONBytes = len(body)
	}
	data, err := conformance.EncodeRecord(rec)
	if err == nil {
		err = os.WriteFile(filepath.Join(t.rec.dir, fmt.Sprintf("%04d.json", t.rec.seq.Add(1))), data, 0o600)
	}
	if err != nil {
		return nil, fmt.Errorf("record a %s response: %w", req.Method, err)
	}
	return resp, nil
}
