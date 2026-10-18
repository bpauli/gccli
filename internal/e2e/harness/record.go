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

type Recorder struct {
	base http.RoundTripper
	dir  string
	seq  atomic.Int64
}

func NewRecorder(dir string, base http.RoundTripper) (*Recorder, error) {
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
	return &Recorder{base: base, dir: dir}, nil
}

func (r *Recorder) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := r.base.RoundTrip(req)
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
		err = os.WriteFile(filepath.Join(r.dir, fmt.Sprintf("%04d.json", r.seq.Add(1))), data, 0o600)
	}
	if err != nil {
		return nil, fmt.Errorf("record a %s response: %w", req.Method, err)
	}
	return resp, nil
}
