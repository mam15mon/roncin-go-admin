package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

// ensureOperatingCompany 在持久化边界守住经营数据归属：客户、订单等经营根对象
// 只能写入启用公司。系统管理、部门、团队或无效组织均显式拒绝。
func ensureOperatingCompany(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID) error {
	exists, err := tx.Organization.Query().Where(
		organizationent.IDEQ(organizationID),
		organizationent.KindEQ(organizationent.KindCompany),
		organizationent.EnabledEQ(true),
	).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOperatingCompanyRequired
	}
	return nil
}
