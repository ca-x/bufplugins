package binding

import (
	"bytes"
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

func (b DefaultBinder) Bind(_ context.Context, req Request, _ httpadapter.MethodSpec, rule httpadapter.HTTPBinding, dst proto.Message) error {
	if dst == nil {
		return errors.New("bind request: nil destination")
	}
	consumed := make(consumedFields)
	if err := b.bindBody(req.HTTPRequest(), rule, dst, consumed); err != nil {
		return err
	}
	if err := bindValues(req.QueryParams(), dst, consumed); err != nil {
		return fmt.Errorf("bind query: %w", err)
	}
	if values, err := req.FormParams(); err != nil {
		return fmt.Errorf("bind form: %w", err)
	} else if len(values) > 0 {
		if err := bindValues(values, dst, consumed); err != nil {
			return fmt.Errorf("bind form: %w", err)
		}
	}
	for _, param := range rule.PathParams {
		value := pathParamValue(req, param)
		if value == "" {
			continue
		}
		if err := setPathField(dst.ProtoReflect(), param, []string{value}); err != nil {
			return fmt.Errorf("bind path %q: %w", param.Name, err)
		}
	}
	return nil
}

func (b DefaultBinder) bindBody(r *http.Request, rule httpadapter.HTTPBinding, dst proto.Message, consumed consumedFields) error {
	selector := rule.Body
	if r == nil || r.Body == nil || selector == "" {
		return nil
	}
	contentType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if contentType == "" {
		contentType = "application/json"
	}
	switch contentType {
	case "application/json":
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("read body: %w", err)
		}
		if len(bytes.TrimSpace(body)) == 0 {
			return nil
		}
		if selector == "*" {
			if err := b.UnmarshalOptions.Unmarshal(body, dst); err != nil {
				return fmt.Errorf("decode json body: %w", err)
			}
			consumed.Add(selector)
			return nil
		}
		var raw json.RawMessage
		if err := json.Unmarshal(body, &raw); err != nil {
			return fmt.Errorf("decode json body: %w", err)
		}
		parent := dst.ProtoReflect()
		fieldPath := rule.BodyFieldPath
		if len(fieldPath) == 0 {
			var err error
			fieldPath, err = httpadapter.CompileFieldPath(parent.Descriptor(), selector)
			if err != nil {
				return fmt.Errorf("body field %q: %w", selector, err)
			}
		}
		if len(fieldPath) > 1 {
			var err error
			parent, err = mutableMessageForPath(parent, fieldPath[:len(fieldPath)-1])
			if err != nil {
				return fmt.Errorf("body field %q: %w", selector, err)
			}
		}
		field := fieldPath[len(fieldPath)-1]
		holder := dynamicFieldMessage(parent, field)
		if holder == nil {
			return fmt.Errorf("body field %q is not a message", selector)
		}
		if err := b.UnmarshalOptions.Unmarshal(raw, holder); err != nil {
			return fmt.Errorf("decode json body field %q: %w", selector, err)
		}
		parent.Set(field, protoreflect.ValueOfMessage(holder.ProtoReflect()))
		consumed.Add(selector)
		return nil
	case "application/x-www-form-urlencoded", "multipart/form-data":
		return nil
	default:
		return fmt.Errorf("unsupported content type %q", contentType)
	}
}

func dynamicFieldMessage(parent protoreflect.Message, field protoreflect.FieldDescriptor) proto.Message {
	if field.IsList() || field.IsMap() || (field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind) {
		return nil
	}
	msg := parent.NewField(field).Message()
	if !msg.IsValid() {
		return nil
	}
	return msg.Interface()
}

type consumedFields map[string]struct{}

func (fields consumedFields) Add(selector string) {
	if selector == "" || fields == nil {
		return
	}
	fields[selector] = struct{}{}
}

func (fields consumedFields) Contains(selector string) bool {
	if len(fields) == 0 {
		return false
	}
	if _, ok := fields["*"]; ok {
		return true
	}
	for consumed := range fields {
		if selector == consumed || strings.HasPrefix(selector, consumed+".") {
			return true
		}
	}
	return false
}

func bindValues(values url.Values, dst proto.Message, consumed consumedFields) error {
	for name, vals := range values {
		if len(vals) == 0 {
			continue
		}
		if consumed.Contains(name) {
			continue
		}
		if err := setField(dst.ProtoReflect(), name, vals); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func setField(msg protoreflect.Message, selector string, values []string) error {
	fieldPath, err := httpadapter.CompileFieldPath(msg.Descriptor(), selector)
	if err != nil {
		return err
	}
	return setFieldPath(msg, fieldPath, values)
}

func setPathField(msg protoreflect.Message, param httpadapter.PathParam, values []string) error {
	fieldPath := param.FieldPath
	if len(fieldPath) == 0 {
		return setField(msg, param.Field, values)
	}
	return setFieldPath(msg, fieldPath, values)
}

func setFieldPath(msg protoreflect.Message, fieldPath []protoreflect.FieldDescriptor, values []string) error {
	parent, err := mutableMessageForPath(msg, fieldPath[:len(fieldPath)-1])
	if err != nil {
		return err
	}
	return assignField(parent, fieldPath[len(fieldPath)-1], values)
}

func mutableMessageForPath(msg protoreflect.Message, fieldPath []protoreflect.FieldDescriptor) (protoreflect.Message, error) {
	for _, field := range fieldPath {
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			return nil, fmt.Errorf("field %q is not a message", field.Name())
		}
		child := msg.Get(field).Message()
		if !child.IsValid() {
			child = msg.NewField(field).Message()
			msg.Set(field, protoreflect.ValueOfMessage(child))
		}
		msg = child
	}
	return msg, nil
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

func pathParamValue(req Request, param httpadapter.PathParam) string {
	if param.Template == "" {
		return req.PathParam(param.Name)
	}
	segments := param.TemplateParts
	if len(segments) == 0 {
		segments = strings.Split(param.Template, "/")
	}
	values := make([]string, 0, len(segments))
	nameIndex := 0
	for _, segment := range segments {
		switch segment {
		case "*", "**":
			if nameIndex >= len(param.Names) {
				return ""
			}
			value := req.PathParam(param.Names[nameIndex])
			nameIndex++
			if value == "" {
				return ""
			}
			values = append(values, value)
		default:
			values = append(values, segment)
		}
	}
	return strings.Join(values, "/")
}
