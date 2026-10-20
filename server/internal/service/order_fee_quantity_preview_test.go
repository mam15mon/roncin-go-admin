package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

type feeQuantityPreviewServiceRepo struct {
	biz.OrderFeeRepo
	organizationID, orderID, unitID uuid.UUID
	calls                           int
}

func (r *feeQuantityPreviewServiceRepo) QuantitySource(_ context.Context, organizationID, orderID, unitID uuid.UUID, source string) (*biz.FeeQuantitySource, error) {
	r.organizationID, r.orderID, r.unitID = organizationID, orderID, unitID
	r.calls++
	return &biz.FeeQuantitySource{UnitCode: "RT", UnitEnabled: true, CargoCount: 2, GrossWeightKG: decimal.NewFromInt(2500), VolumeCBM: decimal.NewFromInt(8)}, nil
}
func TestPreviewFeeQuantityService(t *testing.T) {
	repo := &feeQuantityPreviewServiceRepo{}
	s := NewOrderFeeService(biz.NewOrderFeeUsecase(repo, nil, nil, nil, nil, nil), nil, nil, nil, nil)
	orderID, unitID, organizationID := uuid.New(), uuid.New(), uuid.New()
	request := &v1.PreviewFeeQuantityRequest{OrderId: orderID.String(), BillingUnitId: unitID.String(), Source: "ACTUAL"}
	if _, err := s.PreviewFeeQuantity(context.Background(), request); err == nil {
		t.Fatal("未登录必须拒绝")
	}
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{UserID: uuid.New(), Organization: biz.Organization{ID: organizationID, Kind: biz.OrganizationKindCompany}})
	for _, invalid := range []*v1.PreviewFeeQuantityRequest{
		{OrderId: "invalid", BillingUnitId: unitID.String(), Source: "ACTUAL"},
		{OrderId: orderID.String(), BillingUnitId: "invalid", Source: "ACTUAL"},
		{OrderId: orderID.String(), BillingUnitId: unitID.String(), Source: "INVALID"},
	} {
		if _, err := s.PreviewFeeQuantity(ctx, invalid); err != biz.ErrOrderFeeInvalidArgument {
			t.Fatalf("无效请求=%+v：%v", invalid, err)
		}
	}
	if repo.calls != 0 {
		t.Fatal("无效请求不应访问仓储")
	}
	got, err := s.PreviewFeeQuantity(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if got.Data == nil || !got.Data.Available || got.Data.Quantity != "8" || len(got.Data.Inputs) != 2 || got.Data.Formula == "" || got.Data.SourceLabel == "" {
		t.Fatalf("缺少预览或计算依据：%+v", got)
	}
	if repo.organizationID != organizationID || repo.orderID != orderID || repo.unitID != unitID {
		t.Fatalf("请求范围未正确传递：%+v", repo)
	}
}
