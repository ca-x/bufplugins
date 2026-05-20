package echoadapter

import (
	"connectrpc.com/connect"
	"github.com/ca-x/bufplugins/runtime/httpadapter"
	"github.com/ca-x/bufplugins/runtime/httpadapter/binding"
	adaptererrors "github.com/ca-x/bufplugins/runtime/httpadapter/errors"
	"github.com/ca-x/bufplugins/runtime/httpadapter/response"
	"github.com/ca-x/bufplugins/runtime/httpadapter/validate"
	"github.com/labstack/echo/v5"
)

type Option func(*Config)

type Config struct {
	GroupPrefix          string
	Middlewares          []echo.MiddlewareFunc
	ConnectOptions       []connect.HandlerOption
	RequestBinder        binding.RequestBinder
	ResponseWriter       response.ResponseWriter
	Validator            validate.Validator
	ErrorMapper          adaptererrors.ErrorMapper
	ErrorWriter          adaptererrors.ErrorWriter
	MethodRequestBinders map[string]binding.RequestBinder
	MethodWriters        map[string]response.ResponseWriter
	MethodValidators     map[string]validate.Validator
	MethodErrorMappers   map[string]adaptererrors.ErrorMapper
	MethodErrorWriters   map[string]adaptererrors.ErrorWriter
}

func NewConfig(opts ...Option) Config {
	cfg := Config{
		RequestBinder:        binding.NewDefaultBinder(),
		ResponseWriter:       response.NewDefaultWriter(),
		Validator:            validate.NoopValidator{},
		ErrorMapper:          adaptererrors.DefaultMapper{},
		ErrorWriter:          adaptererrors.DefaultWriter{},
		MethodRequestBinders: make(map[string]binding.RequestBinder),
		MethodWriters:        make(map[string]response.ResponseWriter),
		MethodValidators:     make(map[string]validate.Validator),
		MethodErrorMappers:   make(map[string]adaptererrors.ErrorMapper),
		MethodErrorWriters:   make(map[string]adaptererrors.ErrorWriter),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

func WithGroupPrefix(prefix string) Option {
	return func(cfg *Config) {
		cfg.GroupPrefix = prefix
	}
}

func WithMiddleware(middlewares ...echo.MiddlewareFunc) Option {
	return func(cfg *Config) {
		cfg.Middlewares = append(cfg.Middlewares, middlewares...)
	}
}

func WithConnectOptions(options ...connect.HandlerOption) Option {
	return func(cfg *Config) {
		cfg.ConnectOptions = append(cfg.ConnectOptions, options...)
	}
}

func WithRequestBinder(binder binding.RequestBinder) Option {
	return func(cfg *Config) {
		cfg.RequestBinder = binder
	}
}

func WithMethodRequestBinder(service, method string, binder binding.RequestBinder) Option {
	return func(cfg *Config) {
		cfg.MethodRequestBinders[methodKey(service, method)] = binder
	}
}

func WithResponseWriter(writer response.ResponseWriter) Option {
	return func(cfg *Config) {
		cfg.ResponseWriter = writer
	}
}

func WithMethodResponseWriter(service, method string, writer response.ResponseWriter) Option {
	return func(cfg *Config) {
		cfg.MethodWriters[methodKey(service, method)] = writer
	}
}

func WithValidator(validator validate.Validator) Option {
	return func(cfg *Config) {
		cfg.Validator = validator
	}
}

func WithMethodValidator(service, method string, validator validate.Validator) Option {
	return func(cfg *Config) {
		cfg.MethodValidators[methodKey(service, method)] = validator
	}
}

func WithErrorMapper(mapper adaptererrors.ErrorMapper) Option {
	return func(cfg *Config) {
		cfg.ErrorMapper = mapper
	}
}

func WithMethodErrorMapper(service, method string, mapper adaptererrors.ErrorMapper) Option {
	return func(cfg *Config) {
		cfg.MethodErrorMappers[methodKey(service, method)] = mapper
	}
}

func WithErrorWriter(writer adaptererrors.ErrorWriter) Option {
	return func(cfg *Config) {
		cfg.ErrorWriter = writer
	}
}

func WithMethodErrorWriter(service, method string, writer adaptererrors.ErrorWriter) Option {
	return func(cfg *Config) {
		cfg.MethodErrorWriters[methodKey(service, method)] = writer
	}
}

func (c Config) binder(spec httpadapter.MethodSpec) binding.RequestBinder {
	if binder := c.MethodRequestBinders[spec.Key()]; binder != nil {
		return binder
	}
	return c.RequestBinder
}

func (c Config) writer(spec httpadapter.MethodSpec) response.ResponseWriter {
	if writer := c.MethodWriters[spec.Key()]; writer != nil {
		return writer
	}
	return c.ResponseWriter
}

func (c Config) validator(spec httpadapter.MethodSpec) validate.Validator {
	if validator := c.MethodValidators[spec.Key()]; validator != nil {
		return validator
	}
	return c.Validator
}

func (c Config) errorMapper(spec httpadapter.MethodSpec) adaptererrors.ErrorMapper {
	if mapper := c.MethodErrorMappers[spec.Key()]; mapper != nil {
		return mapper
	}
	return c.ErrorMapper
}

func (c Config) errorWriter(spec httpadapter.MethodSpec) adaptererrors.ErrorWriter {
	if writer := c.MethodErrorWriters[spec.Key()]; writer != nil {
		return writer
	}
	return c.ErrorWriter
}

func methodKey(service, method string) string {
	return service + "." + method
}
