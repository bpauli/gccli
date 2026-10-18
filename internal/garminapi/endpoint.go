package garminapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

var endpoints = mustIndex(
	goalsEndpoints,
	healthEndpoints,
	wellnessEndpoints,
	trainingEndpoints,
	activitiesEndpoints,
	coursesEndpoints,
	workoutsEndpoints,
	bodyEndpoints,
	devicesEndpoints,
	gearEndpoints,
)

// Host is the Garmin server an endpoint lives on.
type Host uint8

const (
	_ Host = iota
	// ConnectAPI is connectapi.garmin.com (or garmin.cn): the Client base URL.
	ConnectAPI
)

// Endpoint is one Garmin request that the CLI sends as the table states it.
//
// Name is <area>.<action>, where area is the Garmin service, such as
// goals.list or badges.in-progress.
//
// Path is the request target after the host, with the query, as it goes on
// the wire. A {name} placeholder in the path part is escaped with
// url.PathEscape. A placeholder in the query part fills one whole value and
// is escaped with url.QueryEscape. Query pairs go out in the order written.
// A literal pair, such as includeAll=true, is always sent. A name can occur
// more than once, and the caller passes its value once.
//
// Every placeholder is required unless its Param says otherwise.
//
// A row is one JSON request. Downloads and uploads stay Client methods.
type Endpoint struct {
	Name   string
	Method string
	Host   Host
	Path   string
	Params []Param
	// Body is true when the request carries a JSON body. Call rejects a body
	// on an endpoint without one, and a missing body on one with it.
	Body bool
}

// Param gives rules to the placeholder with the same Name in Path.
type Param struct {
	Name string
	// Optional lets the caller pass an empty value for a query placeholder.
	// The pair and its separator are then left out.
	Optional bool
	// Default is used when the caller passes an empty value.
	Default string
	// Enum lists the values Call accepts, after Default is applied.
	Enum []string
}

// EndpointError is a Call that the table refuses before any request is sent.
type EndpointError struct {
	Endpoint string
	Param    string
	Problem  string
}

func (e *EndpointError) Error() string {
	if e.Param == "" {
		return fmt.Sprintf("endpoint %s: %s", e.Endpoint, e.Problem)
	}
	return fmt.Sprintf("endpoint %s: %s: %s", e.Endpoint, e.Param, e.Problem)
}

// EndpointMethod returns the HTTP method of the named endpoint, so the cmd
// write census can find the commands that send a write through Call.
func EndpointMethod(name string) (string, bool) {
	r, ok := endpoints[name]
	if !ok {
		return "", false
	}
	return r.Method, true
}

// Call sends the named endpoint with args filling its placeholders and body
// as the JSON request body (nil for none). It checks every argument before it
// sends anything. The result is the same as ConnectAPI: the response body, or
// nil for 204 No Content and an empty body.
func (c *Client) Call(ctx context.Context, name string, args map[string]string, body []byte) (json.RawMessage, error) {
	r, ok := endpoints[name]
	if !ok {
		return nil, &EndpointError{Endpoint: name, Problem: "not in the endpoint table"}
	}
	return c.call(ctx, r, args, body)
}

func (c *Client) call(ctx context.Context, r *route, args map[string]string, body []byte) (json.RawMessage, error) {
	switch {
	case r.Body && body == nil:
		return nil, &EndpointError{Endpoint: r.Name, Problem: "needs a request body"}
	case !r.Body && body != nil:
		return nil, &EndpointError{Endpoint: r.Name, Problem: "takes no request body"}
	}
	target, err := r.target(args)
	if err != nil {
		return nil, err
	}
	req := request{kind: kindJSON, method: r.Method, path: target, body: body}
	if body != nil {
		req.contentType = "application/json"
	}
	resp, err := c.do(ctx, req)
	if err != nil {
		return nil, err
	}
	return jsonResult(resp), nil
}

func jsonResult(resp response) json.RawMessage {
	if resp.status == http.StatusNoContent || len(resp.body) == 0 {
		return nil
	}
	return json.RawMessage(resp.body)
}

type route struct {
	Endpoint
	path   []piece
	query  []pair
	names  []string
	params map[string]Param
}

type piece struct {
	text  string
	param bool
}

type pair struct {
	key   string
	value piece
}

func (r *route) target(args map[string]string) (string, error) {
	for _, k := range sortedKeys(args) {
		if _, ok := r.params[k]; !ok {
			return "", &EndpointError{Endpoint: r.Name, Param: k, Problem: "not a parameter of this endpoint"}
		}
	}
	vals := make(map[string]string, len(r.names))
	for _, name := range r.names {
		p := r.params[name]
		v := args[name]
		if v == "" {
			v = p.Default
		}
		switch {
		case v == "" && !p.Optional:
			return "", &EndpointError{Endpoint: r.Name, Param: name, Problem: "required"}
		case v != "" && p.Enum != nil && !slices.Contains(p.Enum, v):
			return "", &EndpointError{Endpoint: r.Name, Param: name, Problem: fmt.Sprintf("%q is not one of %s", v, strings.Join(p.Enum, ", "))}
		}
		vals[name] = v
	}

	var b strings.Builder
	for _, pc := range r.path {
		if pc.param {
			v := vals[pc.text]
			if v == "." || v == ".." {
				return "", &EndpointError{Endpoint: r.Name, Param: pc.text, Problem: fmt.Sprintf("%q is not a path segment", v)}
			}
			b.WriteString(url.PathEscape(v))
		} else {
			b.WriteString(pc.text)
		}
	}
	sep := "?"
	for _, q := range r.query {
		v := q.value.text
		if q.value.param {
			if v = vals[v]; v == "" {
				continue
			}
			v = url.QueryEscape(v)
		}
		b.WriteString(sep + q.key + "=" + v)
		sep = "&"
	}
	return b.String(), nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	return keys
}

