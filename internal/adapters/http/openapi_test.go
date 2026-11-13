package http

import (
	"bytes"
	nethttp "net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// apiPrefix is the path prefix of every route, declared once in the spec as
// the server URL.
const apiPrefix = "/api/v1"

// ginParam matches a gin path parameter such as ":id".
var ginParam = regexp.MustCompile(`:([A-Za-z0-9_]+)`)

// refPattern matches a local reference such as "#/components/schemas/User".
var refPattern = regexp.MustCompile(`\$ref: ["']?(#/[^"'\s]+)`)

// openAPIDoc is the subset of the OpenAPI document checked by the tests.
type openAPIDoc struct {
	OpenAPI string `yaml:"openapi"`
	Servers []struct {
		URL string `yaml:"url"`
	} `yaml:"servers"`
	Paths      map[string]map[string]any `yaml:"paths"`
	Components map[string]map[string]any `yaml:"components"`
}

func newSpecServer(t *testing.T) *Server {
	t.Helper()

	srv, err := NewServer(Deps{
		Users:        NewUserHandler(&fakeUserService{}),
		Auth:         noopAuthRoutes{},
		Authenticate: func(c *gin.Context) { c.AbortWithStatus(nethttp.StatusUnauthorized) },
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return srv
}

func parseSpec(t *testing.T) openAPIDoc {
	t.Helper()

	var doc openAPIDoc
	if err := yaml.Unmarshal(openAPISpec, &doc); err != nil {
		t.Fatalf("openapi.yaml is not valid YAML: %v", err)
	}
	return doc
}

func TestServer_OpenAPISpec_IsServedWithoutAuthentication(t *testing.T) {
	srv := newSpecServer(t)

	req := httptest.NewRequest(nethttp.MethodGet, apiPrefix+"/openapi.yaml", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != nethttp.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, nethttp.StatusOK)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/yaml") {
		t.Errorf("Content-Type = %q, want application/yaml", ct)
	}
	if !bytes.Equal(rec.Body.Bytes(), openAPISpec) {
		t.Error("body differs from the embedded openapi.yaml")
	}
}

func TestOpenAPISpec_IsOpenAPI31WithAPIPrefixServer(t *testing.T) {
	doc := parseSpec(t)

	if !strings.HasPrefix(doc.OpenAPI, "3.1.") {
		t.Errorf("openapi = %q, want 3.1.x", doc.OpenAPI)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].URL != apiPrefix {
		t.Errorf("servers = %+v, want a single server with url %q", doc.Servers, apiPrefix)
	}
}

func TestOpenAPISpec_DocumentsExactlyTheRegisteredRoutes(t *testing.T) {
	srv := newSpecServer(t)
	doc := parseSpec(t)

	var registered []string
	for _, r := range srv.router.Routes() {
		path, ok := strings.CutPrefix(r.Path, apiPrefix)
		if !ok {
			t.Errorf("route %s %s is outside %s", r.Method, r.Path, apiPrefix)
			continue
		}
		registered = append(registered, r.Method+" "+ginParam.ReplaceAllString(path, "{$1}"))
	}

	var documented []string
	for path, item := range doc.Paths {
		for method := range item {
			switch method {
			case "get", "put", "post", "delete", "patch", "head", "options":
				documented = append(documented, strings.ToUpper(method)+" "+path)
			}
		}
	}

	slices.Sort(registered)
	slices.Sort(documented)
	if !slices.Equal(registered, documented) {
		t.Errorf("documented routes differ from registered ones\nregistered: %v\ndocumented: %v", registered, documented)
	}
}

func TestOpenAPISpec_LocalReferencesResolve(t *testing.T) {
	doc := parseSpec(t)

	for _, m := range refPattern.FindAllSubmatch(openAPISpec, -1) {
		ref := string(m[1])
		parts := strings.Split(strings.TrimPrefix(ref, "#/"), "/")
		if len(parts) != 3 || parts[0] != "components" {
			t.Errorf("unsupported reference %q", ref)
			continue
		}
		if _, ok := doc.Components[parts[1]][parts[2]]; !ok {
			t.Errorf("reference %q does not resolve", ref)
		}
	}
}
