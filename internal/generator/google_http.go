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
	Name     string
	Field    string
	Template string
	Names    []string
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
	usedNames := make(map[string]string)
	var converted strings.Builder
	var convertErr error
	last := 0
	matches := pathVariablePattern.FindAllStringSubmatchIndex(path, -1)
	for _, match := range matches {
		converted.WriteString(path[last:match[0]])
		field := path[match[2]:match[3]]
		template := ""
		if match[6] >= 0 {
			template = path[match[6]:match[7]]
		}
		route, param, err := pathVariableToEcho(path, field, template, match[1] == len(path), usedNames)
		if err != nil {
			convertErr = err
			break
		}
		converted.WriteString(route)
		params = append(params, param)
		last = match[1]
	}
	if convertErr != nil {
		return "", nil, convertErr
	}
	converted.WriteString(path[last:])
	convertedPath := converted.String()
	if strings.Contains(convertedPath, "{") || strings.Contains(convertedPath, "}") {
		return "", nil, fmt.Errorf("invalid path template %q", path)
	}
	return convertedPath, params, nil
}

func pathVariableToEcho(path, field, template string, variableAtPathEnd bool, usedNames map[string]string) (string, pathParam, error) {
	if template == "" {
		name := sanitizeParamName(field)
		if err := registerPathParamName(path, usedNames, name, field); err != nil {
			return "", pathParam{}, err
		}
		return ":" + name, pathParam{Name: name, Field: field}, nil
	}
	if template == "**" {
		if !variableAtPathEnd {
			return "", pathParam{}, fmt.Errorf("deep wildcard path variable %q must end the Echo route in template %q", field, path)
		}
		if err := registerPathParamName(path, usedNames, "*", field); err != nil {
			return "", pathParam{}, err
		}
		return "*", pathParam{Name: "*", Field: field}, nil
	}

	segments := strings.Split(template, "/")
	routeSegments := make([]string, 0, len(segments))
	names := make([]string, 0, len(segments))
	for i, segment := range segments {
		switch segment {
		case "":
			return "", pathParam{}, fmt.Errorf("invalid empty segment in path variable %q in template %q", field, path)
		case "*":
			name := allocatePathParamName(usedNames, sanitizeParamName(field), field)
			routeSegments = append(routeSegments, ":"+name)
			names = append(names, name)
		case "**":
			if i != len(segments)-1 {
				return "", pathParam{}, fmt.Errorf("deep wildcard must be the last segment in path variable %q in template %q", field, path)
			}
			if !variableAtPathEnd {
				return "", pathParam{}, fmt.Errorf("deep wildcard path variable %q must end the Echo route in template %q", field, path)
			}
			if err := registerPathParamName(path, usedNames, "*", field); err != nil {
				return "", pathParam{}, err
			}
			routeSegments = append(routeSegments, "*")
			names = append(names, "*")
		default:
			if strings.Contains(segment, "*") {
				return "", pathParam{}, fmt.Errorf("invalid wildcard segment %q in path variable %q in template %q", segment, field, path)
			}
			routeSegments = append(routeSegments, segment)
		}
	}

	return strings.Join(routeSegments, "/"), pathParam{
		Field:    field,
		Template: template,
		Names:    names,
	}, nil
}

func registerPathParamName(path string, usedNames map[string]string, name, field string) error {
	if previous, ok := usedNames[name]; ok {
		return fmt.Errorf("path template %q has colliding route parameter name %q for fields %q and %q", path, name, previous, field)
	}
	usedNames[name] = field
	return nil
}

func allocatePathParamName(usedNames map[string]string, base, field string) string {
	name := base
	for suffix := 1; ; suffix++ {
		if _, ok := usedNames[name]; !ok {
			usedNames[name] = field
			return name
		}
		name = fmt.Sprintf("%s_%d", base, suffix)
	}
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
