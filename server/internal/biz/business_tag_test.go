package biz

import (
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeBusinessTagBatchDeduplicatesAndValidates(t *testing.T) {
	organizationID := uuid.New()
	orderID := uuid.New()
	tagID := uuid.New()

	orderIDs, tagIDs, err := normalizeBusinessTagBatch(organizationID, OrderBusinessSE, []uuid.UUID{orderID, orderID}, []uuid.UUID{tagID})
	if err != nil {
		t.Fatalf("normalizeBusinessTagBatch error = %v", err)
	}
	if len(orderIDs) != 1 || len(tagIDs) != 1 {
		t.Fatalf("去重后长度错误: orders=%d tags=%d", len(orderIDs), len(tagIDs))
	}

	if _, _, err := normalizeBusinessTagBatch(organizationID, OrderBusinessSE, nil, []uuid.UUID{tagID}); err == nil {
		t.Fatal("空订单列表应返回错误")
	}
	if _, _, err := normalizeBusinessTagBatch(organizationID, OrderBusinessSE, []uuid.UUID{orderID}, nil); err == nil {
		t.Fatal("空标签列表应返回错误")
	}
	if _, _, err := normalizeBusinessTagBatch(organizationID, OrderBusinessSE, []uuid.UUID{uuid.Nil}, []uuid.UUID{tagID}); err == nil {
		t.Fatal("零值订单 ID 应返回错误")
	}
	if _, _, err := normalizeBusinessTagBatch(organizationID, OrderBusinessType("XX"), []uuid.UUID{orderID}, []uuid.UUID{tagID}); err == nil {
		t.Fatal("非法业务类型应返回错误")
	}

	big := make([]uuid.UUID, 201)
	for i := range big {
		big[i] = uuid.New()
	}
	if _, _, err := normalizeBusinessTagBatch(organizationID, OrderBusinessSE, big, []uuid.UUID{tagID}); err == nil {
		t.Fatal("超过 200 个订单应返回错误")
	}
}

func TestBusinessTagAuditRecordsBatchDetails(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	tagID := uuid.New()

	audit := businessTagAudit(organizationID, actorID, "order.tag.batch_assign", []uuid.UUID{orderID}, []uuid.UUID{tagID})
	if audit.Action != "order.tag.batch_assign" {
		t.Fatalf("审计动作错误: %s", audit.Action)
	}
	if audit.ResourceID != "" {
		t.Fatalf("批量操作 ResourceID 应留空，实际为 %s", audit.ResourceID)
	}
	if audit.Details["order.ids"] != orderID.String() || audit.Details["tag.ids"] != tagID.String() {
		t.Fatalf("审计详情缺少批量 ID: %v", audit.Details)
	}
}
