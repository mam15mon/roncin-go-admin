package server

import (
	"testing"

	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func TestFinanceOrganizationOptionsRetirementContract(t *testing.T) {
	for _, name := range []protoreflect.FullName{"finance.v1.FinanceOrganizationOption", "finance.v1.ListFinanceOrganizationOptionsRequest", "finance.v1.ListFinanceOrganizationOptionsResponse"} {
		if _, err := protoregistry.GlobalFiles.FindDescriptorByName(name); err == nil {
			t.Errorf("退役公司候选描述符仍存在: %s", name)
		}
	}
	descriptor, err := protoregistry.GlobalFiles.FindDescriptorByName("finance.v1.SettlementService")
	if err != nil {
		t.Fatal(err)
	}
	methods := descriptor.(protoreflect.ServiceDescriptor).Methods()
	if methods.ByName("ListFinanceOrganizationOptions") != nil || methods.ByName("ListFinanceSettlementPartyOptions") == nil {
		t.Fatal("公司候选必须退役，结算单位候选必须保留")
	}
	purpose := financev1.FinanceOrganizationPurpose(0).Descriptor()
	retained := map[protoreflect.Name]protoreflect.EnumNumber{
		"FINANCE_ORGANIZATION_PURPOSE_UNSPECIFIED":         0,
		"FINANCE_ORGANIZATION_PURPOSE_CASHFLOW_CREATE":     4,
		"FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE": 9,
	}
	if purpose.Values().Len() != len(retained) {
		t.Fatalf("有效用途数量=%d", purpose.Values().Len())
	}
	for name, number := range retained {
		value := purpose.Values().ByName(name)
		if value == nil || value.Number() != number {
			t.Errorf("保留用途编号改变: %s=%v", name, value)
		}
	}
	for number, suffix := range map[protoreflect.EnumNumber]string{
		1: "BILL_READ", 2: "INVOICE_READ", 3: "CASHFLOW_READ", 5: "VERIFICATION_READ", 6: "COMMISSION_READ", 7: "COMMISSION_MANAGE", 8: "FEE_READ", 10: "INVOICE_CREATE", 11: "BILL_CREATE", 12: "NETTING_READ",
	} {
		name := protoreflect.Name("FINANCE_ORGANIZATION_PURPOSE_" + suffix)
		if purpose.Values().ByNumber(number) != nil || !purpose.ReservedRanges().Has(number) || !purpose.ReservedNames().Has(name) {
			t.Errorf("退役用途必须reserved原编号及名称: %s=%d", name, number)
		}
	}
}
