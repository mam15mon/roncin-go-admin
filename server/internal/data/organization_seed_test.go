package data

import (
	"context"
	"slices"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

func TestCreateDefaultBranchCompaniesSeedsAndIsIdempotent(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	headquarters, err := data.db.Organization.Create().
		SetCode("HQ").
		SetName("总部").
		SetKind("system").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建总部组织失败: %v", err)
	}

	seedBranches := func() int {
		t.Helper()
		created := 0
		if err := data.WithTx(ctx, func(tx *ent.Tx) error {
			var seedErr error
			created, seedErr = CreateDefaultBranchCompanies(ctx, tx, headquarters.ID)
			return seedErr
		}); err != nil {
			t.Fatalf("补建默认分公司种子失败: %v", err)
		}
		return created
	}

	if created := seedBranches(); created != len(defaultBranchCompanySeeds) {
		t.Fatalf("首次补建分公司数量 = %d, 期望 %d", created, len(defaultBranchCompanySeeds))
	}
	if created := seedBranches(); created != 0 {
		t.Fatalf("重复补建应跳过已存在编码，实际又创建 %d 个", created)
	}

	companies, err := data.db.Organization.Query().
		Where(organizationent.KindEQ(organizationent.KindCompany)).
		Order(organizationent.ByCode()).
		All(ctx)
	if err != nil {
		t.Fatalf("查询分公司失败: %v", err)
	}
	if len(companies) != len(defaultBranchCompanySeeds) {
		t.Fatalf("分公司数量 = %d, 期望 %d", len(companies), len(defaultBranchCompanySeeds))
	}
	expectedCurrencies := defaultCompanyEnabledCurrencies("CNY")
	for _, company := range companies {
		var seed branchCompanySeed
		found := false
		for _, candidate := range defaultBranchCompanySeeds {
			if candidate.Code == company.Code {
				seed, found = candidate, true
				break
			}
		}
		if !found {
			t.Fatalf("出现种子清单之外的分公司编码 %s", company.Code)
		}
		if company.Name != seed.Name {
			t.Fatalf("分公司 %s 名称 = %s, 期望 %s", company.Code, company.Name, seed.Name)
		}
		if company.ParentID != nil {
			t.Fatalf("分公司 %s 应为独立根节点: %v", company.Code, company.ParentID)
		}
		if company.BaseCurrency == nil || *company.BaseCurrency != "CNY" {
			t.Fatalf("分公司 %s 本币 = %v, 期望 CNY", company.Code, company.BaseCurrency)
		}
		if !slices.Equal(company.EnabledCurrencies, expectedCurrencies) {
			t.Fatalf("分公司 %s 启用币种 = %v, 期望 %v", company.Code, company.EnabledCurrencies, expectedCurrencies)
		}
		ruleCount, err := data.db.NumberRule.Query().
			Where(numberruleent.OrganizationIDEQ(company.ID)).
			Count(ctx)
		if err != nil {
			t.Fatalf("统计分公司 %s 单号规则失败: %v", company.Code, err)
		}
		if ruleCount != len(biz.DefaultNumberRules()) {
			t.Fatalf("分公司 %s 单号规则数量 = %d, 期望 %d", company.Code, ruleCount, len(biz.DefaultNumberRules()))
		}
	}
}
