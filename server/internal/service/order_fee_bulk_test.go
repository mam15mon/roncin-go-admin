package service

import (
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func bulkTargetForTest(feeID string, version uint64) *v1.BulkOrderFeeTarget {
	return &v1.BulkOrderFeeTarget{FeeId: feeID, ExpectedVersion: version}
}

// TestParseOrderFeeBulkTargetsRejectsInvalidStructures 断言批量目标集合的结构
// 校验：非法订单/费用 UUID、空目标、重复费用与零版本一律拒绝。
func TestParseOrderFeeBulkTargetsRejectsInvalidStructures(t *testing.T) {
	validFeeID := uuid.NewString()
	tests := []struct {
		name    string
		orderID string
		targets []*v1.BulkOrderFeeTarget
	}{
		{name: "非法订单 UUID", orderID: "not-a-uuid", targets: []*v1.BulkOrderFeeTarget{bulkTargetForTest(validFeeID, 1)}},
		{name: "空目标", orderID: uuid.NewString()},
		{name: "非法费用 UUID", orderID: uuid.NewString(), targets: []*v1.BulkOrderFeeTarget{bulkTargetForTest("bad", 1)}},
		{name: "零版本", orderID: uuid.NewString(), targets: []*v1.BulkOrderFeeTarget{bulkTargetForTest(validFeeID, 0)}},
		{name: "重复费用", orderID: uuid.NewString(), targets: []*v1.BulkOrderFeeTarget{bulkTargetForTest(validFeeID, 1), bulkTargetForTest(validFeeID, 2)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := parseOrderFeeBulkTargets(test.orderID, test.targets); err != biz.ErrOrderFeeInvalidArgument {
				t.Fatalf("%s应被结构层拒绝，实际错误为 %v", test.name, err)
			}
		})
	}
}

// TestParseOrderFeeBulkTargetsAcceptsValidInput 断言合法目标集合原样转换。
func TestParseOrderFeeBulkTargetsAcceptsValidInput(t *testing.T) {
	orderID := uuid.Must(uuid.NewV7())
	feeID := uuid.Must(uuid.NewV7())
	orderIDText, feeIDText := orderID.String(), feeID.String()

	parsedOrderID, targets, err := parseOrderFeeBulkTargets(orderIDText, []*v1.BulkOrderFeeTarget{
		bulkTargetForTest(feeIDText, 3),
		bulkTargetForTest(uuid.NewString(), 1),
	})
	if err != nil {
		t.Fatalf("合法目标集合解析失败: %v", err)
	}
	if parsedOrderID != orderID || len(targets) != 2 {
		t.Fatalf("解析结果不正确: order=%s targets=%d", parsedOrderID, len(targets))
	}
	if targets[0].FeeID != feeID || targets[0].ExpectedVersion != 3 {
		t.Fatalf("目标字段转换不正确: %+v", targets[0])
	}
}

// TestParseOrderFeeBulkUpdatePartyRequiresSettlementParty 断言批量修改必须提供
// 合法的统一结算单位：缺失、非法 UUID 与空 UUID 均拒绝。
func TestParseOrderFeeBulkUpdatePartyRequiresSettlementParty(t *testing.T) {
	tests := []struct {
		name  string
		party *string
	}{
		{name: "缺失目标值"},
		{name: "非法结算单位 UUID", party: &[]string{"not-a-uuid"}[0]},
		{name: "空结算单位 UUID", party: &[]string{uuid.Nil.String()}[0]},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := parseOrderFeeBulkUpdateParty(test.party); err != biz.ErrOrderFeeInvalidArgument {
				t.Fatalf("%s应被结构层拒绝，实际错误为 %v", test.name, err)
			}
		})
	}

	party := uuid.NewString()
	parsedParty, err := parseOrderFeeBulkUpdateParty(&party)
	if err != nil || parsedParty == nil || parsedParty.String() != party {
		t.Fatalf("合法结算单位目标值解析不正确: party=%v err=%v", parsedParty, err)
	}
}
