package generator

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	annotations "google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
)

type httpBinding struct {
	Method       string
	Path         string
	PathParams   []pathParam
	Body         string
	ResponseBody string
}

type pathParam struct {
	Name  string
	Field string
}

func methodBindings(method *protogen.Method) ([]httpBinding, error) {
	options := method.Desc.Options()
	if options == nil || !proto.HasExtension(options, annotations.E_Http) {
		return nil, nil
	}
	ext := proto.GetExtension(options, annotations.E_Http)
	rule, ok := ext.(*annotations.HttpRule)
	if !ok {
		return nil, fmt.Errorf("%s: google.api.http extension has unexpected type %T", method.Desc.FullName(), ext)
	}
	var bindings []httpBinding
	primary, ok, err := bindingFromRule(rule)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", method.Desc.FullName(), err)
	}
	if ok {
		bindings = append(bindings, primary)
	}
	for _, additional := range rule.GetAdditionalBindings() {
		binding, ok, err := bindingFromRule(additional)
		if err != nil {
			return nil, fmt.Errorf("%s additional binding: %w", method.Desc.FullName(), err)
		}
		if ok {
			bindings = append(bindings, binding)
		}
	}
	return bindings, nil
}

func bindingFromRule(rule *annotations.HttpRule) (httpBinding, bool, error) {
	method, path, err := pattern(rule)
	if err != nil {
		return httpBinding{}, false, err
	}
	if method == "" || path == "" {
		return httpBinding{}, false, nil
	}
	echoPath, params, err := googlePathToEcho(path)
	if err != nil {
		return httpBinding{}, false, err
	}
	return httpBinding{
		Method:       method,
		Path:         echoPath,
		PathParams:   params,
		Body:         rule.GetBody(),
		ResponseBody: rule.GetResponseBody(),
	}, true, nil
}

func pattern(rule *annotations.HttpRule) (string, string, error) {
	switch p := rule.GetPattern().(type) {
	case *annotations.HttpRule_Get:
		return http.MethodGet, p.Get, nil
	case *annotations.HttpRule_Put:
		return http.MethodPut, p.Put, nil
	case *annotations.HttpRule_Post:
		return http.MethodPost, p.Post, nil
	case *annotations.HttpRule_Delete:
		return http.MethodDelete, p.Delete, nil
	case *annotations.HttpRule_Patch:
		return http.MethodPatch, p.Patch, nil
	case *annotations.HttpRule_Custom:
		custom := rule.GetCustom()
		if custom == nil {
			return "", "", fmt.Errorf("custom HTTP rule is nil")
		}
		return strings.ToUpper(custom.GetKind()), custom.GetPath(), nil
	default:
		return "", "", nil
	}
}

var pathVariablePattern = regexp.MustCompile(`\{([^}=]+)(=([^}]+))?\}`)

func googlePathToEcho(path string) (string, []pathParam, error) {
	var params []pathParam
	converted := pathVariablePattern.ReplaceAllStringFunc(path, func(match string) string {
		parts := pathVariablePattern.FindStringSubmatch(match)
		field := parts[1]
		name := sanitizeParamName(field)
		if parts[3] == "**" {
			name = "*"
		}
		params = append(params, pathParam{Name: name, Field: field})
		if name == "*" {
			return "*"
		}
		return ":" + name
	})
	if strings.Contains(converted, "{") || strings.Contains(converted, "}") {
		return "", nil, fmt.Errorf("invalid path template %q", path)
	}
	return converted, params, nil
}

func sanitizeParamName(name string) string {
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "param"
	}
	return b.String()
}
