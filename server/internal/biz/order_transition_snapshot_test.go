package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestOrderTransitionsIncludeSeaSummaries(t *testing.T) {
	ctx := context.Background()
	org, actor, id := uuid.New(), uuid.New(), uuid.New()
	master := &SeaMasterBillSummary{Version: 3}
	document := &SeaOrderDocumentSummary{DocumentStructure: SeaDocumentStructureDirect, LinkVersion: 2}
	repo := &orderRepoStub{current: &Order{ID: id, BusinessType: OrderBusinessSE, Version: 4, FlowStatus: OrderFlowBooked, TerminationStatus: OrderTerminationActive, ClosureStatus: OrderClosureOpen}, closureReadiness: &OrderClosureReadiness{ClosureStatus: OrderClosureOpen}}
	uc := NewOrderUsecase(repo, &seaMasterBillRepoStub{summaries: map[uuid.UUID]*SeaMasterBillSummary{id: master}}, &seaDocumentRepoMock{getSummariesByOrderIDsFunc: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]*SeaOrderDocumentSummary, error) {
		return map[uuid.UUID]*SeaOrderDocumentSummary{id: document}, nil
	}}, nil, nil)
	terminationType := OrderTerminationCustomsReturn
	for _, tc := range []struct {
		name string
		run  func() (*Order, error)
	}{
		{"主流程", func() (*Order, error) {
			return uc.TransitionStatus(ctx, org, actor, id, 4, OrderFlowSpaceAllocated, "")
		}},
		{"退关", func() (*Order, error) {
			return uc.TransitionTermination(ctx, org, actor, id, 4, OrderTerminationTerminating, &terminationType, "客户要求")
		}},
		{"完结", func() (*Order, error) { return uc.TransitionClosure(ctx, org, actor, id, 4, OrderClosureClosed, "") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			order, err := tc.run()
			if err != nil {
				t.Fatal(err)
			}
			if order.SeaMasterBill != master || order.SeaDocumentSummary != document || order.SeaDocumentLinkVersion == nil || *order.SeaDocumentLinkVersion != 2 || order.SeaDocumentStructure == nil || *order.SeaDocumentStructure != SeaDocumentStructureDirect {
				t.Fatalf("状态响应缺少详情关联摘要: %+v", order)
			}
		})
	}
}
