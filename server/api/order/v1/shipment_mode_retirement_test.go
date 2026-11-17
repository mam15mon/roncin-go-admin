package v1

import (
	"testing"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func TestShipmentModeRetirementContract(t *testing.T) {
	for _, tc := range []struct {
		descriptor protoreflect.MessageDescriptor
		field      protoreflect.FieldNumber
	}{
		{(&Order{}).ProtoReflect().Descriptor(), 13},
		{(&CreateOrderRequest{}).ProtoReflect().Descriptor(), 11},
		{(&UpdateOrderRequest{}).ProtoReflect().Descriptor(), 12},
	} {
		t.Run(string(tc.descriptor.Name()), func(t *testing.T) {
			if tc.descriptor.Fields().ByName("shipment_mode") != nil || tc.descriptor.Fields().ByNumber(tc.field) != nil {
				t.Fatal("退役模式字段仍存在")
			}
			if !tc.descriptor.ReservedNames().Has("shipment_mode") || !tc.descriptor.ReservedRanges().Has(tc.field) {
				t.Fatal("退役字段名称与编号必须保留，防止复用")
			}
		})
	}
	if File_order_v1_order_proto.Enums().ByName("ShipmentMode") != nil {
		t.Fatal("退役模式枚举仍存在")
	}
	service := File_order_v1_order_proto.Services().ByName("OrderService")
	if service == nil {
		t.Fatal("普通订单服务不可删除")
	}
	if service.Methods().ByName("ListOrderConsolidations") != nil {
		t.Fatal("自拼汇总RPC仍存在")
	}
	for _, name := range []protoreflect.Name{"OrderCargoMeasurement", "OrderConsolidationMember", "OrderConsolidationSummary", "ListOrderConsolidationsRequest", "ListOrderConsolidationsResponse"} {
		if File_order_v1_order_proto.Messages().ByName(name) != nil {
			t.Fatalf("专用汇总消息%s仍存在", name)
		}
	}
	for _, name := range []protoreflect.Name{"GetOrder", "CreateOrder", "UpdateOrder", "ListSameBatchOrders"} {
		if service.Methods().ByName(name) == nil {
			t.Fatalf("相邻订单RPC%s不可删除", name)
		}
	}
}
