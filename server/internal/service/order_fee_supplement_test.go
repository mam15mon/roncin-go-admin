package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestOrderFeeSupplementToAPIIncludesActorNames(t *testing.T) {
	decidedBy := uuid.New()
	decidedByName := "李审批"
	request := &biz.OrderFeeSupplementRequest{
		Fee:             biz.OrderFeeSupplementFeeSnapshot{SettlementPartyName: "示例供应商"},
		RequestedBy:     uuid.New(),
		RequestedByName: "张发起",
		RequestedAt:     time.Date(2026, 9, 21, 22, 44, 0, 0, time.UTC),
		DecidedBy:       &decidedBy,
		DecidedByName:   &decidedByName,
	}

	result := orderFeeSupplementToAPI(request)
	if result.SettlementPartyName != "示例供应商" {
		t.Fatalf("审核所需结算单位名称映射不符: %+v", result)
	}
	if result.RequestedByName != "张发起" || result.RequestedBy != request.RequestedBy.String() {
		t.Fatalf("发起人 ID 与姓名映射不符: %+v", result)
	}
	if result.DecidedByName == nil || *result.DecidedByName != decidedByName || result.DecidedBy == nil || *result.DecidedBy != decidedBy.String() {
		t.Fatalf("决策人 ID 与姓名映射不符: %+v", result)
	}

	request.DecidedBy = nil
	request.DecidedByName = nil
	pending := orderFeeSupplementToAPI(request)
	if pending.DecidedBy != nil || pending.DecidedByName != nil {
		t.Fatalf("待审批申请不应返回决策人: %+v", pending)
	}
}