func mustIndex(tables ...[]Endpoint) map[string]*route {
	idx, err := index(tables...)
	if err != nil {
		panic(err)
	}
	return idx
}

func index(tables ...[]Endpoint) (map[string]*route, error) {
	idx := map[string]*route{}
	for _, table := range tables {
		for _, e := range table {
			if _, dup := idx[e.Name]; dup {
				return nil, fmt.Errorf("garminapi: endpoint %q is defined twice", e.Name)
			}
			r, err := compile(e)
			if err != nil {
				return nil, fmt.Errorf("garminapi: endpoint %q: %w", e.Name, err)
			}
			idx[e.Name] = r
		}
	}
	return idx, nil
}

var (
	endpointNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*(\.[a-z][a-z0-9]*(-[a-z0-9]+)*)+$`)
	placeholderRe  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*$`)
	literalRe      = regexp.MustCompile(`^[A-Za-z0-9._~-]*$`)
	pathTextRe     = regexp.MustCompile(`^[A-Za-z0-9._~/-]*$`)
	methods        = []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete}
)

func compile(e Endpoint) (*route, error) {
	switch {
	case !endpointNameRe.MatchString(e.Name):
		return nil, errors.New("name must be dot-separated lower-case words, like badges.in-progress")
	case !slices.Contains(methods, e.Method):
		return nil, fmt.Errorf("method %q is not GET, POST, PUT, PATCH or DELETE", e.Method)
	case e.Host != ConnectAPI:
		return nil, fmt.Errorf("unknown host %d", e.Host)
	case e.Body && e.Method == http.MethodGet:
		return nil, errors.New("a GET has no body")
	case !strings.HasPrefix(e.Path, "/"):
		return nil, errors.New("path must start with /")
	}

	r := &route{Endpoint: e, params: map[string]Param{}}
	inPath := map[string]bool{}
	use := func(name string) error {
		if !placeholderRe.MatchString(name) {
			return fmt.Errorf("bad placeholder {%s}", name)
		}
		if _, seen := r.params[name]; !seen {
			r.names = append(r.names, name)
			r.params[name] = Param{}
		}
		return nil
	}

	pathPart, queryPart, hasQuery := strings.Cut(e.Path, "?")
	for rest := pathPart; rest != ""; {
		open := strings.IndexByte(rest, '{')
		if open < 0 {
			open = len(rest)
		}
		if strings.ContainsRune(rest[:open], '}') {
			return nil, errors.New("unmatched } in path")
		}
		if !pathTextRe.MatchString(rest[:open]) {
			return nil, fmt.Errorf("path text %q may hold only letters, digits and . _ ~ / -", rest[:open])
		}
		if open > 0 {
			r.path = append(r.path, piece{text: rest[:open]})
		}
		if open == len(rest) {
			break
		}
		name, after, ok := strings.Cut(rest[open+1:], "}")
		if !ok {
			return nil, errors.New("unmatched { in path")
		}
		if err := use(name); err != nil {
			return nil, err
		}
		inPath[name] = true
		r.path = append(r.path, piece{text: name, param: true})
		rest = after
	}
	if hasQuery {
		for _, kv := range strings.Split(queryPart, "&") {
			key, val, ok := strings.Cut(kv, "=")
			if !ok || key == "" || !literalRe.MatchString(key) {
				return nil, fmt.Errorf("query pair %q must be key=value", kv)
			}
			p := pair{key: key, value: piece{text: val}}
			if strings.HasPrefix(val, "{") && strings.HasSuffix(val, "}") {
				p.value = piece{text: val[1 : len(val)-1], param: true}
				if err := use(p.value.text); err != nil {
					return nil, err
				}
			} else if !literalRe.MatchString(val) {
				return nil, fmt.Errorf("query value %q must be a literal or one whole {placeholder}", val)
			}
			r.query = append(r.query, p)
		}
	}

	for _, p := range e.Params {
		prev, ok := r.params[p.Name]
		switch {
		case !ok:
			return nil, fmt.Errorf("param %q has no placeholder in the path", p.Name)
		case prev.Name != "":
			return nil, fmt.Errorf("param %q is listed twice", p.Name)
		case p.Optional && p.Default != "":
			return nil, fmt.Errorf("param %q: a default already makes it optional", p.Name)
		case p.Optional && inPath[p.Name]:
			return nil, fmt.Errorf("param %q: a path placeholder cannot be optional", p.Name)
		case p.Default != "" && p.Enum != nil && !slices.Contains(p.Enum, p.Default):
			return nil, fmt.Errorf("param %q: default %q is not in its enum", p.Name, p.Default)
		}
		r.params[p.Name] = p
	}
	return r, nil
}
