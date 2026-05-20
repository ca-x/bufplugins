package validate

import (
	"context"

	protovalidate "buf.build/go/protovalidate"
	"github.com/ca-x/bufplugins/runtime/httpadapter"
	"google.golang.org/protobuf/proto"
)

type Protovalidate struct {
	validator protovalidate.Validator
}

func NewProtovalidate(options ...protovalidate.ValidatorOption) (*Protovalidate, error) {
	validator, err := protovalidate.New(options...)
	if err != nil {
		return nil, err
	}
	return &Protovalidate{validator: validator}, nil
}

func MustProtovalidate(options ...protovalidate.ValidatorOption) *Protovalidate {
	validator, err := NewProtovalidate(options...)
	if err != nil {
		panic(err)
	}
	return validator
}

func (v *Protovalidate) Validate(ctx context.Context, _ httpadapter.MethodSpec, msg proto.Message) error {
	if v == nil || v.validator == nil || msg == nil {
		return nil
	}
	_ = ctx
	return v.validator.Validate(msg)
}
