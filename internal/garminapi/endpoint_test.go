package garminapi

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

func mustRoute(t *testing.T, e Endpoint) *route {
	t.Helper()
	e.Host = ConnectAPI
	if e.Method == "" {
		e.Method = http.MethodGet
	}
	r, err := compile(e)
	if err != nil {
		t.Fatalf("compile %s: %v", e.Name, err)
	}
	return r
}

type endpointCase struct {
	name string
	args map[string]string
	want string
}

func testEndpoints(t *testing.T, table []Endpoint, cases []endpointCase) {
	t.Helper()
	for _, e := range table {
		if !slices.ContainsFunc(cases, func(c endpointCase) bool { return c.name == e.Name }) {
			t.Errorf("row %s has no case", e.Name)
		}
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Method + " " + r.URL.RequestURI()
				_, _ = w.Write([]byte(`[{"id":1}]`))
			}))
			var body []byte
			if r, ok := endpoints[tc.name]; ok && r.Body {
				body = []byte(`{}`)
			}
			data, err := client.Call(context.Background(), tc.name, tc.args, body)
			if err != nil || got != tc.want || string(data) != `[{"id":1}]` {
				t.Errorf("Call = %q, %v; sent %q, want %q", data, err, got, tc.want)
			}
		})
	}
}

type sent struct {
	line, accept, contentType, body string
}

func TestCallSendsTheTableRequest(t *testing.T) {
	tests := []struct {
		name  string
		e     Endpoint
		args  map[string]string
		body  []byte
		reply int
		want  sent
		data  string
	}{
		{
			name: "query pairs keep their order and an empty optional pair is left out",
			e: Endpoint{
				Name: "activities.list", Path: "/activitylist-service/activities/search/activities?start={start}&limit={limit}&activityType={activityType}",
				Params: []Param{{Name: "activityType", Optional: true}},
			},
			args: map[string]string{"start": "0", "limit": "20", "activityType": ""},
			want: sent{line: "GET /activitylist-service/activities/search/activities?start=0&limit=20", accept: "application/json"},
			data: `[]`,
		},
		{
			name: "an optional pair with a value is sent in place",
			e: Endpoint{
				Name: "activities.list", Path: "/a?start={start}&limit={limit}&activityType={activityType}",
				Params: []Param{{Name: "activityType", Optional: true}},
			},
			args: map[string]string{"start": "0", "limit": "20", "activityType": "running"},
			want: sent{line: "GET /a?start=0&limit=20&activityType=running", accept: "application/json"},
			data: `[]`,
		},
		{
			name: "a default fills an empty value",
			e:    Endpoint{Name: "training.plans", Path: "/trainingplan-service/trainingplan/plans?locale={locale}", Params: []Param{{Name: "locale", Default: "en"}}},
			want: sent{line: "GET /trainingplan-service/trainingplan/plans?locale=en", accept: "application/json"},
			data: `[]`,
		},
		{
			name: "literal pairs go out where written and a repeated name takes one value",
			e:    Endpoint{Name: "health.rhr", Path: "/userstats-service/wellness/daily/{displayName}?fromDate={date}&untilDate={date}&metricId=60"},
			args: map[string]string{"displayName": "jo", "date": "2024-01-15"},
			want: sent{line: "GET /userstats-service/wellness/daily/jo?fromDate=2024-01-15&untilDate=2024-01-15&metricId=60", accept: "application/json"},
			data: `{}`,
		},
		{
			name: "path and query values are escaped",
			e:    Endpoint{Name: "x.get", Path: "/x/{id}?q={q}"},
			args: map[string]string{"id": "a/b c", "q": "r&s=t"},
			want: sent{line: "GET /x/a%2Fb%20c?q=r%26s%3Dt", accept: "application/json"},
			data: `{}`,
		},
		{
			name: "non-ASCII values are sent as percent-encoded UTF-8",
			e:    Endpoint{Name: "x.get", Path: "/x/{id}?q={q}"},
			args: map[string]string{"id": "läufe", "q": "läufe"},
			want: sent{line: "GET /x/l%C3%A4ufe?q=l%C3%A4ufe", accept: "application/json"},
			data: `{}`,
		},
		{
			name: "a body is sent as JSON",
			e:    Endpoint{Name: "workouts.schedule.add", Method: http.MethodPost, Path: "/workout-service/schedule/{id}", Body: true},
			args: map[string]string{"id": "9"},
			body: []byte(`{"date":"2024-01-15"}`),
			want: sent{line: "POST /workout-service/schedule/9", accept: "application/json", contentType: "application/json", body: `{"date":"2024-01-15"}`},
			data: `{"workoutScheduleId":10}`,
		},
		{
			name:  "a write without a body sends no content type, and 204 returns nil",
			e:     Endpoint{Name: "gear.unlink", Method: http.MethodPut, Path: "/gear-service/gear/unlink/{gear}/activity/{activity}"},
			args:  map[string]string{"gear": "abc", "activity": "1"},
			reply: http.StatusNoContent,
			want:  sent{line: "PUT /gear-service/gear/unlink/abc/activity/1", accept: "application/json"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got sent
			_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				b, _ := io.ReadAll(r.Body)
				got = sent{r.Method + " " + r.URL.RequestURI(), r.Header.Get("Accept"), r.Header.Get("Content-Type"), string(b)}
				if tc.reply != 0 {
					w.WriteHeader(tc.reply)
				}
				_, _ = w.Write([]byte(tc.data))
			}))
			data, err := client.call(context.Background(), mustRoute(t, tc.e), tc.args, tc.body)
			if err != nil {
				t.Fatalf("call: %v", err)
			}
			if got != tc.want {
				t.Errorf("sent %+v\nwant %+v", got, tc.want)
			}
			if string(data) != tc.data || (tc.data == "") != (data == nil) {
				t.Errorf("result %q, want %q", data, tc.data)
			}
		})
	}
}

