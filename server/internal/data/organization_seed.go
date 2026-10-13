package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
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
}

// CreateDefaultBranchCompanies 与 bootstrap 共用锁序及停用规则，并补齐实际存在的公司。
func CreateDefaultBranchCompanies(ctx context.Context, tx *ent.Tx, systemID uuid.UUID) (*AdministratorProvisioningSummary, error) {
	client := tx.Client()
	account, err := lockInitialAdministrator(ctx, client)
	if err != nil {
		return nil, err
	}
	organizations, err := client.Organization.Query().Order(organizationent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	system, err := uniqueAdministratorSystem(organizations)
	if err != nil {
		return nil, err
	}
	if system == nil || system.ID != systemID || !system.Enabled {
		return nil, biz.ErrAdminBootstrapSystemInactive
	}
	plans, err := prepareAdministratorPlans(ctx, client, account, organizations, systemID)
	if err != nil {
		return nil, err
	}
	created, err := createDefaultBranchCompanyRecords(ctx, tx)
	if err != nil {
		return nil, err
	}
	summary := &AdministratorProvisioningSummary{Username: account.Username, CompaniesCreated: len(created)}
	for _, item := range created {
		plans = append(plans, &administratorWorkspacePlan{organization: item})
	}
	for _, plan := range plans {
		if err := provisionAdministratorPlan(ctx, client, account, plan, "seed-branches", &account.ID, summary); err != nil {
			return nil, err
		}
	}
	return summary, nil
}

// createDefaultBranchCompanyRecords 要求调用者已经锁定初始管理员和现有组织。
// 已存在的公司保留配置，角色/成员的补齐与停用检查由调用者统一处理。
func createDefaultBranchCompanyRecords(ctx context.Context, tx *ent.Tx) ([]*ent.Organization, error) {
	created := make([]*ent.Organization, 0, len(defaultBranchCompanySeeds))
	for _, seed := range defaultBranchCompanySeeds {
		existing, err := tx.Organization.Query().Where(organizationent.CodeEQ(seed.Code)).Only(ctx)
		if err == nil {
			if existing.Kind != organizationent.KindCompany {
				return nil, fmt.Errorf("默认公司编码 %s 被非公司组织使用，请人工核对", seed.Code)
			}
			continue
		}
		if !ent.IsNotFound(err) {
			return nil, fmt.Errorf("检查分公司种子 %s: %w", seed.Code, err)
		}
		item, err := tx.Organization.Create().SetCode(seed.Code).SetName(seed.Name).SetKind(organizationent.KindCompany).SetBaseCurrency("CNY").SetEnabledCurrencies(defaultCompanyEnabledCurrencies("CNY")).Save(ctx)
		if err != nil {
			return nil, mapEntError(err, nil, biz.ErrAdminOrganizationCodeExists)
		}
		if err := CreateDefaultNumberRules(ctx, tx, item.ID); err != nil {
			return nil, err
		}
		if err := initializeCompanyFeeSettings(ctx, tx.Client(), item.ID); err != nil {
			return nil, err
		}
		created = append(created, item)
	}
	return created, nil
}
