package biz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

type orderLockRepoStub struct {
	state          *OrderLockState
	lockResult     *OrderLockResult
	unlockResult   *OrderUnlockResult
	unlockRequests []*OrderUnlockRequest
	totalRequests  int
	unlockRequest  *OrderUnlockRequest

	lastLockIdempotencyKey   string
	lastUnlockIdempotencyKey string
	lastUnlockReason         *string
	lastPage                 int
	lastPageSize             int
}

func (s *orderLockRepoStub) GetOrderLockState(ctx context.Context, organizationID, orderID uuid.UUID, caller *Principal) (*OrderLockState, error) {
	return s.state, nil
}

func (s *orderLockRepoStub) LockOrder(ctx context.Context, caller *Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, audit *AuditEvent) (*OrderLockResult, error) {
	s.lastLockIdempotencyKey = idempotencyKey
	return s.lockResult, nil
}

func (s *orderLockRepoStub) RequestOrderUnlock(ctx context.Context, caller *Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, reason *string, audit *AuditEvent) (*OrderUnlockResult, error) {
	s.lastUnlockIdempotencyKey = idempotencyKey
	s.lastUnlockReason = reason
	return s.unlockResult, nil
}

func (s *orderLockRepoStub) ListOrderUnlockRequests(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int) ([]*OrderUnlockRequest, int, error) {
	s.lastPage = page
	s.lastPageSize = pageSize
	return s.unlockRequests, s.totalRequests, nil
}

func (s *orderLockRepoStub) GetOrderUnlockRequest(ctx context.Context, organizationID, orderID, requestID uuid.UUID) (*OrderUnlockRequest, error) {
	return s.unlockRequest, nil
}

func TestNewErrOrderBusinessLocked(t *testing.T) {
	orderID := uuid.New()
	lockedAt := time.Date(2026, 9, 4, 12, 0, 0, 0, time.UTC)
	err := NewErrOrderBusinessLocked(orderID, "SO2609040001", 3, lockedAt, "张三")

	kErr := errors.FromError(err)
	if kErr == nil {
		t.Fatalf("expected kratos error, got %v", err)
	}
	if kErr.Reason != "ORDER_BUSINESS_LOCKED" {
		t.Errorf("expected reason ORDER_BUSINESS_LOCKED, got %s", kErr.Reason)
	}
	if kErr.Code != 409 {
		t.Errorf("expected status code 409, got %d", kErr.Code)
	}
	if kErr.Metadata["order_id"] != orderID.String() {
		t.Errorf("expected order_id %s, got %s", orderID.String(), kErr.Metadata["order_id"])
	}
	if kErr.Metadata["order_no"] != "SO2609040001" {
		t.Errorf("expected order_no SO2609040001, got %s", kErr.Metadata["order_no"])
	}
	if kErr.Metadata["lock_generation"] != "3" {
		t.Errorf("expected lock_generation 3, got %s", kErr.Metadata["lock_generation"])
	}
	if kErr.Metadata["locked_by_name"] != "张三" {
		t.Errorf("expected locked_by_name 张三, got %s", kErr.Metadata["locked_by_name"])
	}
}

func TestNewErrSeaMasterBillMemberOrderLocked(t *testing.T) {
	err := NewErrSeaMasterBillMemberOrderLocked(2, []string{"SO2609040001", "SO2609040002"})

	kErr := errors.FromError(err)
	if kErr == nil {
		t.Fatalf("expected kratos error, got %v", err)
	}
	if kErr.Reason != "SEA_MASTER_BILL_MEMBER_ORDER_LOCKED" {
		t.Errorf("expected reason SEA_MASTER_BILL_MEMBER_ORDER_LOCKED, got %s", kErr.Reason)
	}
	if kErr.Code != 409 {
		t.Errorf("expected status code 409, got %d", kErr.Code)
	}
	if kErr.Metadata["locked_count"] != "2" {
		t.Errorf("expected locked_count 2, got %s", kErr.Metadata["locked_count"])
	}
	if kErr.Metadata["locked_order_nos"] != "SO2609040001,SO2609040002" {
		t.Errorf("expected locked_order_nos SO2609040001,SO2609040002, got %s", kErr.Metadata["locked_order_nos"])
	}
}

