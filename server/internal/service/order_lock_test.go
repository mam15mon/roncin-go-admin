package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type orderLockServiceRepoStub struct {
	lastPage     int
	lastPageSize int
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
