package data

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// adminRepo 按 admin_organization / admin_user / admin_user_membership /
// admin_role / admin_audit 五个文件按聚合拆分实现，本文件只保留仓储锚点。
type adminRepo struct{ data *Data }

func NewAdminRepo(data *Data) biz.AdminRepo { return &adminRepo{data: data} }

var _ biz.AdminRepo = (*adminRepo)(nil)
