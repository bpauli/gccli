package requestlog

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFormat(t *testing.T) {
	upload := func(boundary string) (string, string) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		if err := w.SetBoundary(boundary); err != nil {
			t.Fatal(err)
		}
		part, _ := w.CreateFormFile("file", "run.fit")
		_, _ = part.Write([]byte("fit\n"))
		_ = w.Close()
		return w.FormDataContentType(), buf.String()
	}
	typeA, bodyA := upload("aaaa")
	typeB, bodyB := upload("bbbb")

	tests := []struct {
		name, method, uri, contentType, body, want string
	}{
		{"no body", "GET", "/a?b=1", "", "", "GET /a?b=1\n"},
		{"json keeps key order", "POST", "/a", "application/json", `{"b":1,"a":[2]}`, "POST /a\n  {\n    \"b\": 1,\n    \"a\": [\n      2\n    ]\n  }\n"},
		{"multipart", "POST", "/up", typeA, bodyA, "POST /up\n  part \"file\" file \"run.fit\" 4 bytes sha256 245f96285b109ccf\n"},
		{"multipart other boundary", "POST", "/up", typeB, bodyB, "POST /up\n  part \"file\" file \"run.fit\" 4 bytes sha256 245f96285b109ccf\n"},
		{"other body", "PUT", "/a", "text/plain", "x\ny", "PUT /a\n  \"x\\ny\"\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.uri, strings.NewReader(tc.body))
			if tc.contentType != "" {
				r.Header.Set("Content-Type", tc.contentType)
			}
			got, err := Format(r)
			if err != nil || got != tc.want {
				t.Errorf("Format = %q, %v\nwant %q", got, err, tc.want)
			}
		})
	}
}
