package v1

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestOrderContainerFieldNumbersRemainCompatible(t *testing.T) {
	tests := []struct {
		name     string
		message  protoreflect.MessageDescriptor
		expected map[protoreflect.Name]protoreflect.FieldNumber
	}{
		{
			name:    "OrderContainer",
			message: (&OrderContainer{}).ProtoReflect().Descriptor(),
			expected: map[protoreflect.Name]protoreflect.FieldNumber{
				"seal_no": 5, "gross_weight_kg": 6, "volume_cbm": 7, "note": 8,
				"created_at": 9, "updated_at": 10, "shipping_document_id": 11,
			},
		},
		{
			name:    "AddContainerRequest",
			message: (&AddContainerRequest{}).ProtoReflect().Descriptor(),
			expected: map[protoreflect.Name]protoreflect.FieldNumber{
				"seal_no": 4, "gross_weight_kg": 5, "volume_cbm": 6, "note": 7,
				"shipping_document_id": 8,
			},
		},
		{
			name:    "UpdateContainerRequest",
			message: (&UpdateContainerRequest{}).ProtoReflect().Descriptor(),
			expected: map[protoreflect.Name]protoreflect.FieldNumber{
				"seal_no": 5, "gross_weight_kg": 6, "volume_cbm": 7, "note": 8,
				"shipping_document_id": 9,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fields := test.message.Fields()
			for name, number := range test.expected {
				field := fields.ByName(name)
				if field == nil {
					t.Fatalf("字段 %s 不存在", name)
				}
				if field.Number() != number {
					t.Fatalf("字段 %s 的编号应为 %d，实际为 %d", name, number, field.Number())
				}
			}
		})
	}
}
