package data

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type objectStorageDeleter struct {
	enterprise biz.EnterpriseImageStorage
	orders     biz.OrderAttachmentStorage
}

// NewObjectStorageDeleter 按明确的对象命名空间路由，不对未知对象执行删除。
func NewObjectStorageDeleter(enterprise biz.EnterpriseImageStorage, orders biz.OrderAttachmentStorage) biz.ObjectStorageDeleter {
	return &objectStorageDeleter{enterprise: enterprise, orders: orders}
}

func (s *objectStorageDeleter) Enabled() bool {
	return (s.enterprise != nil && s.enterprise.Enabled()) || (s.orders != nil && s.orders.Enabled())
}

func (s *objectStorageDeleter) Delete(ctx context.Context, organizationID uuid.UUID, key string) error {
	switch {
	case strings.HasPrefix(key, "orders/"+organizationID.String()+"/"):
		if s.orders == nil || !s.orders.Enabled() {
			return biz.ErrOrderAttachmentStorageUnavailable
		}
		return s.orders.Delete(ctx, organizationID, key)
	case strings.HasPrefix(key, "enterprise-resources/"+organizationID.String()+"/"):
		if s.enterprise == nil || !s.enterprise.Enabled() {
			return biz.ErrEnterpriseImageStorageUnavailable
		}
		return s.enterprise.Delete(ctx, organizationID, key)
	default:
		return biz.ErrOrderAttachmentInvalidArgument
	}
}
