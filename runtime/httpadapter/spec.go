package httpadapter

import (
	"context"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type MessageFactory func() proto.Message

type UnaryInvoker func(context.Context, proto.Message) (proto.Message, error)

type ServiceSpec struct {
	Name        string
	GoName      string
	ConnectPath string
	Methods     []MethodSpec
}

type MethodSpec struct {
	ServiceName     string
	ServiceGoName   string
	Name            string
	GoName          string
	Procedure       string
	ConnectPath     string
	HTTPBindings    []HTTPBinding
	RequestFactory  MessageFactory
	ResponseFactory MessageFactory
	UnaryInvoker    UnaryInvoker
	ClientStreaming bool
	ServerStreaming bool
}

type HTTPBinding struct {
	Method                string
	Path                  string
	PathParams            []PathParam
	Body                  string
	BodyFieldPath         []protoreflect.FieldDescriptor
	ResponseBody          string
	ResponseBodyFieldPath []protoreflect.FieldDescriptor
}

type PathParam struct {
	Name          string
	Field         string
	FieldPath     []protoreflect.FieldDescriptor
	Template      string
	TemplateParts []string
	Names         []string
}

func (m MethodSpec) Key() string {
	return MethodKey(m.ServiceGoName, m.GoName)
}

func MethodKey(serviceGoName, methodGoName string) string {
	return serviceGoName + "." + methodGoName
}
