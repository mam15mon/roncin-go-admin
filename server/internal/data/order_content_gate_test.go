package data

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

// TestEnsureOrderBusinessContentEditableGate 锁定统一内容写门禁的四类阻断、
// 校验顺序与正常放行。生命周期阻断场景不触碰锁定人查询，users 传 nil 即可。
func TestEnsureOrderBusinessContentEditableGate(t *testing.T) {
	lockedAt := time.Now().UTC()
	cases := []struct {
		name       string
		order      *ent.Order
		wantErr    bool
		wantCode   int32
		wantReason string
	}{
		{
			name:    "ACTIVE+OPEN+未锁定 放行",
			order:   &ent.Order{BusinessType: orderent.BusinessTypeSE, TerminationStatus: orderent.TerminationStatusACTIVE, ClosureStatus: orderent.ClosureStatusOPEN},
			wantErr: false,
		},
		{
			name:    "nil 订单放行",
			order:   nil,
			wantErr: false,
		},
		{
			name:       "非法业务类型拒绝",
			order:      &ent.Order{BusinessType: orderent.BusinessType("UNKNOWN"), TerminationStatus: orderent.TerminationStatusACTIVE, ClosureStatus: orderent.ClosureStatusOPEN},
			wantErr:    true,
			wantCode:   400,
			wantReason: "ORDER_BUSINESS_UNSUPPORTED",
		},
		{
			name:       "TERMINATING 返回终止进行中",
			order:      &ent.Order{BusinessType: orderent.BusinessTypeSE, TerminationStatus: orderent.TerminationStatusTERMINATING, ClosureStatus: orderent.ClosureStatusOPEN},
			wantErr:    true,
			wantCode:   409,
			wantReason: "ORDER_TERMINATION_IN_PROGRESS",
		},
		{
			name:       "TERMINATED 返回已终止",
			order:      &ent.Order{BusinessType: orderent.BusinessTypeSE, TerminationStatus: orderent.TerminationStatusTERMINATED, ClosureStatus: orderent.ClosureStatusOPEN},
			wantErr:    true,
			wantCode:   409,
			wantReason: "ORDER_TERMINATED",
		},
		{
			name:       "CLOSED 返回已结案",
			order:      &ent.Order{BusinessType: orderent.BusinessTypeSE, TerminationStatus: orderent.TerminationStatusACTIVE, ClosureStatus: orderent.ClosureStatusCLOSED},
			wantErr:    true,
			wantCode:   409,
			wantReason: "ORDER_CLOSED",
		},
		{
			name:       "TERMINATED 优先于结案状态",
			order:      &ent.Order{BusinessType: orderent.BusinessTypeSE, TerminationStatus: orderent.TerminationStatusTERMINATED, ClosureStatus: orderent.ClosureStatusCLOSED},
			wantErr:    true,
			wantCode:   409,
			wantReason: "ORDER_TERMINATED",
		},
		{
			name:       "TERMINATED 优先于历史业务锁",
			order:      &ent.Order{BusinessType: orderent.BusinessTypeSE, TerminationStatus: orderent.TerminationStatusTERMINATED, ClosureStatus: orderent.ClosureStatusOPEN, LockedAt: &lockedAt},
			wantErr:    true,
			wantCode:   409,
			wantReason: "ORDER_TERMINATED",
		},
		{
			name:       "ACTIVE+OPEN 已锁定返回业务锁错误",
			order:      &ent.Order{ID: uuid.Must(uuid.NewV7()), OrderNo: "SE-LOCK-1", BusinessType: orderent.BusinessTypeSE, TerminationStatus: orderent.TerminationStatusACTIVE, ClosureStatus: orderent.ClosureStatusOPEN, LockedAt: &lockedAt, LockGeneration: 2},
			wantErr:    true,
			wantCode:   409,
			wantReason: "ORDER_BUSINESS_LOCKED",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ensureOrderBusinessContentEditable(context.Background(), nil, tc.order)
			if !tc.wantErr {
				if err != nil {
					t.Fatalf("期望放行，得到 %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("期望阻断，实际放行")
			}
			kErr := errors.FromError(err)
			if kErr == nil {
				t.Fatalf("非领域错误: %v", err)
			}
			if kErr.Code != tc.wantCode {
				t.Fatalf("HTTP code = %d，期望 %d", kErr.Code, tc.wantCode)
			}
			if kErr.Reason != tc.wantReason {
				t.Fatalf("reason = %s，期望 %s", kErr.Reason, tc.wantReason)
			}
		})
	}
}
