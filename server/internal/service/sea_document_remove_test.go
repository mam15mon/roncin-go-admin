package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestRemoveSeaHouseBillCascadeRequiresReleasePodDeletePermission(t *testing.T) {
	organizationID := uuid.New()
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{
		UserID:       uuid.New(),
		Organization: biz.Organization{ID: organizationID},
	})
	service := NewSeaDocumentService(nil, nil)
	_, err := service.RemoveSeaHouseBill(ctx, &v1.RemoveSeaHouseBillRequest{
		OrderId:                  uuid.NewString(),
		Id:                       uuid.NewString(),
		ExpectedVersion:          1,
		ExpectedLinkVersion:      1,
		RemoveRelatedReleasePods: func() *bool { value := true; return &value }(),
	})
	if err != biz.ErrPermissionDenied {
		t.Fatalf("关联删除缺少放货记录删除权限时错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
}