func TestCallRefusesBeforeSending(t *testing.T) {
	status := Endpoint{Name: "goals.list", Path: "/goal-service/goal/goals?status={status}", Params: []Param{{Name: "status", Enum: []string{"active", "future", "past"}}}}
	post := Endpoint{Name: "workouts.schedule.add", Method: http.MethodPost, Path: "/workout-service/schedule/{id}", Body: true}
	tests := []struct {
		name string
		e    Endpoint
		args map[string]string
		body []byte
		want string
	}{
		{"missing required", status, nil, nil, "endpoint goals.list: status: required"},
		{"value not in enum", status, map[string]string{"status": "bogus"}, nil, `endpoint goals.list: status: "bogus" is not one of active, future, past`},
		{"unknown argument", status, map[string]string{"status": "active", "stauts": "past"}, nil, "endpoint goals.list: stauts: not a parameter of this endpoint"},
		{"empty path value", post, map[string]string{"id": ""}, []byte(`{}`), "endpoint workouts.schedule.add: id: required"},
		{"dot segment path value", post, map[string]string{"id": ".."}, []byte(`{}`), `endpoint workouts.schedule.add: id: ".." is not a path segment`},
		{"body missing", post, map[string]string{"id": "9"}, nil, "endpoint workouts.schedule.add: needs a request body"},
		{"body not taken", status, map[string]string{"status": "active"}, []byte(`{}`), "endpoint goals.list: takes no request body"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, client := testServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				t.Errorf("request sent: %s %s", r.Method, r.URL.RequestURI())
			}))
			_, err := client.call(context.Background(), mustRoute(t, tc.e), tc.args, tc.body)
			var epErr *EndpointError
			if !errors.As(err, &epErr) || err.Error() != tc.want {
				t.Errorf("err = %v, want EndpointError %q", err, tc.want)
			}
		})
	}
}

func TestCallUnknownName(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("request sent: %s", r.URL.RequestURI())
	}))
	_, err := client.Call(context.Background(), "goals.lsit", nil, nil)
	if err == nil || err.Error() != "endpoint goals.lsit: not in the endpoint table" {
		t.Errorf("err = %v", err)
	}
}

