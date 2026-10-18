package requestlog

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

// Format returns one request as the golden files record it: a line with
// the method and the request URI, then the body, if any, indented by two
// spaces. A JSON body is indented with its keys in the order sent. A
// multipart body is one line per part with the size and hash of its data, so
// the random boundary never shows. Another body is one quoted line.
// Format reads and closes r.Body.
func Format(r *http.Request) (string, error) {
	var body []byte
	if r.Body != nil {
		var err error
		body, err = io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err != nil {
			return "", err
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s\n", r.Method, r.URL.RequestURI())
	if len(body) == 0 {
		return b.String(), nil
	}
	mediaType, params, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	switch {
	case strings.HasPrefix(mediaType, "multipart/"):
		if err := logParts(&b, multipart.NewReader(bytes.NewReader(body), params["boundary"])); err != nil {
			return "", err
		}
	case json.Valid(body):
		var out bytes.Buffer
		_ = json.Indent(&out, body, "  ", "  ")
		fmt.Fprintf(&b, "  %s\n", out.Bytes())
	default:
		fmt.Fprintf(&b, "  %s\n", strconv.Quote(string(body)))
	}
	return b.String(), nil
}

func logParts(b *strings.Builder, mr *multipart.Reader) error {
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		data, err := io.ReadAll(p)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "  part %q", p.FormName())
		if p.FileName() != "" {
			fmt.Fprintf(b, " file %q", p.FileName())
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(b, " %d bytes sha256 %x\n", len(data), sum[:8])
	}
}
