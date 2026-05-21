package echoadapter

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ca-x/bufplugins/runtime/httpadapter"
	"github.com/labstack/echo/v5"
)

func TestStripPrefixUpdatesRawPathDuringHandlerAndRestoresRequest(t *testing.T) {
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/files/a/b" {
			t.Fatalf("Path during handler = %q, want %q", r.URL.Path, "/files/a/b")
		}
		if r.URL.RawPath != "/files/a%2Fb" {
			t.Fatalf("RawPath during handler = %q, want %q", r.URL.RawPath, "/files/a%2Fb")
		}
	})
	handler := stripPrefix("/api/v1", next)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/v1/files/a%2Fb", nil)
	req.URL.Path = "/api/v1/files/a/b"
	req.URL.RawPath = "/api/v1/files/a%2Fb"
	originalPath := req.URL.Path
	originalRawPath := req.URL.RawPath

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if req.URL.Path != originalPath {
		t.Fatalf("Path after handler = %q, want restored %q", req.URL.Path, originalPath)
	}
	if req.URL.RawPath != originalRawPath {
		t.Fatalf("RawPath after handler = %q, want restored %q", req.URL.RawPath, originalRawPath)
	}
}

func TestRegisterRejectsStreamingRESTBinding(t *testing.T) {
	err := (ServiceRegistrar{
		Spec: httpadapter.ServiceSpec{
			Methods: []httpadapter.MethodSpec{
				{
					Procedure:       "/test.Service/Stream",
					ServerStreaming: true,
					HTTPBindings: []httpadapter.HTTPBinding{
						{Method: http.MethodGet, Path: "/stream"},
					},
				},
			},
		},
		Config: NewConfig(),
	}).Register(echo.New())
	if err == nil {
		t.Fatal("Register() error = nil, want streaming REST binding error")
	}
	if !strings.Contains(err.Error(), "streaming REST bindings are not supported") {
		t.Fatalf("Register() error = %v, want streaming REST binding error", err)
	}
}
