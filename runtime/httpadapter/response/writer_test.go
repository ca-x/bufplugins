package response

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/ca-x/bufplugins/runtime/httpadapter"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type fakeResponse struct {
	header http.Header
	status int
	body   []byte
}

func (r *fakeResponse) Header() http.Header {
	if r.header == nil {
		r.header = make(http.Header)
	}
	return r.header
}

func (r *fakeResponse) JSONBlob(status int, body []byte) error {
	r.status = status
	r.body = append(r.body[:0], body...)
	return nil
}

func (r *fakeResponse) NoContent(status int) error {
	r.status = status
	r.body = nil
	return nil
}

func TestDefaultWriterWritesRepeatedResponseBody(t *testing.T) {
	src := newRepeatedResponse(t)
	resp := &fakeResponse{}

	err := NewDefaultWriter().Write(
		context.Background(),
		resp,
		httpadapter.MethodSpec{},
		httpadapter.HTTPBinding{ResponseBody: "results"},
		src,
	)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if resp.status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.status, http.StatusOK)
	}
	var got []string
	if err := json.Unmarshal(resp.body, &got); err != nil {
		t.Fatalf("response body %q is not a JSON string array: %v", string(resp.body), err)
	}
	want := []string{"one", "two"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func TestDefaultWriterWritesMapResponseBody(t *testing.T) {
	src := newMapResponse(t)
	resp := &fakeResponse{}

	err := NewDefaultWriter().Write(
		context.Background(),
		resp,
		httpadapter.MethodSpec{},
		httpadapter.HTTPBinding{ResponseBody: "labels"},
		src,
	)
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if resp.status != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.status, http.StatusOK)
	}
	var got map[string]int32
	if err := json.Unmarshal(resp.body, &got); err != nil {
		t.Fatalf("response body %q is not a JSON int32 map: %v", string(resp.body), err)
	}
	want := map[string]int32{"one": 1, "two": 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("body = %#v, want %#v", got, want)
	}
}

func newMapResponse(t *testing.T) proto.Message {
	t.Helper()

	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("map_response.proto"),
		Package: proto.String("testpb"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("MapResponse"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("labels"),
						JsonName: proto.String("labels"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".testpb.MapResponse.LabelsEntry"),
					},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name: proto.String("LabelsEntry"),
						Field: []*descriptorpb.FieldDescriptorProto{
							{
								Name:     proto.String("key"),
								JsonName: proto.String("key"),
								Number:   proto.Int32(1),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
							},
							{
								Name:     proto.String("value"),
								JsonName: proto.String("value"),
								Number:   proto.Int32(2),
								Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
								Type:     descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
							},
						},
						Options: &descriptorpb.MessageOptions{MapEntry: proto.Bool(true)},
					},
				},
			},
		},
	}, nil)
	if err != nil {
		t.Fatalf("build map response descriptor: %v", err)
	}

	desc := file.Messages().ByName("MapResponse")
	msg := dynamicpb.NewMessage(desc)
	labels := desc.Fields().ByName("labels")
	values := msg.Mutable(labels).Map()
	values.Set(protoreflect.ValueOfString("one").MapKey(), protoreflect.ValueOfInt32(1))
	values.Set(protoreflect.ValueOfString("two").MapKey(), protoreflect.ValueOfInt32(2))
	return msg
}

func newRepeatedResponse(t *testing.T) proto.Message {
	t.Helper()

	file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
		Syntax:  proto.String("proto3"),
		Name:    proto.String("repeated_response.proto"),
		Package: proto.String("testpb"),
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: proto.String("RepeatedResponse"),
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
		t.Fatalf("build repeated response descriptor: %v", err)
	}

	desc := file.Messages().ByName("RepeatedResponse")
	msg := dynamicpb.NewMessage(desc)
	results := desc.Fields().ByName("results")
	values := msg.Mutable(results).List()
	values.Append(protoreflect.ValueOfString("one"))
	values.Append(protoreflect.ValueOfString("two"))
	return msg
}
