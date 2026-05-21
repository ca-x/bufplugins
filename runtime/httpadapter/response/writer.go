package response

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/ca-x/bufplugins/runtime/httpadapter"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Response interface {
	Header() http.Header
	JSONBlob(int, []byte) error
	NoContent(int) error
}

type ResponseWriter interface {
	Write(context.Context, Response, httpadapter.MethodSpec, httpadapter.HTTPBinding, proto.Message) error
}

type DefaultWriter struct {
	MarshalOptions protojson.MarshalOptions
}

type selectedBody struct {
	message proto.Message
	fields  []protoreflect.FieldDescriptor
}

func NewDefaultWriter() DefaultWriter {
	return DefaultWriter{
		MarshalOptions: protojson.MarshalOptions{
			UseProtoNames:   false,
			EmitUnpopulated: false,
		},
	}
}

func (w DefaultWriter) Write(_ context.Context, resp Response, _ httpadapter.MethodSpec, rule httpadapter.HTTPBinding, src proto.Message) error {
	status := http.StatusOK
	if src == nil {
		return resp.NoContent(http.StatusNoContent)
	}
	selected, ok, err := selectBody(src, rule)
	if err != nil {
		return err
	}
	if !ok {
		return resp.NoContent(http.StatusNoContent)
	}
	data, err := marshalSelected(w.MarshalOptions, selected)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	resp.Header().Set("Content-Type", "application/json")
	return resp.JSONBlob(status, data)
}

func selectBody(src proto.Message, rule httpadapter.HTTPBinding) (any, bool, error) {
	selector := rule.ResponseBody
	if selector == "" || selector == "*" {
		return src, true, nil
	}
	msg := src.ProtoReflect()
	fields := rule.ResponseBodyFieldPath
	var present bool
	var err error
	if len(fields) == 0 {
		_, fields, present, err = valueBySelector(msg, selector)
		if err != nil {
			return nil, false, err
		}
	} else {
		present, err = hasFieldPath(msg, fields)
		if err != nil {
			return nil, false, err
		}
	}
	if !present {
		return nil, false, nil
	}
	return selectedBody{message: src, fields: fields}, true, nil
}

func valueBySelector(msg protoreflect.Message, selector string) (protoreflect.Value, []protoreflect.FieldDescriptor, bool, error) {
	fields, err := httpadapter.CompileFieldPath(msg.Descriptor(), selector)
	if err != nil {
		return protoreflect.Value{}, nil, false, fmt.Errorf("response field %q: %w", selector, err)
	}
	value, present, err := valueByFieldPath(msg, fields)
	return value, fields, present, err
}

func hasFieldPath(msg protoreflect.Message, fields []protoreflect.FieldDescriptor) (bool, error) {
	_, present, err := valueByFieldPath(msg, fields)
	return present, err
}

func valueByFieldPath(msg protoreflect.Message, fields []protoreflect.FieldDescriptor) (protoreflect.Value, bool, error) {
	current := msg
	for i, field := range fields {
		value := current.Get(field)
		if i == len(fields)-1 {
			return value, current.Has(field), nil
		}
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			return protoreflect.Value{}, false, fmt.Errorf("response field %q is not a message", field.Name())
		}
		if !current.Has(field) {
			return protoreflect.Value{}, false, nil
		}
		current = value.Message()
	}
	return protoreflect.Value{}, false, nil
}

func marshalSelected(options protojson.MarshalOptions, selected any) ([]byte, error) {
	switch value := selected.(type) {
	case selectedBody:
		return marshalSelectedBody(options, value)
	case proto.Message:
		return options.Marshal(value)
	case protoreflect.Message:
		return options.Marshal(value.Interface())
	case protoreflect.Value:
		return marshalSelected(options, value.Interface())
	case protoreflect.EnumNumber:
		return []byte(strconv.FormatInt(int64(value), 10)), nil
	case []byte:
		return json.Marshal(value)
	default:
		return json.Marshal(value)
	}
}

func marshalSelectedBody(options protojson.MarshalOptions, selected selectedBody) ([]byte, error) {
	raw, err := options.Marshal(selected.message)
	if err != nil {
		return nil, err
	}
	for _, field := range selected.fields {
		var object map[string]json.RawMessage
		if err := json.Unmarshal(raw, &object); err != nil {
			return nil, err
		}
		name := field.JSONName()
		if options.UseProtoNames {
			name = string(field.Name())
		}
		next, ok := object[name]
		if !ok {
			return nil, fmt.Errorf("response field %q not found in marshaled body", field.Name())
		}
		raw = next
	}
	return raw, nil
}