func TestCallReturnsAPIErrors(t *testing.T) {
	_, client := testServer(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	_, err := client.call(context.Background(), mustRoute(t, Endpoint{Name: "badges.earned", Path: "/badge-service/badge/earned"}), nil, nil)
	var apiErr *GarminAPIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusNotFound {
		t.Errorf("err = %v, want GarminAPIError 404", err)
	}
}

func TestIndexRejectsDuplicateNames(t *testing.T) {
	row := Endpoint{Name: "goals.list", Method: http.MethodGet, Host: ConnectAPI, Path: "/goal-service/goal/goals"}
	_, err := index([]Endpoint{row}, []Endpoint{row})
	if err == nil || err.Error() != `garminapi: endpoint "goals.list" is defined twice` {
		t.Errorf("err = %v", err)
	}
}

func TestIndexRejectsMalformedRows(t *testing.T) {
	ok := Endpoint{Name: "x.get", Method: http.MethodGet, Host: ConnectAPI, Path: "/x/{id}?q={q}"}
	tests := []struct {
		name string
		edit func(*Endpoint)
		want string
	}{
		{"name", func(e *Endpoint) { e.Name = "X_Get" }, "name must be"},
		{"name without area", func(e *Endpoint) { e.Name = "goals" }, "name must be"},
		{"method", func(e *Endpoint) { e.Method = "get" }, `method "get"`},
		{"host unset", func(e *Endpoint) { e.Host = 0 }, "unknown host"},
		{"GET body", func(e *Endpoint) { e.Body = true }, "a GET has no body"},
		{"relative path", func(e *Endpoint) { e.Path = "x" }, "must start with /"},
		{"open brace", func(e *Endpoint) { e.Path = "/x/{id" }, "unmatched {"},
		{"close brace", func(e *Endpoint) { e.Path = "/x/id}" }, "unmatched }"},
		{"space in path", func(e *Endpoint) { e.Path = "/x y/{id}" }, `path text "/x y/" may hold only`},
		{"hash in path", func(e *Endpoint) { e.Path = "/x/{id}#frag" }, `path text "#frag" may hold only`},
		{"escape in path", func(e *Endpoint) { e.Path = "/x%2F/{id}" }, `path text "/x%2F/" may hold only`},
		{"partial query value", func(e *Endpoint) { e.Path = "/x?q=a{q}" }, "one whole {placeholder}"},
		{"param without placeholder", func(e *Endpoint) { e.Params = []Param{{Name: "nope"}} }, `"nope" has no placeholder`},
		{"param twice", func(e *Endpoint) { e.Params = []Param{{Name: "q"}, {Name: "q"}} }, "listed twice"},
		{"optional path", func(e *Endpoint) { e.Params = []Param{{Name: "id", Optional: true}} }, "cannot be optional"},
		{"optional with default", func(e *Endpoint) { e.Params = []Param{{Name: "q", Optional: true, Default: "a"}} }, "already makes it optional"},
		{"default outside enum", func(e *Endpoint) { e.Params = []Param{{Name: "q", Default: "a", Enum: []string{"b"}}} }, "not in its enum"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := ok
			tc.edit(&e)
			_, err := index([]Endpoint{e})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func TestEveryTableIsIndexed(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	rows := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.CompositeLit); ok {
				if arr, ok := lit.Type.(*ast.ArrayType); ok {
					if id, ok := arr.Elt.(*ast.Ident); ok && id.Name == "Endpoint" {
						rows += len(lit.Elts)
					}
				}
			}
			return true
		})
	}
	if rows != len(endpoints) {
		t.Errorf("the package declares %d endpoint rows but the index holds %d: add each service table to mustIndex in endpoint.go", rows, len(endpoints))
	}
}

func TestCallSitesMatchTable(t *testing.T) {
	fset := token.NewFileSet()
	var files []string
	for _, pattern := range []string{"../cmd/*.go", "*.go", "../e2e/*.go"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	called := map[string]bool{}
	for _, file := range files {
		e2e := strings.HasPrefix(file, "../e2e/")
		if strings.HasSuffix(file, "_test.go") && !e2e {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 4 {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); !ok || sel.Sel.Name != "Call" {
				return true
			}
			pos := fset.Position(call.Pos())
			lit, ok := call.Args[1].(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				t.Errorf("%s: pass the endpoint name to Client.Call as a string literal, so the write census can read it", pos)
				return true
			}
			name, _ := strconv.Unquote(lit.Value)
			r, ok := endpoints[name]
			if !ok {
				t.Errorf("%s: Client.Call names %q, which is not a row of the endpoint table", pos, name)
				return true
			}
			called[name] = called[name] || !e2e
			id, ok := call.Args[3].(*ast.Ident)
			switch noBody := ok && id.Name == "nil"; {
			case noBody && r.Body:
				t.Errorf("%s: row %s needs a request body, but the call passes nil", pos, name)
			case !noBody && !r.Body:
				t.Errorf("%s: row %s takes no request body, but the call passes one", pos, name)
			}
			keys, ok := literalKeys(call.Args[2])
			if !ok {
				return true
			}
			for _, k := range keys {
				if _, ok := r.params[k]; !ok {
					t.Errorf("%s: row %s has no placeholder {%s}", pos, name, k)
				}
			}
			for _, p := range r.names {
				if rule := r.params[p]; !rule.Optional && rule.Default == "" && !slices.Contains(keys, p) {
					t.Errorf("%s: row %s needs a value for {%s}", pos, name, p)
				}
			}
			return true
		})
	}
	for _, name := range slices.Sorted(maps.Keys(endpoints)) {
		if !called[name] {
			t.Errorf("row %s has no caller in internal/cmd or internal/garminapi: delete it or call it", name)
		}
	}
}

func literalKeys(e ast.Expr) (keys []string, ok bool) {
	if id, isIdent := e.(*ast.Ident); isIdent && id.Name == "nil" {
		return nil, true
	}
	lit, isLit := e.(*ast.CompositeLit)
	if !isLit {
		return nil, false
	}
	if _, isMap := lit.Type.(*ast.MapType); !isMap {
		return nil, false
	}
	for _, el := range lit.Elts {
		kv, isKV := el.(*ast.KeyValueExpr)
		if !isKV {
			return nil, false
		}
		k, isStr := kv.Key.(*ast.BasicLit)
		if !isStr || k.Kind != token.STRING {
			return nil, false
		}
		key, err := strconv.Unquote(k.Value)
		if err != nil {
			return nil, false
		}
		keys = append(keys, key)
	}
	return keys, true
}
