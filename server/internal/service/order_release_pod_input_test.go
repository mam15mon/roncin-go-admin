package service

import (
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestOrderReleasePodInputFromAPIRejectsInvalidDocumentReferences(t *testing.T) {
	orderID := uuid.NewString()
	seaID := uuid.NewString()
	legacyID := uuid.NewString()
	tests := []struct {
		name             string
		legacyDocumentID string
		seaDocumentType  v1.SeaDocumentType
		seaDocumentID    string
	}{
		{name: "旧单证UUID非法", legacyDocumentID: "not-a-uuid"},
		{name: "海运单证UUID非法", seaDocumentType: v1.SeaDocumentType_SEA_DOCUMENT_TYPE_MASTER_BILL, seaDocumentID: "not-a-uuid"},
		{name: "海运单证类型未知", seaDocumentType: v1.SeaDocumentType(99), seaDocumentID: seaID},
		{name: "海运单证缺少ID", seaDocumentType: v1.SeaDocumentType_SEA_DOCUMENT_TYPE_HOUSE_BILL},
		{name: "海运单证缺少类型", seaDocumentID: seaID},
		{name: "混合旧单证与海运单证", legacyDocumentID: legacyID, seaDocumentType: v1.SeaDocumentType_SEA_DOCUMENT_TYPE_HOUSE_BILL, seaDocumentID: seaID},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, _, err := orderReleasePodInputFromAPI(
				orderID,
				test.legacyDocumentID,
				test.seaDocumentType,
				test.seaDocumentID,
				"",
				"",
				"",
			)
			if err != biz.ErrOrderReleasePodDocumentInvalid {
				t.Fatalf("错误 = %v，期望 %v", err, biz.ErrOrderReleasePodDocumentInvalid)
			}
		})
	}
}

func TestOrderReleasePodInputFromAPIMapsExplicitDocumentReferences(t *testing.T) {
	orderID := uuid.New()
	documentID := uuid.New()
	tests := []struct {
		name              string
		legacyDocumentID  string
		seaDocumentType   v1.SeaDocumentType
		seaDocumentID     string
		wantLegacy        bool
		wantSeaType       biz.SeaDocumentType
		wantSeaDocumentID bool
	}{
		{name: "无关联"},
		{name: "旧单证", legacyDocumentID: documentID.String(), wantLegacy: true},
		{name: "MBL", seaDocumentType: v1.SeaDocumentType_SEA_DOCUMENT_TYPE_MASTER_BILL, seaDocumentID: documentID.String(), wantSeaType: biz.SeaDocumentTypeMasterBill, wantSeaDocumentID: true},
		{name: "HBL", seaDocumentType: v1.SeaDocumentType_SEA_DOCUMENT_TYPE_HOUSE_BILL, seaDocumentID: documentID.String(), wantSeaType: biz.SeaDocumentTypeHouseBill, wantSeaDocumentID: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotOrderID, input, err := orderReleasePodInputFromAPI(
				orderID.String(),
				test.legacyDocumentID,
				test.seaDocumentType,
				test.seaDocumentID,
				"",
				"",
				"",
			)
			if err != nil {
				t.Fatalf("转换失败: %v", err)
			}
			if gotOrderID != orderID {
				t.Fatalf("订单 ID = %s，期望 %s", gotOrderID, orderID)
			}
			if (input.ShippingDocumentID != nil) != test.wantLegacy {
				t.Fatalf("旧单证引用 = %v，期望存在=%v", input.ShippingDocumentID, test.wantLegacy)
			}
			if input.SeaDocumentType != test.wantSeaType {
				t.Fatalf("海运单证类型 = %q，期望 %q", input.SeaDocumentType, test.wantSeaType)
			}
			if (input.SeaDocumentID != nil) != test.wantSeaDocumentID {
				t.Fatalf("海运单证 ID = %v，期望存在=%v", input.SeaDocumentID, test.wantSeaDocumentID)
			}
		})
	}
}
