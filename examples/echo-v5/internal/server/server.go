package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	helloworldv1 "github.com/ca-x/bufplugins/examples/echo-v5/api/helloworld/v1"
	"github.com/ca-x/bufplugins/examples/echo-v5/api/helloworld/v1/helloworldv1connect"
	helloworldv1echo "github.com/ca-x/bufplugins/examples/echo-v5/api/helloworld/v1/helloworldv1echo"
	"github.com/ca-x/bufplugins/runtime/echoadapter"
	adaptervalidate "github.com/ca-x/bufplugins/runtime/httpadapter/validate"
	"github.com/labstack/echo/v5"
)

type Greeter struct{}

func (Greeter) SayHello(ctx context.Context, req *connect.Request[helloworldv1.SayHelloRequest]) (*connect.Response[helloworldv1.SayHelloResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	return connect.NewResponse(&helloworldv1.SayHelloResponse{
		Message: fmt.Sprintf("Hello, %s!", name),
	}), nil
}

func (Greeter) LuckySearch(ctx context.Context, req *connect.Request[helloworldv1.LuckySearchRequest]) (*connect.Response[helloworldv1.LuckySearchResponse], error) {
	keyword := strings.TrimSpace(req.Msg.GetKeyword())
	if keyword == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("keyword is required"))
	}

	limit := req.Msg.GetLimit()
	if limit == 0 {
		limit = 1
	}

	results := make([]string, 0, limit)
	for i := int32(1); i <= limit; i++ {
		results = append(results, fmt.Sprintf("%s-%d", keyword, i))
	}

	return connect.NewResponse(&helloworldv1.LuckySearchResponse{
		Results: results,
	}), nil
}

func (Greeter) CreateGreeting(ctx context.Context, req *connect.Request[helloworldv1.CreateGreetingRequest]) (*connect.Response[helloworldv1.CreateGreetingResponse], error) {
	name := strings.TrimSpace(req.Msg.GetName())
	message := strings.TrimSpace(req.Msg.GetMessage())
	if name == "" || message == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name and message are required"))
	}

	return connect.NewResponse(&helloworldv1.CreateGreetingResponse{
		Id:      "greeting-" + strings.ToLower(strings.ReplaceAll(name, " ", "-")),
		Message: message,
	}), nil
}

type Config struct {
	Service        helloworldv1connect.GreeterServiceHandler
	ConnectOptions []connect.HandlerOption
	EchoOptions    []echoadapter.Option
}

func NewHandler(cfg Config) (http.Handler, error) {
	e := echo.New()
	if err := RegisterRoutes(e, cfg); err != nil {
		return nil, err
	}
	return e, nil
}

func RegisterRoutes(e *echo.Echo, cfg Config) error {
	svc := cfg.Service
	if svc == nil {
		svc = Greeter{}
	}

	validator, err := adaptervalidate.NewProtovalidate()
	if err != nil {
		return err
	}

	opts := append([]echoadapter.Option{
		echoadapter.WithConnectOptions(cfg.ConnectOptions...),
		echoadapter.WithValidator(validator),
	}, cfg.EchoOptions...)

	registrar := helloworldv1echo.NewGreeterServiceEchoRegistrar(svc, opts...)
	return registrar.Register(e)
}
