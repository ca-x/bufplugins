package errors

import (
	"context"
	stderrors "errors"
	"net/http"

	"connectrpc.com/connect"
	"github.com/ca-x/bufplugins/runtime/httpadapter"
)

type Kind string

const (
	KindBinding    Kind = "binding"
	KindValidation Kind = "validation"
	KindConnect    Kind = "connect"
	KindInternal   Kind = "internal"
)

type Error struct {
	Kind Kind
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return string(e.Kind)
	}
	return e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}

func Wrap(kind Kind, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Kind: kind, Err: err}
}

type HTTPError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
	Cause   error  `json:"-"`
}

type ErrorMapper interface {
	MapError(context.Context, httpadapter.MethodSpec, error) *HTTPError
}

type ErrorWriter interface {
	WriteError(context.Context, Response, httpadapter.MethodSpec, *HTTPError) error
}

type Response interface {
	JSON(int, any) error
}

type DefaultMapper struct{}

func (DefaultMapper) MapError(_ context.Context, _ httpadapter.MethodSpec, err error) *HTTPError {
	if err == nil {
		return nil
	}
	var typed *Error
	if stderrors.As(err, &typed) {
		switch typed.Kind {
		case KindBinding:
			return &HTTPError{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: typed.Err.Error(), Cause: err}
		case KindValidation:
			return &HTTPError{Status: http.StatusUnprocessableEntity, Code: "VALIDATION_ERROR", Message: typed.Err.Error(), Cause: err}
		}
	}
	if code := connect.CodeOf(err); code != connect.CodeUnknown {
		return &HTTPError{
			Status:  statusFromConnectCode(code),
			Code:    code.String(),
			Message: err.Error(),
			Cause:   err,
		}
	}
	return &HTTPError{Status: http.StatusInternalServerError, Code: "INTERNAL", Message: "internal server error", Cause: err}
}

type DefaultWriter struct{}

func (DefaultWriter) WriteError(_ context.Context, resp Response, _ httpadapter.MethodSpec, err *HTTPError) error {
	if err == nil {
		return nil
	}
	return resp.JSON(err.Status, err)
}

func statusFromConnectCode(code connect.Code) int {
	switch code {
	case connect.CodeCanceled:
		return 499
	case connect.CodeUnknown:
		return http.StatusInternalServerError
	case connect.CodeInvalidArgument:
		return http.StatusBadRequest
	case connect.CodeDeadlineExceeded:
		return http.StatusGatewayTimeout
	case connect.CodeNotFound:
		return http.StatusNotFound
	case connect.CodeAlreadyExists:
		return http.StatusConflict
	case connect.CodePermissionDenied:
		return http.StatusForbidden
	case connect.CodeResourceExhausted:
		return http.StatusTooManyRequests
	case connect.CodeFailedPrecondition:
		return http.StatusBadRequest
	case connect.CodeAborted:
		return http.StatusConflict
	case connect.CodeOutOfRange:
		return http.StatusBadRequest
	case connect.CodeUnimplemented:
		return http.StatusNotImplemented
	case connect.CodeInternal:
		return http.StatusInternalServerError
	case connect.CodeUnavailable:
		return http.StatusServiceUnavailable
	case connect.CodeDataLoss:
		return http.StatusInternalServerError
	case connect.CodeUnauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}
