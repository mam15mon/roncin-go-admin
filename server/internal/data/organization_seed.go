package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

// defaultBranchCompanySeeds 是建库种子中的默认分公司清单，各公司为独立根节点。
type branchCompanySeed struct {
	Code string
	Name string
}

var defaultBranchCompanySeeds = []branchCompanySeed{
	{Code: "BJ", Name: "融迅（北京）供应链管理有限公司"},
	{Code: "CD", Name: "融迅（成都）供应链管理有限公司"},
	{Code: "SH", Name: "融迅（上海）供应链管理有限公司"},
	{Code: "CQ", Name: "融迅（重庆）供应链管理有限公司"},
}

// CreateDefaultBranchCompanies 幂等补建默认公司种子：编码已存在的
// 分支跳过不覆盖；新建分支与后台手工创建的分公司保持同构（本币 CNY、默认
// 启用币种与全套默认单号规则）。
func CreateDefaultBranchCompanies(ctx context.Context, tx *ent.Tx, _ uuid.UUID) (int, error) {
	created := 0
	for _, seed := range defaultBranchCompanySeeds {
		exists, err := tx.Organization.Query().Where(organizationent.CodeEQ(seed.Code)).Exist(ctx)
		if err != nil {
			return created, fmt.Errorf("检查分公司种子 %s: %w", seed.Code, err)
		}
		if exists {
			continue
		}
		organization, err := tx.Organization.Create().
			SetCode(seed.Code).
			SetName(seed.Name).
			SetKind(organizationent.KindCompany).
			SetBaseCurrency("CNY").
			SetEnabledCurrencies(defaultCompanyEnabledCurrencies("CNY")).
			Save(ctx)
		if err != nil {
			return created, fmt.Errorf("创建分公司种子 %s: %w", seed.Code, err)
		}
		if err := CreateDefaultNumberRules(ctx, tx, organization.ID); err != nil {
			return created, err
		}
		if err := initializeCompanyAdministrator(ctx, tx.Client(), organization.ID); err != nil {
			return created, err
		}
		if err := initializeCompanyFeeSettings(ctx, tx.Client(), organization.ID); err != nil {
			return created, err
		}
		created++
	}
	return created, nil
}
