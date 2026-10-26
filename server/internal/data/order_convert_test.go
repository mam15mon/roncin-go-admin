package data

import (
	"slices"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

func TestOrderToBizReviewStatusFollowsManualOrderLock(t *testing.T) {
	lockedAt := time.Now().UTC()
	for _, tc := range []struct {
		name       string
		lockedAt   *time.Time
		oldStatus  orderent.FeeReviewStatus
		wantStatus string
	}{
		{"锁定订单不读取旧未提交状态", &lockedAt, orderent.FeeReviewStatusUNSUBMITTED, "APPROVED"},
		{"已解锁订单不读取历史批准状态", nil, orderent.FeeReviewStatusAPPROVED, "UNSUBMITTED"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			order := orderToBiz(&ent.Order{
				LockedAt:        tc.lockedAt,
				FeeReviewStatus: tc.oldStatus,
				Edges:           ent.OrderEdges{Organization: &ent.Organization{}},
			})
			if order.FeeReview.Status != tc.wantStatus {
				t.Fatalf("审核状态=%s，期望 %s", order.FeeReview.Status, tc.wantStatus)
			}
		})
	}
}

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
		for _, closureStatus := range []biz.OrderClosureStatus{biz.OrderClosureOpen, biz.OrderClosureClosed} {
			t.Run(string(flowStatus)+"/"+string(closureStatus)+" 未终止且未锁定时开放编辑", func(t *testing.T) {
				actions := orderAllowedActions(&biz.Order{
					FlowStatus:        flowStatus,
					TerminationStatus: biz.OrderTerminationActive,
					ClosureStatus:     closureStatus,
				})
				if !slices.Contains(actions, biz.OrderActionEdit) {
					t.Fatalf("flow_status=%s closure_status=%s 的可编辑订单未投影 EDIT: %v", flowStatus, closureStatus, actions)
				}
			})
		}
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
			name: "完结并审核锁定",
			order: &biz.Order{
				FlowStatus:        biz.OrderFlowDocumentReleased,
				TerminationStatus: biz.OrderTerminationActive,
				ClosureStatus:     biz.OrderClosureClosed,
				LockedAt:          &lockedAt,
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

func TestOrderToBizPreservesVoyageScheduleInstants(t *testing.T) {
	etd := time.Date(2026, time.September, 24, 16, 0, 0, 0, time.UTC)
	eta := time.Date(2026, time.October, 9, 16, 0, 0, 0, time.UTC)
	shanghaiOffset := time.FixedZone("UTC+8", 8*60*60)
	localETD, localETA := etd.In(shanghaiOffset), eta.In(shanghaiOffset)
	for _, testCase := range []struct {
		name       string
		execution  *ent.SeaTransportExecution
		orderETD   string
		orderETA   string
		wantETD    string
		wantETA    string
		summaryETD string
		summaryETA string
	}{
		{
			name:      "数据库UTC时间保留本地零点对应的真实时刻",
			execution: &ent.SeaTransportExecution{Etd: &etd, Eta: &eta},
			orderETD:  "2026-09-01T00:00:00Z", orderETA: "2026-09-02T00:00:00Z",
			wantETD: "2026-09-24T16:00:00Z", wantETA: "2026-10-09T16:00:00Z",
			summaryETD: "2026-09-24T16:00:00Z", summaryETA: "2026-10-09T16:00:00Z",
		},
		{
			name:      "带时区偏移的运输执行保留偏移及时刻",
			execution: &ent.SeaTransportExecution{Etd: &localETD, Eta: &localETA},
			wantETD:   "2026-09-25T00:00:00+08:00", wantETA: "2026-10-10T00:00:00+08:00",
			summaryETD: "2026-09-25T00:00:00+08:00", summaryETA: "2026-10-10T00:00:00+08:00",
		},
		{
			name:     "无运输执行时保持订单原始船期",
			orderETD: "2026-09-25T00:00:00+08:00", orderETA: "2026-10-10T00:00:00+08:00",
			wantETD: "2026-09-25T00:00:00+08:00", wantETA: "2026-10-10T00:00:00+08:00",
		},
		{
			name:      "运输执行无ETD时不生成日期",
			execution: &ent.SeaTransportExecution{Eta: &eta},
			wantETA:   "2026-10-09T16:00:00Z", summaryETA: "2026-10-09T16:00:00Z",
		},
		{
			name:      "运输执行无船期时保持空值",
			execution: &ent.SeaTransportExecution{},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			order := &ent.Order{
				BusinessType: orderent.BusinessTypeSE,
				Etd:          testCase.orderETD, Eta: testCase.orderETA,
				Edges: ent.OrderEdges{
					Organization: &ent.Organization{},
					SeaMasterBillLinks: []*ent.SeaMasterBillOrderLink{{
						Edges: ent.SeaMasterBillOrderLinkEdges{
							MasterBill: &ent.SeaMasterBill{}, TransportExecution: testCase.execution,
						},
					}},
				},
			}
			result := orderToBiz(order)
			if result.SeaMasterBill == nil {
				t.Fatal("应保留主单摘要")
			}
			for _, field := range []struct {
				name string
				got  string
				want string
			}{
				{"订单ETD", result.ETD, testCase.wantETD},
				{"订单ETA", result.ETA, testCase.wantETA},
				{"摘要ETD", result.SeaMasterBill.ETD, testCase.summaryETD},
				{"摘要ETA", result.SeaMasterBill.ETA, testCase.summaryETA},
			} {
				if field.got != field.want {
					t.Fatalf("%s = %q，期望 %q", field.name, field.got, field.want)
				}
				if field.got != "" {
					parsed, err := time.Parse(time.RFC3339, field.got)
					if err != nil {
						t.Fatalf("%s 必须保留完整RFC3339时刻: %v", field.name, err)
					}
					if testCase.execution != nil {
						wantInstant := etd
						if field.name == "订单ETA" || field.name == "摘要ETA" {
							wantInstant = eta
						}
						if !parsed.Equal(wantInstant) {
							t.Fatalf("%s 丢失原始时刻: %v", field.name, parsed)
						}
					}
				}
			}
		})
	}
}

