package binding

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/ca-x/bufplugins/runtime/httpadapter"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Request interface {
	HTTPRequest() *http.Request
	PathParam(name string) string
	QueryParams() url.Values
	FormParams() (url.Values, error)
}

type RequestBinder interface {
	Bind(context.Context, Request, httpadapter.MethodSpec, httpadapter.HTTPBinding, proto.Message) error
}

type DefaultBinder struct {
	UnmarshalOptions protojson.UnmarshalOptions
}

func NewDefaultBinder() DefaultBinder {
	return DefaultBinder{
		UnmarshalOptions: protojson.UnmarshalOptions{
			DiscardUnknown: true,
		},
	}
}

func (b DefaultBinder) Bind(_ context.Context, req Request, spec httpadapter.MethodSpec, rule httpadapter.HTTPBinding, dst proto.Message) error {
	if dst == nil {
		return errors.New("bind request: nil destination")
	}
	if err := b.bindBody(req.HTTPRequest(), rule.Body, dst); err != nil {
		return err
	}
	if err := bindValues(req.QueryParams(), dst, nil); err != nil {
		return fmt.Errorf("bind query: %w", err)
	}
	if values, err := req.FormParams(); err != nil {
		return fmt.Errorf("bind form: %w", err)
	} else if len(values) > 0 {
		if err := bindValues(values, dst, nil); err != nil {
			return fmt.Errorf("bind form: %w", err)
		}
	}
	for _, param := range rule.PathParams {
		value := req.PathParam(param.Name)
		if value == "" {
			continue
		}
		if err := setField(dst.ProtoReflect(), param.Field, []string{value}); err != nil {
			return fmt.Errorf("bind path %q: %w", param.Name, err)
		}
	}
	_ = spec
	return nil
}

func (b DefaultBinder) bindBody(r *http.Request, selector string, dst proto.Message) error {
	if r == nil || r.Body == nil || selector == "" {
		return nil
	}
	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	switch contentType {
	case "application/json", "application/json; charset=utf-8":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}
		if len(strings.TrimSpace(string(body))) == 0 {
			return nil
		}
		if selector == "*" {
			if err := b.UnmarshalOptions.Unmarshal(body, dst); err != nil {
				return fmt.Errorf("decode json body: %w", err)
			}
			return nil
		}
		var raw json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return fmt.Errorf("decode json body: %w", err)
		}
		field := dst.ProtoReflect().Descriptor().Fields().ByName(protoreflect.Name(selector))
		if field == nil {
			field = dst.ProtoReflect().Descriptor().Fields().ByJSONName(selector)
		}
		if field == nil {
			return fmt.Errorf("body field %q not found", selector)
		}
		holder := dynamicFieldMessage(dst, field)
		if holder == nil {
			return fmt.Errorf("body field %q is not a message", selector)
		}
		if err := b.UnmarshalOptions.Unmarshal(raw, holder); err != nil {
			return fmt.Errorf("decode json body field %q: %w", selector, err)
		}
		dst.ProtoReflect().Set(field, protoreflect.ValueOfMessage(holder.ProtoReflect()))
		return nil
	case "application/x-www-form-urlencoded", "multipart/form-data":
		return nil
	default:
		return fmt.Errorf("unsupported content type %q", contentType)
	}
}

func dynamicFieldMessage(dst proto.Message, field protoreflect.FieldDescriptor) proto.Message {
	msg := dst.ProtoReflect().NewField(field).Message()
	if !msg.IsValid() {
		return nil
	}
	return msg.Interface()
}

func bindValues(values url.Values, dst proto.Message, consumed map[string]struct{}) error {
	for name, vals := range values {
		if len(vals) == 0 {
			continue
		}
		if consumed != nil {
			if _, ok := consumed[name]; ok {
				continue
			}
		}
		if err := setField(dst.ProtoReflect(), name, vals); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func setField(msg protoreflect.Message, selector string, values []string) error {
	parts := strings.Split(selector, ".")
	for i, part := range parts {
		field := findField(msg.Descriptor(), part)
		if field == nil {
			return fmt.Errorf("field %q not found", selector)
		}
		if i < len(parts)-1 {
			if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
				return fmt.Errorf("field %q is not a message", part)
			}
			child := msg.Get(field).Message()
			if !child.IsValid() {
				child = msg.NewField(field).Message()
				msg.Set(field, protoreflect.ValueOfMessage(child))
			}
			msg = child
			continue
		}
		return assignField(msg, field, values)
	}
	return nil
}

func findField(desc protoreflect.MessageDescriptor, name string) protoreflect.FieldDescriptor {
	fields := desc.Fields()
	if field := fields.ByName(protoreflect.Name(name)); field != nil {
		return field
	}
	return fields.ByJSONName(name)
}

func assignField(msg protoreflect.Message, field protoreflect.FieldDescriptor, values []string) error {
	if field.IsList() {
		list := msg.Mutable(field).List()
		for _, value := range values {
			parsed, err := parseValue(field, value)
			if err != nil {
				return err
			}
			list.Append(parsed)
		}
		return nil
	}
	parsed, err := parseValue(field, values[len(values)-1])
	if err != nil {
		return err
	}
	msg.Set(field, parsed)
	return nil
}

func parseValue(field protoreflect.FieldDescriptor, raw string) (protoreflect.Value, error) {
	switch field.Kind() {
	case protoreflect.BoolKind:
		v, err := strconv.ParseBool(raw)
		return protoreflect.ValueOfBool(v), err
	case protoreflect.EnumKind:
		if n, err := strconv.ParseInt(raw, 10, 32); err == nil {
			return protoreflect.ValueOfEnum(protoreflect.EnumNumber(n)), nil
		}
		enum := field.Enum().Values().ByName(protoreflect.Name(raw))
		if enum == nil {
			return protoreflect.Value{}, fmt.Errorf("unknown enum value %q", raw)
		}
		return protoreflect.ValueOfEnum(enum.Number()), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		v, err := strconv.ParseInt(raw, 10, 32)
		return protoreflect.ValueOfInt32(int32(v)), err
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		v, err := strconv.ParseInt(raw, 10, 64)
		return protoreflect.ValueOfInt64(v), err
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		v, err := strconv.ParseUint(raw, 10, 32)
		return protoreflect.ValueOfUint32(uint32(v)), err
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		v, err := strconv.ParseUint(raw, 10, 64)
		return protoreflect.ValueOfUint64(v), err
	case protoreflect.FloatKind:
		v, err := strconv.ParseFloat(raw, 32)
		return protoreflect.ValueOfFloat32(float32(v)), err
	case protoreflect.DoubleKind:
		v, err := strconv.ParseFloat(raw, 64)
		return protoreflect.ValueOfFloat64(v), err
	case protoreflect.StringKind:
		return protoreflect.ValueOfString(raw), nil
	case protoreflect.BytesKind:
		return protoreflect.ValueOfBytes([]byte(raw)), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("unsupported field kind %s", field.Kind())
	}
}
