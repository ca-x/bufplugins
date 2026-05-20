package response

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
	selected, ok, err := selectBody(src, rule.ResponseBody)
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

func selectBody(src proto.Message, selector string) (any, bool, error) {
	if selector == "" || selector == "*" {
		return src, true, nil
	}
	msg := src.ProtoReflect()
	value, present, err := valueBySelector(msg, selector)
	if err != nil {
		return nil, false, err
	}
	if !value.IsValid() || !present {
		return nil, false, nil
	}
	return value.Interface(), true, nil
}

func valueBySelector(msg protoreflect.Message, selector string) (protoreflect.Value, bool, error) {
	var field protoreflect.FieldDescriptor
	current := msg
	parts := strings.Split(selector, ".")
	for i, part := range parts {
		field = findField(current.Descriptor(), part)
		if field == nil {
			return protoreflect.Value{}, false, fmt.Errorf("response field %q not found", selector)
		}
		value := current.Get(field)
		if i == len(parts)-1 {
			return value, current.Has(field), nil
		}
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			return protoreflect.Value{}, false, fmt.Errorf("response field %q is not a message", part)
		}
		current = value.Message()
	}
	return protoreflect.Value{}, false, nil
}

func findField(desc protoreflect.MessageDescriptor, name string) protoreflect.FieldDescriptor {
	if field := desc.Fields().ByName(protoreflect.Name(name)); field != nil {
		return field
	}
	return desc.Fields().ByJSONName(name)
}

func marshalSelected(options protojson.MarshalOptions, selected any) ([]byte, error) {
	switch value := selected.(type) {
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
