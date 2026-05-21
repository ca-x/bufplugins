package httpadapter

import (
	"fmt"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func CompileServiceSpec(spec ServiceSpec) (ServiceSpec, error) {
	for methodIndex := range spec.Methods {
		method := &spec.Methods[methodIndex]
		if len(method.HTTPBindings) == 0 {
			continue
		}
		if method.RequestFactory == nil {
			return ServiceSpec{}, fmt.Errorf("%s: nil request factory", method.Procedure)
		}
		request := method.RequestFactory()
		if request == nil {
			return ServiceSpec{}, fmt.Errorf("%s: request factory returned nil", method.Procedure)
		}
		requestDesc := request.ProtoReflect().Descriptor()

		var responseDesc protoreflect.MessageDescriptor
		if method.ResponseFactory != nil {
			response := method.ResponseFactory()
			if response != nil {
				responseDesc = response.ProtoReflect().Descriptor()
			}
		}

		for bindingIndex := range method.HTTPBindings {
			binding := &method.HTTPBindings[bindingIndex]
			if binding.Body != "" && binding.Body != "*" && len(binding.BodyFieldPath) == 0 {
				fieldPath, err := CompileFieldPath(requestDesc, binding.Body)
				if err != nil {
					return ServiceSpec{}, fmt.Errorf("%s body %q: %w", method.Procedure, binding.Body, err)
				}
				binding.BodyFieldPath = fieldPath
			}
			if binding.ResponseBody != "" && binding.ResponseBody != "*" && len(binding.ResponseBodyFieldPath) == 0 {
				if responseDesc == nil {
					return ServiceSpec{}, fmt.Errorf("%s response_body %q: nil response descriptor", method.Procedure, binding.ResponseBody)
				}
				fieldPath, err := CompileFieldPath(responseDesc, binding.ResponseBody)
				if err != nil {
					return ServiceSpec{}, fmt.Errorf("%s response_body %q: %w", method.Procedure, binding.ResponseBody, err)
				}
				binding.ResponseBodyFieldPath = fieldPath
			}
			for paramIndex := range binding.PathParams {
				param := &binding.PathParams[paramIndex]
				if len(param.FieldPath) == 0 {
					fieldPath, err := CompileFieldPath(requestDesc, param.Field)
					if err != nil {
						return ServiceSpec{}, fmt.Errorf("%s path field %q: %w", method.Procedure, param.Field, err)
					}
					param.FieldPath = fieldPath
				}
				if param.Template != "" && len(param.TemplateParts) == 0 {
					param.TemplateParts = strings.Split(param.Template, "/")
				}
			}
		}
	}
	return spec, nil
}

func CompileFieldPath(desc protoreflect.MessageDescriptor, selector string) ([]protoreflect.FieldDescriptor, error) {
	if desc == nil {
		return nil, fmt.Errorf("nil message descriptor")
	}
	if selector == "" {
		return nil, fmt.Errorf("empty field selector")
	}
	parts := strings.Split(selector, ".")
	fields := make([]protoreflect.FieldDescriptor, 0, len(parts))
	current := desc
	for i, part := range parts {
		field := FindField(current, part)
		if field == nil {
			return nil, fmt.Errorf("field %q not found", selector)
		}
		fields = append(fields, field)
		if i == len(parts)-1 {
			return fields, nil
		}
		if field.Kind() != protoreflect.MessageKind && field.Kind() != protoreflect.GroupKind {
			return nil, fmt.Errorf("field %q is not a message", part)
		}
		current = field.Message()
	}
	return fields, nil
}
