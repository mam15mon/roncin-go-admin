package service

import (
	"reflect"
	"testing"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestSeaHouseBillPrintDetailsBoundaryRoundTrip(t *testing.T) {
	text := "上海 / SHANGHAI"
	count := int32(0)
	input := &v1.SeaHouseBillPrintDetails{HeaderContactText: &text, BookingNoText: &text, ExportReferencesText: &text, FooterReferenceText: &text, PreCarriageText: &text, PlaceOfReceiptText: &text, DomesticRoutingText: &text, PointOfOriginText: &text, OnwardRoutingText: &text, VesselVoyageText: &text, PortOfLoadingText: &text, PortOfDischargeText: &text, PlaceOfDeliveryText: &text, ContainersText: &text, OriginalCount: &count, ShippedOnBoardDate: &text, IssuePlaceText: &text, IssueDate: &text, DeliveryContactText: &text}
	domain := seaHouseBillPrintDetailsFromAPI(input)
	if output := seaHouseBillPrintDetailsToAPI(domain); !reflect.DeepEqual(input, output) {
		t.Fatalf("印字跨层转换丢失字段: %#v", output)
	}
	house := seaHouseBillToAPI(&biz.SeaHouseBill{PrintDetails: domain, PrintTemplate: biz.SeaHouseBillPrintTemplateRoncinSelf})
	if house.GetPrintTemplate() != "RONCIN_SELF" || !reflect.DeepEqual(house.PrintDetails, input) {
		t.Fatal("当前 HBL 未返回印字及模板")
	}
	version := seaDocumentVersionToAPI(&biz.SeaDocumentVersion{PrintDetails: domain})
	if !reflect.DeepEqual(version.PrintDetails, input) {
		t.Fatal("不可变版本未返回印字")
	}
	parsed, err := seaHouseBillInputFromAPI(&v1.SeaHouseBillInput{PrintDetails: input})
	if err != nil || !reflect.DeepEqual(parsed.PrintDetails, domain) {
		t.Fatalf("HBL 请求未接收印字: %v", err)
	}
	if seaHouseBillPrintDetailsFromAPI(nil) != nil || seaHouseBillPrintDetailsToAPI(nil) != nil {
		t.Fatal("空印字未保留空语义")
	}
}
