package data

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type commissionRepo struct{ data *Data }

func NewCommissionRepo(data *Data) biz.CommissionRepo { return &commissionRepo{data: data} }

var _ biz.CommissionRepo = (*commissionRepo)(nil)
