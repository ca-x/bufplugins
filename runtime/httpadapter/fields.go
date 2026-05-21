package httpadapter

import "google.golang.org/protobuf/reflect/protoreflect"

func FindField(desc protoreflect.MessageDescriptor, name string) protoreflect.FieldDescriptor {
	fields := desc.Fields()
	if field := fields.ByName(protoreflect.Name(name)); field != nil {
		return field
	}
	return fields.ByJSONName(name)
}
