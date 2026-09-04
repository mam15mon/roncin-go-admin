package data

import (
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestEnsureLockedMembersAllowSharedMasterBillUpdateSortsOrderNumbers(t *testing.T) {
	lockedAt := time.Now().UTC()
	lockContext := &seaMasterBillUpdateLockContext{
		memberOrders: []*ent.Order{
			{OrderNo: "SE-003", LockedAt: &lockedAt},
			{OrderNo: "SE-001", LockedAt: &lockedAt},
			{OrderNo: "SE-002"},
		},
	}

	err := ensureLockedMembersAllowSharedMasterBillUpdate(lockContext)
	kratosErr := errors.FromError(err)
	if kratosErr == nil || kratosErr.Reason != "SEA_MASTER_BILL_MEMBER_ORDER_LOCKED" {
		t.Fatalf("期望共享 MBL 成员锁定错误，得到 %v", err)
	}
	if got := kratosErr.Metadata["locked_count"]; got != "2" {
		t.Fatalf("locked_count = %q，期望 2", got)
	}
	if got := kratosErr.Metadata["locked_order_nos"]; got != "SE-001,SE-003" {
		t.Fatalf("locked_order_nos = %q，期望按订单号稳定排序", got)
	}
}
