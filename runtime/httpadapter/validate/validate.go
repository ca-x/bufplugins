package validate

import (
	"context"

	"github.com/ca-x/bufplugins/runtime/httpadapter"
	"google.golang.org/protobuf/proto"
)

type Validator interface {
	Validate(context.Context, httpadapter.MethodSpec, proto.Message) error
}

type NoopValidator struct{}

func (NoopValidator) Validate(context.Context, httpadapter.MethodSpec, proto.Message) error {
	return nil
}