func TestOrderAllowedActionsCloseBeforeFlowCompleted(t *testing.T) {
	lockedAt := time.Now().UTC()
	for _, businessType := range []biz.OrderBusinessType{biz.OrderBusinessSE, biz.OrderBusinessAE} {
		for _, flow := range []biz.OrderFlowStatus{biz.OrderFlowDraft, biz.OrderFlowBooked, biz.OrderFlowSpaceAllocated, biz.OrderFlowTruckingArranged, biz.OrderFlowDocumentCutoff, biz.OrderFlowCustomsDeclarationArranged, biz.OrderFlowDocumentReleased} {
			for _, termination := range []biz.OrderTerminationStatus{biz.OrderTerminationActive, biz.OrderTerminationTerminating, biz.OrderTerminationTerminated} {
				t.Run(string(businessType)+"/"+string(flow)+"/"+string(termination), func(t *testing.T) {
					for _, lock := range []*time.Time{nil, &lockedAt} {
						order := &biz.Order{BusinessType: businessType, FlowStatus: flow, TerminationStatus: termination, ClosureStatus: biz.OrderClosureOpen, LockedAt: lock}
						if !slices.Contains(orderAllowedActions(order), biz.OrderActionClose) {
							t.Fatal("未完结且无活动异常的订单应开放完结，不受主流程或业务锁限制")
						}
						order.HasActiveException = true
						if slices.Contains(orderAllowedActions(order), biz.OrderActionClose) {
							t.Fatal("活动异常仍应阻止完结")
						}
						order.ClosureStatus = biz.OrderClosureClosed
						expected := []biz.OrderAllowedAction{}
						if lock == nil {
							expected = append(expected, biz.OrderActionReopen)
						}
						if termination == biz.OrderTerminationActive && lock == nil {
							expected = append(expected, biz.OrderActionEdit)
						}
						if actions := orderAllowedActions(order); !slices.Equal(actions, expected) {
							t.Fatalf("已完结应保留反结案及未锁定的编辑资格: got=%v want=%v", actions, expected)
						}
					}
				})
			}
		}
	}
}
