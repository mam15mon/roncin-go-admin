package data

import (
	"context"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestReferenceDataOrganizationCurrencyEnablement(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	hqID, branchID := newCrossCalcOrgTree(t, data)
	repo := NewReferenceDataRepo(data)

	// 1. 分公司默认读取货币列表：本位币锁定，主流币种默认启用
	currencies, err := repo.ListCurrencies(ctx, branchID, false)
	if err != nil {
		t.Fatalf("ListCurrencies(branchID, false) error = %v", err)
	}
	if len(currencies) == 0 {
		t.Fatalf("currencies is empty")
	}

	// 本位币 CAD 必须排在第一位且 IsBaseCurrency=true, Enabled=true
	first := currencies[0]
	if first.Code != "CAD" || !first.IsBaseCurrency || !first.Enabled {
		t.Fatalf("first currency = %#v, want CAD with IsBaseCurrency=true and Enabled=true", first)
	}

	// 2. 分公司仅拉取已启用币种 (enabledOnly=true)
	enabledCurrencies, err := repo.ListCurrencies(ctx, branchID, true)
	if err != nil {
		t.Fatalf("ListCurrencies(branchID, true) error = %v", err)
	}
	enabledCodeMap := make(map[string]bool)
	for _, c := range enabledCurrencies {
		if !c.Enabled {
			t.Fatalf("currency %s should be enabled", c.Code)
		}
		enabledCodeMap[c.Code] = true
	}

	// 验证 9 个核心币种及本位币默认必须全部激活
	expectedDefaults := []string{"USD", "GBP", "EUR", "CHF", "AUD", "SGD", "HKD", "THB", "CNY", "CAD"}
	for _, expected := range expectedDefaults {
		if !enabledCodeMap[expected] {
			t.Fatalf("expected default currency %s to be enabled, but it was not", expected)
		}
	}

	// 验证未在默认列表中的币种（如 JPY）默认未激活
	if enabledCodeMap["JPY"] {
		t.Fatalf("JPY should not be enabled by default for branch")
	}

	// 3. 尝试禁用本位币必须被拦截报错
	if _, err := repo.SetCurrencyEnabled(ctx, branchID, "CAD", false); err != biz.ErrCurrencyBaseCannotBeDisabled {
		t.Fatalf("SetCurrencyEnabled(CAD, false) error = %v, want ErrCurrencyBaseCannotBeDisabled", err)
	}

	// 4. 分公司停用与重新启用非本位币 (如 USD)
	updated, err := repo.SetCurrencyEnabled(ctx, branchID, "USD", false)
	if err != nil {
		t.Fatalf("SetCurrencyEnabled(USD, false) error = %v", err)
	}
	if updated.Enabled {
		t.Fatalf("updated USD enabled = true, want false")
	}

	// 再次拉取 enabledOnly 应该不再包含 USD
	afterDisable, err := repo.ListCurrencies(ctx, branchID, true)
	if err != nil {
		t.Fatalf("ListCurrencies(branchID, true) error = %v", err)
	}
	for _, c := range afterDisable {
		if c.Code == "USD" {
			t.Fatalf("disabled USD should not be in enabledOnly list")
		}
	}

	// 重新启用 USD
	reEnabled, err := repo.SetCurrencyEnabled(ctx, branchID, "USD", true)
	if err != nil {
		t.Fatalf("SetCurrencyEnabled(USD, true) error = %v", err)
	}
	if !reEnabled.Enabled {
		t.Fatalf("reEnabled USD enabled = false, want true")
	}

	// 5. 总部视角读取全部币种均为启用
	hqCurrencies, err := repo.ListCurrencies(ctx, hqID, false)
	if err != nil {
		t.Fatalf("ListCurrencies(hqID, false) error = %v", err)
	}
	for _, c := range hqCurrencies {
		if !c.Enabled {
			t.Fatalf("HQ currency %s should be enabled", c.Code)
		}
	}
}

func TestAdminCreateOrganizationInitializesEnabledCurrencies(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	adminRepo := NewAdminRepo(data)
	refRepo := NewReferenceDataRepo(data)
	hqID, _ := newCrossCalcOrgTree(t, data)

	newCompany, err := adminRepo.CreateOrganization(ctx, &biz.AdminOrganization{
		Code:         "NEW-BRANCH-01",
		Name:         "新设立分公司",
		Kind:         biz.OrganizationKindCompany,
		BaseCurrency: "USD",
	}, &biz.AuditEvent{Action: "admin.organization.create", Result: "success", Details: map[string]string{}})
	if err != nil {
		t.Fatalf("CreateOrganization error = %v", err)
	}

	// 检查数据库中持久化的 enabled_currencies
	orgEntity, err := data.db.Organization.Get(ctx, newCompany.ID)
	if err != nil {
		t.Fatalf("Get organization error = %v", err)
	}
	if len(orgEntity.EnabledCurrencies) == 0 {
		t.Fatalf("expected organization to have persisted enabled_currencies")
	}

	// 检查通过 ListCurrencies 读取的已激活币种
	enabledList, err := refRepo.ListCurrencies(ctx, newCompany.ID, true)
	if err != nil {
		t.Fatalf("ListCurrencies error = %v", err)
	}
	enabledMap := make(map[string]bool)
	for _, item := range enabledList {
		enabledMap[item.Code] = true
	}

	wantCodes := []string{"USD", "GBP", "EUR", "CHF", "AUD", "SGD", "HKD", "THB", "CNY"}
	for _, code := range wantCodes {
		if !enabledMap[code] {
			t.Fatalf("expected currency %s to be enabled on new company", code)
		}
	}
	if enabledMap["JPY"] {
		t.Fatalf("JPY should not be enabled by default on new company")
	}

	// 2. 测试创建本币为非 9 大预设币种（例如 CAD）的公司：本币 CAD 必须一并激活
	cadCompany, err := adminRepo.CreateOrganization(ctx, &biz.AdminOrganization{
		Code:         "NEW-BRANCH-CAD",
		Name:         "新设立加拿大分公司",
		Kind:         biz.OrganizationKindCompany,
		BaseCurrency: "CAD",
	}, &biz.AuditEvent{Action: "admin.organization.create", Result: "success", Details: map[string]string{}})
	if err != nil {
		t.Fatalf("CreateOrganization with CAD error = %v", err)
	}

	cadOrg, err := data.db.Organization.Get(ctx, cadCompany.ID)
	if err != nil {
		t.Fatalf("Get cad organization error = %v", err)
	}
	hasCAD := false
	for _, c := range cadOrg.EnabledCurrencies {
		if c == "CAD" {
			hasCAD = true
			break
		}
	}
	if !hasCAD {
		t.Fatalf("expected CAD to be in enabled_currencies for CAD company")
	}

	// 3. 测试更新公司本币：将 newCompany 从 USD 改为 CAD，CAD 必须自动加入 enabled_currencies
	updatedCompany, err := adminRepo.UpdateOrganization(ctx, hqID, &biz.AdminOrganization{
		ID:           newCompany.ID,
		Name:         "新设立分公司-已更名",
		Kind:         biz.OrganizationKindCompany,
		Enabled:      true,
		BaseCurrency: "CAD",
	}, &biz.AuditEvent{Action: "admin.organization.create", Result: "success", Details: map[string]string{}})
	if err != nil {
		t.Fatalf("UpdateOrganization error = %v", err)
	}
	if updatedCompany.BaseCurrency != "CAD" {
		t.Fatalf("updatedCompany baseCurrency = %s, want CAD", updatedCompany.BaseCurrency)
	}

	reloaded, err := data.db.Organization.Get(ctx, newCompany.ID)
	if err != nil {
		t.Fatalf("Get reloaded organization error = %v", err)
	}
	hasUpdatedBase := false
	for _, c := range reloaded.EnabledCurrencies {
		if c == "CAD" {
			hasUpdatedBase = true
			break
		}
	}
	if !hasUpdatedBase {
		t.Fatalf("expected new base currency CAD to be added to enabled_currencies on update")
	}
}
