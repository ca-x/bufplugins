package httpadapter

import (
	"context"

	"google.golang.org/protobuf/proto"
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
	Method       string
	Path         string
	PathParams   []PathParam
	Body         string
	ResponseBody string
}

type PathParam struct {
	Name  string
	Field string
}

func (m MethodSpec) Key() string {
	return m.ServiceGoName + "." + m.GoName
}
