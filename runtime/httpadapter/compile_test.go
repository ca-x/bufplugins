package httpadapter

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestCompileServiceSpecPrecompilesBindingFields(t *testing.T) {
	reqDesc, respDesc := compileTestDescriptors(t)

	spec, err := CompileServiceSpec(ServiceSpec{
		Methods: []MethodSpec{
			{
				Procedure:       "/test.Service/Get",
				RequestFactory:  func() proto.Message { return dynamicpb.NewMessage(reqDesc) },
				ResponseFactory: func() proto.Message { return dynamicpb.NewMessage(respDesc) },
				HTTPBindings: []HTTPBinding{
					{
						Method:       "GET",
						Path:         "/v1/:name",
						Body:         "child",
						ResponseBody: "results",
						PathParams: []PathParam{
							{Name: "name", Field: "name"},
							{Field: "resource", Template: "shelves/*/books/**", Names: []string{"shelf", "*"}},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CompileServiceSpec() error = %v", err)
	}

	binding := spec.Methods[0].HTTPBindings[0]
	if len(binding.BodyFieldPath) != 1 || binding.BodyFieldPath[0].Name() != "child" {
		t.Fatalf("BodyFieldPath = %#v, want child descriptor", binding.BodyFieldPath)
	}
	if len(binding.ResponseBodyFieldPath) != 1 || binding.ResponseBodyFieldPath[0].Name() != "results" {
		t.Fatalf("ResponseBodyFieldPath = %#v, want results descriptor", binding.ResponseBodyFieldPath)
	}
	if len(binding.PathParams[0].FieldPath) != 1 || binding.PathParams[0].FieldPath[0].Name() != "name" {
		t.Fatalf("PathParams[0].FieldPath = %#v, want name descriptor", binding.PathParams[0].FieldPath)
	}
	if got := binding.PathParams[1].TemplateParts; len(got) != 4 || got[0] != "shelves" || got[1] != "*" || got[2] != "books" || got[3] != "**" {
		t.Fatalf("TemplateParts = %#v, want split template", got)
	}
}

func TestCompileServiceSpecRejectsInvalidField(t *testing.T) {
	reqDesc, _ := compileTestDescriptors(t)

	_, err := CompileServiceSpec(ServiceSpec{
		Methods: []MethodSpec{
			{
				Procedure:      "/test.Service/Get",
				RequestFactory: func() proto.Message { return dynamicpb.NewMessage(reqDesc) },
				HTTPBindings: []HTTPBinding{
					{
						Method: "GET",
						Path:   "/v1/:missing",
						PathParams: []PathParam{
							{Name: "missing", Field: "missing"},
						},
					},
				},
			},
		},
	})
	if err == nil {
		t.Fatal("CompileServiceSpec() error = nil, want invalid field error")
	}
	if !strings.Contains(err.Error(), `field "missing" not found`) {
		t.Fatalf("CompileServiceSpec() error = %v, want missing field error", err)
	}
}

func compileTestDescriptors(t *testing.T) (reqDesc, respDesc protoreflect.MessageDescriptor) {
	t.Helper()

	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("compile_test.proto"),
		Package: proto.String("testpb"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("Child"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("value"),
						JsonName: proto.String("value"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
			{
				Name: proto.String("Request"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("name"),
						JsonName: proto.String("name"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:     proto.String("resource"),
						JsonName: proto.String("resource"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
					{
						Name:     proto.String("child"),
						JsonName: proto.String("child"),
						Number:   proto.Int32(3),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".testpb.Child"),
					},
				},
			},
			{
				Name: proto.String("Response"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("results"),
						JsonName: proto.String("results"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
					},
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("build descriptor: %v", err)
	}
	return file.Messages().ByName("Request"), file.Messages().ByName("Response")
}
