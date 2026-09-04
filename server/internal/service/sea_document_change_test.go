package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestSeaDocumentChangeDTOConversions(t *testing.T) {
	now := time.Date(2026, 9, 4, 16, 0, 0, 0, time.UTC)
	versionID := uuid.New()
	documentID := uuid.New()
	orderID := uuid.New()
	value := "不可变发货人"
	version := seaDocumentVersionToAPI(&biz.SeaDocumentVersion{
		ID: versionID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: documentID,
		OrderID: orderID, MasterBillID: uuid.New(), VersionNo: 2, SourceEntityVersion: 3,
		DocumentNo: "HBL-001", Status: "DRAFT", Source: biz.VersionSourceAmendment,
		Content: &biz.SeaBillContent{ShipperText: &value}, CreatedAt: now,
	})
	if version.GetId() != versionID.String() || version.GetDocumentType() != v1.SeaDocumentType_SEA_DOCUMENT_TYPE_HOUSE_BILL || version.GetSource() != v1.SeaDocumentVersionSource_SEA_DOCUMENT_VERSION_SOURCE_AMENDMENT || version.GetContent().GetShipperText() != value {
		t.Fatalf("不可变版本 DTO 映射错误: %+v", version)
	}

	oldID, newID, chainID := uuid.New(), uuid.New(), uuid.New()
	sequence := 2
	event := seaDocumentEventToAPI(&biz.SeaDocumentEvent{
		ID: uuid.New(), EventType: biz.SeaDocumentEventTypeSwitch, DocumentType: biz.SeaDocumentTypeHouseBill,
		OldHouseBillID: &oldID, NewHouseBillID: &newID, ChainID: &chainID, Sequence: &sequence,
		Reason: "二次换单", CreatedAt: now,
	})
	if event.GetEventType() != v1.SeaDocumentEventType_SEA_DOCUMENT_EVENT_TYPE_SWITCH || event.GetOldHouseBillId() != oldID.String() || event.GetNewHouseBillId() != newID.String() || event.GetSequence() != 2 {
		t.Fatalf("Switch 事件 DTO 映射错误: %+v", event)
	}
}

func TestSeaDocumentChangePagination(t *testing.T) {
	if page, size, err := listDocumentPage(0, 0); err != nil || page != 1 || size != 20 {
		t.Fatalf("默认分页错误: page=%d size=%d err=%v", page, size, err)
	}
	for _, size := range []int32{1, 200} {
		if page, got, err := listDocumentPage(1, size); err != nil || page != 1 || got != int(size) {
			t.Fatalf("合法分页边界错误: input=%d page=%d size=%d err=%v", size, page, got, err)
		}
	}
	if _, _, err := listDocumentPage(1, 201); err == nil {
		t.Fatal("page_size 超过 200 应被拒绝")
	}
}