func TestOrderLockUsecase_ValidationAndPagination(t *testing.T) {
	stub := &orderLockRepoStub{
		state:      &OrderLockState{IsLocked: true},
		lockResult: &OrderLockResult{},
	}
	uc := NewOrderLockUsecase(stub)

	ctx := context.Background()
	orderID := uuid.New()
	principal := &Principal{UserID: uuid.New(), Organization: Organization{ID: uuid.New()}}

	// 1. LockOrder requires idempotencyKey
	_, err := uc.LockOrder(ctx, principal, orderID, 1, "", nil)
	if err == nil {
		t.Error("expected error when idempotencyKey is empty")
	}

	// 2. LockOrder with idempotencyKey succeeds
	_, err = uc.LockOrder(ctx, principal, orderID, 1, "  test-key-1  ", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastLockIdempotencyKey != "test-key-1" {
		t.Errorf("expected idempotencyKey test-key-1, got %s", stub.lastLockIdempotencyKey)
	}

	// 3. RequestOrderUnlock requires idempotencyKey
	_, err = uc.RequestOrderUnlock(ctx, principal, orderID, 1, "", nil, nil)
	if err == nil {
		t.Error("expected error when idempotencyKey is empty")
	}

	// 4. RequestOrderUnlock with idempotencyKey succeeds
	reason := "  客户要求改单  "
	_, err = uc.RequestOrderUnlock(ctx, principal, orderID, 1, "  test-key-2  ", &reason, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stub.lastUnlockIdempotencyKey != "test-key-2" {
		t.Errorf("expected idempotencyKey test-key-2, got %s", stub.lastUnlockIdempotencyKey)
	}
	if stub.lastUnlockReason == nil || *stub.lastUnlockReason != "客户要求改单" {
		t.Fatalf("expected trimmed reason, got %#v", stub.lastUnlockReason)
	}

	// 5. 输入边界必须在领域层拒绝，不能依赖数据库字段长度或静默修正。
	invalidCalls := []func() error{
		func() error { _, err := uc.LockOrder(ctx, principal, orderID, 0, "key", nil); return err },
		func() error {
			_, err := uc.LockOrder(ctx, principal, orderID, 1, strings.Repeat("a", 129), nil)
			return err
		},
		func() error { _, err := uc.LockOrder(ctx, principal, orderID, 1, "bad\nkey", nil); return err },
		func() error { _, err := uc.LockOrder(ctx, nil, orderID, 1, "key", nil); return err },
		func() error { _, err := uc.LockOrder(ctx, principal, uuid.Nil, 1, "key", nil); return err },
		func() error {
			longReason := strings.Repeat("改", 501)
			_, err := uc.RequestOrderUnlock(ctx, principal, orderID, 1, "key", &longReason, nil)
			return err
		},
		func() error {
			badReason := "原因\n换行"
			_, err := uc.RequestOrderUnlock(ctx, principal, orderID, 1, "key", &badReason, nil)
			return err
		},
	}
	for i, call := range invalidCalls {
		if err := call(); err != ErrOrderInvalidArgument {
			t.Errorf("invalid call %d expected ErrOrderInvalidArgument, got %v", i, err)
		}
	}

	// 6. usecase 只接受已经归一化且满足公共上限的分页参数。
	for _, tc := range []struct {
		page, pageSize int
		wantErr        bool
	}{
		{page: 1, pageSize: 20},
		{page: 1, pageSize: MaxListPageSize},
		{page: 0, pageSize: 20, wantErr: true},
		{page: -1, pageSize: 20, wantErr: true},
		{page: 1, pageSize: 0, wantErr: true},
		{page: 1, pageSize: MaxListPageSize + 1, wantErr: true},
	} {
		_, _, err = uc.ListOrderUnlockRequests(ctx, principal.Organization.ID, orderID, tc.page, tc.pageSize)
		if tc.wantErr && err != ErrOrderInvalidArgument {
			t.Errorf("pagination (%d,%d) expected invalid argument, got %v", tc.page, tc.pageSize, err)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("pagination (%d,%d) unexpected error: %v", tc.page, tc.pageSize, err)
		}
	}
}
