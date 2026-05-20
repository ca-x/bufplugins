package echoadapter

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/ca-x/bufplugins/runtime/httpadapter"
	adaptererrors "github.com/ca-x/bufplugins/runtime/httpadapter/errors"
	"github.com/labstack/echo/v5"
)

type ServiceRegistrar struct {
	Spec           httpadapter.ServiceSpec
	ConnectHandler http.Handler
	Config         Config
}

func RegisterService(e *echo.Echo, spec httpadapter.ServiceSpec, connectHandler http.Handler, opts ...Option) error {
	return ServiceRegistrar{
		Spec:           spec,
		ConnectHandler: connectHandler,
		Config:         NewConfig(opts...),
	}.Register(e)
}

func (r ServiceRegistrar) Register(e *echo.Echo) error {
	if e == nil {
		return adaptererrors.Wrap(adaptererrors.KindInternal, errors.New("register echo service: nil echo"))
	}
	group := e.Group(r.Config.GroupPrefix, r.Config.Middlewares...)
	if r.ConnectHandler != nil && r.Spec.ConnectPath != "" {
		connectPath := strings.TrimSuffix(r.Spec.ConnectPath, "/")
		group.Any(connectPath+"/*", echo.WrapHandler(stripPrefix(r.Config.GroupPrefix, r.ConnectHandler)))
	}
	for _, method := range r.Spec.Methods {
		if method.ClientStreaming || method.ServerStreaming {
			continue
		}
		for _, rule := range method.HTTPBindings {
			spec := method
			binding := rule
			group.Add(binding.Method, binding.Path, func(c *echo.Context) error {
				return r.handleREST(c.Request().Context(), echoRequest{c: c}, echoResponse{c: c}, spec, binding)
			})
		}
	}
	return nil
}

func stripPrefix(prefix string, next http.Handler) http.Handler {
	prefix = strings.TrimRight(prefix, "/")
	if prefix == "" || prefix == "/" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		original := r.URL.Path
		r.URL.Path = strings.TrimPrefix(r.URL.Path, prefix)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		next.ServeHTTP(w, r)
		r.URL.Path = original
	})
}

func (r ServiceRegistrar) handleREST(ctx context.Context, req echoRequest, resp echoResponse, spec httpadapter.MethodSpec, rule httpadapter.HTTPBinding) error {
	msg := spec.RequestFactory()
	if err := r.Config.binder(spec).Bind(ctx, req, spec, rule, msg); err != nil {
		return r.writeError(ctx, resp, spec, adaptererrors.Wrap(adaptererrors.KindBinding, err))
	}
	if err := r.Config.validator(spec).Validate(ctx, spec, msg); err != nil {
		return r.writeError(ctx, resp, spec, adaptererrors.Wrap(adaptererrors.KindValidation, err))
	}
	out, err := spec.UnaryInvoker(ctx, msg)
	if err != nil {
		return r.writeError(ctx, resp, spec, err)
	}
	if err := r.Config.writer(spec).Write(ctx, resp, spec, rule, out); err != nil {
		return r.writeError(ctx, resp, spec, err)
	}
	return nil
}

func (r ServiceRegistrar) writeError(ctx context.Context, resp echoResponse, spec httpadapter.MethodSpec, err error) error {
	httpErr := r.Config.errorMapper(spec).MapError(ctx, spec, err)
	return r.Config.errorWriter(spec).WriteError(ctx, resp, spec, httpErr)
}

type echoRequest struct {
	c *echo.Context
}

func (r echoRequest) HTTPRequest() *http.Request {
	return r.c.Request()
}

func (r echoRequest) PathParam(name string) string {
	return r.c.Param(name)
}

func (r echoRequest) QueryParams() url.Values {
	return r.c.QueryParams()
}

func (r echoRequest) FormParams() (url.Values, error) {
	return r.c.FormValues()
}

type echoResponse struct {
	c *echo.Context
}

func (r echoResponse) Header() http.Header {
	return r.c.Response().Header()
}

func (r echoResponse) JSON(code int, body any) error {
	return r.c.JSON(code, body)
}

func (r echoResponse) JSONBlob(code int, data []byte) error {
	return r.c.JSONBlob(code, data)
}

func (r echoResponse) NoContent(code int) error {
	return r.c.NoContent(code)
}
