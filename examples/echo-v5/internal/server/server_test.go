package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	helloworldv1 "github.com/ca-x/bufplugins/examples/echo-v5/api/helloworld/v1"
	"github.com/ca-x/bufplugins/examples/echo-v5/api/helloworld/v1/helloworldv1connect"
	helloworldv1echo "github.com/ca-x/bufplugins/examples/echo-v5/api/helloworld/v1/helloworldv1echo"
	"github.com/ca-x/bufplugins/runtime/echoadapter"
	adaptervalidate "github.com/ca-x/bufplugins/runtime/httpadapter/validate"
	"github.com/labstack/echo/v5"
)

func TestRegisterRoutesAcceptsInjectedService(t *testing.T) {
	t.Parallel()

	var svc helloworldv1connect.GreeterServiceHandler = Greeter{}
	e := echo.New()

	if err := RegisterRoutes(e, Config{Service: svc}); err != nil {
		t.Fatalf("RegisterRoutes() error = %v", err)
	}
}

func TestNewHandlerReturnsHTTPHandler(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}
	if _, ok := handler.(http.Handler); !ok {
		t.Fatalf("NewHandler() returned %T, want http.Handler", handler)
	}
}

func TestGeneratedRegistrarSupportsDIOptions(t *testing.T) {
	t.Parallel()

	validator, err := adaptervalidate.NewProtovalidate()
	if err != nil {
		t.Fatalf("NewProtovalidate() error = %v", err)
	}

	registrar := helloworldv1echo.NewGreeterServiceEchoRegistrar(
		Greeter{},
		echoadapter.WithValidator(validator),
	)

	if err := registrar.Register(echo.New()); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
}

func TestGreeterImplementsGeneratedHandler(t *testing.T) {
	t.Parallel()

	var _ helloworldv1connect.GreeterServiceHandler = Greeter{}
	_ = helloworldv1.SayHelloRequest{}
}

func TestRESTRoutes(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/helloworld/czyt", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /helloworld/czyt status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"message":"Hello, czyt!"}` {
		t.Fatalf("GET /helloworld/czyt body = %s", got)
	}

	req = httptest.NewRequest(http.MethodPost, "/greetings", bytes.NewBufferString(`{"name":"czyt","message":"Hi"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("POST /greetings status = %d body = %s", rec.Code, rec.Body.String())
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `"Hi"` {
		t.Fatalf("POST /greetings body = %s", got)
	}
}

func TestRESTValidationError(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/helloworld/a", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("validation status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestConnectRoute(t *testing.T) {
	t.Parallel()

	handler, err := NewHandler(Config{})
	if err != nil {
		t.Fatalf("NewHandler() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, helloworldv1connect.GreeterServiceSayHelloProcedure, bytes.NewBufferString(`{"name":"czyt"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("connect route status = %d body = %s", rec.Code, rec.Body.String())
	}
}
