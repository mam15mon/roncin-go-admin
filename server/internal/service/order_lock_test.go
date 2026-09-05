package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type orderLockServiceRepoStub struct {
	lastPage     int
	lastPageSize int
}

func TestOrderLockDTOIncludesBusinessTypeAndOptionalSeaReferences(t *testing.T) {
	lockedAt := time.Date(2026, time.September, 5, 8, 0, 0, 0, time.UTC)
	orderID := uuid.New()
	lockRecordID := uuid.New()
	nonSea := orderLockRecordToAPI(&biz.OrderLockRecord{
		ID: lockRecordID, OrderID: orderID, OrderNo: "AI-001", BusinessType: biz.OrderBusinessAI,
		Generation: 1, LockedBy: uuid.New(), LockedByName: "测试锁定人", LockedAt: lockedAt, OrderVersionAtLock: 2,
	})
	if nonSea.GetBusinessType() != v1.BusinessType_BUSINESS_TYPE_AI {
		t.Fatalf("非海运出口锁记录业务类型 = %s", nonSea.GetBusinessType())
	}
	if nonSea.MasterBillId != nil || nonSea.MasterBillVersionId != nil {
		t.Fatalf("非海运出口锁记录不得返回 MBL 引用: %#v", nonSea)
	}

	masterBillID, masterBillVersionID := uuid.New(), uuid.New()
	sea := orderLockRecordToAPI(&biz.OrderLockRecord{
		ID: uuid.New(), OrderID: orderID, OrderNo: "SE-001", BusinessType: biz.OrderBusinessSE,
		Generation: 1, LockedBy: uuid.New(), LockedByName: "测试锁定人", LockedAt: lockedAt, OrderVersionAtLock: 2,
		MasterBillID: &masterBillID, MasterBillVersionID: &masterBillVersionID,
	})
	if sea.GetBusinessType() != v1.BusinessType_BUSINESS_TYPE_SE || sea.GetMasterBillId() != masterBillID.String() || sea.GetMasterBillVersionId() != masterBillVersionID.String() {
		t.Fatalf("海运出口锁记录映射不完整: %#v", sea)
	}

	state := orderLockStateToAPI(&biz.OrderLockState{OrderID: orderID, OrderNo: "LAND-001", BusinessType: biz.OrderBusinessLand})
	if state.GetBusinessType() != v1.BusinessType_BUSINESS_TYPE_LAND {
		t.Fatalf("锁状态业务类型 = %s", state.GetBusinessType())
	}
	request := orderUnlockRequestToAPI(&biz.OrderUnlockRequest{ID: uuid.New(), OrderID: orderID, LockRecordID: lockRecordID, BusinessType: biz.OrderBusinessRail})
	if request.GetBusinessType() != v1.BusinessType_BUSINESS_TYPE_RAIL {
		t.Fatalf("解锁请求业务类型 = %s", request.GetBusinessType())
	}
}

func (*orderLockServiceRepoStub) GetOrderLockState(context.Context, uuid.UUID, uuid.UUID, *biz.Principal) (*biz.OrderLockState, error) {
	return nil, nil
}

func (*orderLockServiceRepoStub) LockOrder(context.Context, *biz.Principal, uuid.UUID, uint64, string, *biz.AuditEvent) (*biz.OrderLockResult, error) {
	return nil, nil
}

func (*orderLockServiceRepoStub) RequestOrderUnlock(context.Context, *biz.Principal, uuid.UUID, uint64, string, *string, *biz.AuditEvent) (*biz.OrderUnlockResult, error) {
	return nil, nil
}

func (s *orderLockServiceRepoStub) ListOrderUnlockRequests(_ context.Context, _, _ uuid.UUID, page, pageSize int) ([]*biz.OrderUnlockRequest, int, error) {
	s.lastPage = page
	s.lastPageSize = pageSize
	return nil, 0, nil
}

func (*orderLockServiceRepoStub) GetOrderUnlockRequest(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*biz.OrderUnlockRequest, error) {
	return nil, nil
}

func TestOrderLockServiceListPaginationBoundaries(t *testing.T) {
	principal := &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: uuid.New()}}
	ctx := biz.WithPrincipal(context.Background(), principal)
	orderID := uuid.New()
	repo := &orderLockServiceRepoStub{}
	svc := NewOrderLockService(biz.NewOrderLockUsecase(repo))

	for _, tc := range []struct {
		name               string
		page, pageSize     int32
		wantPage, wantSize int32
		wantErr            bool
	}{
		{name: "零值使用默认值", wantPage: 1, wantSize: 20},
		{name: "允许统一最大值", page: 1, pageSize: 200, wantPage: 1, wantSize: 200},
		{name: "拒绝负页码", page: -1, pageSize: 20, wantErr: true},
		{name: "拒绝超过统一上限", page: 1, pageSize: 201, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response, err := svc.ListOrderUnlockRequests(ctx, &v1.ListOrderUnlockRequestsRequest{
				OrderId:  orderID.String(),
				Page:     tc.page,
				PageSize: tc.pageSize,
			})
			if tc.wantErr {
				if err != biz.ErrOrderInvalidArgument {
					t.Fatalf("期望非法参数错误，实际为 %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("分页请求失败: %v", err)
			}
			if response.GetData().GetPage() != tc.wantPage || response.GetData().GetPageSize() != tc.wantSize {
				t.Fatalf("响应未回显归一化分页: page=%d page_size=%d", response.GetData().GetPage(), response.GetData().GetPageSize())
			}
			if repo.lastPage != int(tc.wantPage) || repo.lastPageSize != int(tc.wantSize) {
				t.Fatalf("仓储收到错误分页: page=%d page_size=%d", repo.lastPage, repo.lastPageSize)
			}
		})
	}
}
