package data

import (
	"slices"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestOrderAllowedActionsEditFollowsBusinessContentGate(t *testing.T) {
	allFlowStatuses := []biz.OrderFlowStatus{
		biz.OrderFlowDraft,
		biz.OrderFlowBooked,
		biz.OrderFlowSpaceAllocated,
		biz.OrderFlowTruckingArranged,
		biz.OrderFlowDocumentCutoff,
		biz.OrderFlowCustomsDeclarationArranged,
		biz.OrderFlowDocumentReleased,
	}
	for _, flowStatus := range allFlowStatuses {
		t.Run(string(flowStatus)+" active open unlocked exposes edit", func(t *testing.T) {
			actions := orderAllowedActions(&biz.Order{
				FlowStatus:        flowStatus,
				TerminationStatus: biz.OrderTerminationActive,
				ClosureStatus:     biz.OrderClosureOpen,
			})
			if !slices.Contains(actions, biz.OrderActionEdit) {
				t.Fatalf("flow_status=%s 的可编辑订单未投影 EDIT: %v", flowStatus, actions)
			}
		})
	}

	lockedAt := time.Now().UTC()
	for _, tc := range []struct {
		name  string
		order *biz.Order
	}{
		{
			name: "business locked",
			order: &biz.Order{
				FlowStatus:        biz.OrderFlowBooked,
				TerminationStatus: biz.OrderTerminationActive,
				ClosureStatus:     biz.OrderClosureOpen,
				LockedAt:          &lockedAt,
			},
		},
		{
			name: "terminating",
			order: &biz.Order{
				FlowStatus:        biz.OrderFlowBooked,
				TerminationStatus: biz.OrderTerminationTerminating,
				ClosureStatus:     biz.OrderClosureOpen,
			},
		},
		{
			name: "terminated",
			order: &biz.Order{
				FlowStatus:        biz.OrderFlowBooked,
				TerminationStatus: biz.OrderTerminationTerminated,
				ClosureStatus:     biz.OrderClosureOpen,
			},
		},
		{
			name: "closed",
			order: &biz.Order{
				FlowStatus:        biz.OrderFlowDocumentReleased,
				TerminationStatus: biz.OrderTerminationActive,
				ClosureStatus:     biz.OrderClosureClosed,
			},
		},
	} {
		t.Run(tc.name+" hides edit", func(t *testing.T) {
			actions := orderAllowedActions(tc.order)
			if slices.Contains(actions, biz.OrderActionEdit) {
				t.Fatalf("不可编辑订单不应投影 EDIT: %v", actions)
			}
		})
	}
}
