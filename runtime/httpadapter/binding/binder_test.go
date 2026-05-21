package binding

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ca-x/bufplugins/runtime/httpadapter"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type fakeRequest struct {
	httpRequest *http.Request
	query       url.Values
	form        url.Values
	path        map[string]string
	formErr     error
}

func (r fakeRequest) HTTPRequest() *http.Request {
	return r.httpRequest
}

func (r fakeRequest) PathParam(name string) string {
	return r.path[name]
}

func (r fakeRequest) QueryParams() url.Values {
	return r.query
}

func (r fakeRequest) FormParams() (url.Values, error) {
	return r.form, r.formErr
}

func TestDefaultBinderBodyStarIsNotOverriddenByQueryOrForm(t *testing.T) {
	httpReq, err := http.NewRequest(http.MethodPost, "/greetings", strings.NewReader(`{"name":"body-name","message":"body-message"}`))
	if err != nil {
		t.Fatal(err)
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=utf-8")

	req := fakeRequest{
		httpRequest: httpReq,
		query: url.Values{
			"name":    {"query-name"},
			"message": {"query-message"},
		},
		form: url.Values{
			"name":    {"form-name"},
			"message": {"form-message"},
		},
	}
	dst := newGreetingRequest(t)

	err = NewDefaultBinder().Bind(
		context.Background(),
		req,
		httpadapter.MethodSpec{},
		httpadapter.HTTPBinding{Body: "*"},
		dst,
	)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if got := stringField(dst, "name"); got != "body-name" {
		t.Fatalf("name = %q, want %q", got, "body-name")
	}
	if got := stringField(dst, "message"); got != "body-message" {
		t.Fatalf("message = %q, want %q", got, "body-message")
	}
}

func TestDefaultBinderBindsTemplatedPathParam(t *testing.T) {
	req := fakeRequest{
		httpRequest: httptestRequest(t, http.MethodGet, "/v1/shelves/s1/books/b1/pages/p1"),
		path: map[string]string{
			"name": "s1",
			"*":    "b1/pages/p1",
		},
	}
	dst := newGreetingRequest(t)

	err := NewDefaultBinder().Bind(
		context.Background(),
		req,
		httpadapter.MethodSpec{},
		httpadapter.HTTPBinding{
			PathParams: []httpadapter.PathParam{
				{
					Field:    "name",
					Template: "shelves/*/books/**",
					Names:    []string{"name", "*"},
				},
			},
		},
		dst,
	)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}
	if got := stringField(dst, "name"); got != "shelves/s1/books/b1/pages/p1" {
		t.Fatalf("name = %q, want %q", got, "shelves/s1/books/b1/pages/p1")
	}
}

func httptestRequest(t *testing.T, method, target string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func newGreetingRequest(t *testing.T) *dynamicpb.Message {
	t.Helper()

	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("greeting_request.proto"),
		Package: proto.String("testpb"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("GreetingRequest"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("name"),
						JsonName: proto.String("name"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:     proto.String("message"),
						JsonName: proto.String("message"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("build greeting request descriptor: %v", err)
	}
	return dynamicpb.NewMessage(file.Messages().ByName("GreetingRequest"))
}

func stringField(msg protoreflect.Message, name protoreflect.Name) string {
	field := msg.Descriptor().Fields().ByName(name)
	return msg.Get(field).String()
}
