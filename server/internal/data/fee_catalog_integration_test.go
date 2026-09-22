package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	"github.com/shopspring/decimal"
)

// feeCatalogHeadquartersContext 构造系统管理工作台主体。
func feeCatalogHeadquartersContext(orgID uuid.UUID, permissions ...string) context.Context {
	keys := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		keys[permission] = struct{}{}
	}
	return biz.WithPrincipal(context.Background(), &biz.Principal{
		Organization:      biz.Organization{ID: orgID, Kind: biz.OrganizationKindSystem},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: orgID, Kind: biz.OrganizationKindSystem}},
		RoleGrants:        []biz.RoleGrant{{RoleCode: "administrator", DataScope: biz.DataScopeAll, Permissions: keys}},
	})
}

// feeCatalogBranchContext 构造独立公司工作台主体。
func feeCatalogBranchContext(branchID, headquartersID uuid.UUID, permissions ...string) context.Context {
	keys := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		keys[permission] = struct{}{}
	}
	return biz.WithPrincipal(context.Background(), &biz.Principal{
		Organization: biz.Organization{ID: branchID, Kind: biz.OrganizationKindCompany},
		OrganizationNodes: []biz.OrganizationScopeNode{
			{ID: headquartersID, Kind: biz.OrganizationKindSystem},
			{ID: branchID, Kind: biz.OrganizationKindCompany},
		},
		RoleGrants: []biz.RoleGrant{{RoleCode: "administrator", DataScope: biz.DataScopeAll, Permissions: keys}},
	})
}

// newFeeCatalogReferences 准备费用设置引用脚手架：A 型计费单位与费用大类（全局）、
// 总部与分公司各自的 C 型税务名称行。返回值依次为计费单位、总部税务名称、
// 分公司税务名称、费用大类 ID。
func newFeeCatalogReferences(t *testing.T, data *Data, hqID, branchID uuid.UUID) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	suffix := uuid.NewString()[:8]
	ctx := context.Background()
	billingUnit, err := data.db.BillingUnit.Create().
		SetCode("FC-UNIT-" + suffix).SetName("费用科目测试票").
		SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatalf("创建计费单位: %v", err)
	}
	hqTaxable, err := data.db.TaxableService.Create().
		SetOrganizationID(hqID).SetName("费用科目测试总部应税服务-" + suffix).
		SetDefaultTaxRate("0").SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatalf("创建总部税务名称: %v", err)
	}
	branchTaxable, err := data.db.TaxableService.Create().
		SetOrganizationID(branchID).SetName("费用科目测试分公司应税服务-" + suffix).
		SetDefaultTaxRate("0").SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatalf("创建分公司税务名称: %v", err)
	}
	chargeCategory, err := data.db.MasterDataItem.Create().
		SetKind("charge_category").SetCode("FC-CAT-" + suffix).
		SetName("费用科目测试大类").SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatalf("创建费用大类: %v", err)
	}
	return billingUnit.ID, hqTaxable.ID, branchTaxable.ID, chargeCategory.ID
}

func feeSettingInputForTest(code string, billingUnitID, chargeCategoryID, taxableServiceID uuid.UUID) *biz.FeeSetting {
	return &biz.FeeSetting{
		FeeCode:          code,
		NameZH:           "费用科目测试-" + code,
		ChargeCategoryID: chargeCategoryID,
		DefaultCurrency:  "CNY",
		BillingUnitID:    billingUnitID,
		TaxRate:          decimal.RequireFromString("6.00"),
		TaxableServiceID: taxableServiceID,
		SortOrder:        100,
	}
}

// TestFeeCatalogReferenceValidationPostgres 验证引用校验三分支：计费单位与费用
// 大类接受 A 型全局行；税务名称（C 型）必须是本组织有效行；全部引用要求 enabled。
func TestFeeCatalogReferenceValidationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	hqID, branchID := newCrossCalcOrgTree(t, data)
	billingUnitID, hqTaxableID, branchTaxableID, chargeCategoryID := newFeeCatalogReferences(t, data, hqID, branchID)
	usecase := biz.NewFeeCatalogUsecase(NewFeeCatalogRepo(data))
	branchCtx := feeCatalogBranchContext(branchID, hqID, access.FinanceFeeSettingCreate)

	t.Run("分公司创建必须挂本组织税务名称", func(t *testing.T) {
		if _, err := usecase.CreateFeeSetting(branchCtx, branchID, uuid.New(), feeSettingInputForTest("FC_REF_TAXABLE", billingUnitID, chargeCategoryID, hqTaxableID)); !errors.Is(err, biz.ErrFeeCatalogReferenceInvalid) {
			t.Fatalf("引用他组织税务名称应被拒绝，实际 %v", err)
		}
		if _, err := usecase.CreateFeeSetting(branchCtx, branchID, uuid.New(), feeSettingInputForTest("FC_REF_TAXABLE", billingUnitID, chargeCategoryID, branchTaxableID)); err != nil {
			t.Fatalf("引用本组织税务名称应通过: %v", err)
		}
	})

	t.Run("A 型全局引用对分公司开放", func(t *testing.T) {
		// 计费单位与费用大类均为 A 型全局行（无组织维度），分公司直接引用应通过。
		if _, err := usecase.CreateFeeSetting(branchCtx, branchID, uuid.New(), feeSettingInputForTest("FC_REF_GLOBAL", billingUnitID, chargeCategoryID, branchTaxableID)); err != nil {
			t.Fatalf("引用 A 型全局计费单位与费用大类应通过: %v", err)
		}
	})

	t.Run("全部引用要求 enabled", func(t *testing.T) {
		ctx := context.Background()
		suffix := uuid.NewString()[:8]
		disabledUnit, err := data.db.BillingUnit.Create().SetCode("FC-DIS-UNIT-" + suffix).SetName("停用单位").SetEnabled(false).Save(ctx)
		if err != nil {
			t.Fatalf("创建停用计费单位: %v", err)
		}
		disabledCategory, err := data.db.MasterDataItem.Create().SetKind("charge_category").SetCode("FC-DIS-CAT-" + suffix).SetName("停用大类").SetEnabled(false).Save(ctx)
		if err != nil {
			t.Fatalf("创建停用费用大类: %v", err)
		}
		disabledTaxable, err := data.db.TaxableService.Create().SetOrganizationID(branchID).SetName("停用应税服务-" + suffix).SetDefaultTaxRate("0").SetEnabled(false).Save(ctx)
		if err != nil {
			t.Fatalf("创建停用税务名称: %v", err)
		}
		cases := []struct {
			name  string
			input *biz.FeeSetting
		}{
			{"停用计费单位", feeSettingInputForTest("FC_DIS_UNIT", disabledUnit.ID, chargeCategoryID, branchTaxableID)},
			{"停用费用大类", feeSettingInputForTest("FC_DIS_CAT", billingUnitID, disabledCategory.ID, branchTaxableID)},
			{"停用税务名称", feeSettingInputForTest("FC_DIS_TAX", billingUnitID, chargeCategoryID, disabledTaxable.ID)},
		}
		for _, testCase := range cases {
			if _, err := usecase.CreateFeeSetting(branchCtx, branchID, uuid.New(), testCase.input); !errors.Is(err, biz.ErrFeeCatalogReferenceInvalid) {
				t.Fatalf("%s应被拒绝，实际 %v", testCase.name, err)
			}
		}
	})
}

func TestFeeCatalogCompanyTemplatesPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	system, company := newCrossCalcOrgTree(t, d)
	unit, _, taxID, category := newFeeCatalogReferences(t, d, system, company)
	uc := biz.NewFeeCatalogUsecase(NewFeeCatalogRepo(d))
	systemCtx := feeCatalogHeadquartersContext(system, access.FinanceFeeSettingCreate, access.FinanceFeeSettingUpdate, access.FinanceFeeSettingRead)
	companyCtx := feeCatalogBranchContext(company, system, access.FinanceFeeSettingCreate, access.FinanceFeeSettingUpdate)
	input := &biz.FeeSettingTemplate{FeeSetting: *feeSettingInputForTest("FC_TEMPLATE", unit, category, uuid.Nil), TaxableService: biz.TaxableService{Name: "模板应税服务", DefaultTaxRate: decimal.NewFromInt(6)}}
	input.Enabled = true
	bad := *input
	bad.TaxableService.Name = ""
	if _, err := uc.SaveFeeSettingTemplate(systemCtx, uuid.Nil, &bad); !errors.Is(err, biz.ErrFeeCatalogInvalidArgument) {
		t.Fatalf("模板税务名称必须显式输入: %v", err)
	}
	if _, err := uc.ListFeeSettingTemplates(companyCtx, biz.FeeCatalogListOptions{Page: 1, PageSize: 200}); err == nil {
		t.Fatal("公司不得访问系统初始目录")
	}
	if _, err := uc.ListFeeSettingTemplates(systemCtx, biz.FeeCatalogListOptions{Page: 1, PageSize: 201}); !errors.Is(err, biz.ErrFeeCatalogInvalidArgument) {
		t.Fatalf("模板分页上限: %v", err)
	}
	template, err := uc.SaveFeeSettingTemplate(systemCtx, uuid.Nil, input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uc.SaveFeeSettingTemplate(companyCtx, uuid.Nil, input); err == nil {
		t.Fatal("公司不得维护系统模板")
	}
	if _, err := uc.CreateFeeSetting(systemCtx, system, uuid.New(), feeSettingInputForTest("FC_SYSTEM", unit, category, taxID)); err == nil {
		t.Fatal("系统工作台不得创建公司科目")
	}
	err = d.WithTx(context.Background(), func(tx *ent.Tx) error {
		return initializeCompanyFeeSettings(context.Background(), tx.Client(), company)
	})
	if err != nil {
		t.Fatal(err)
	}
	copied, err := d.db.FeeSetting.Query().Where(feesettingent.OrganizationIDEQ(company), feesettingent.FeeCodeEQ("FC_TEMPLATE")).WithTaxableService().Only(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if copied.Edges.TaxableService.OrganizationID != company || copied.ID == template.ID {
		t.Fatal("模板必须生成公司独立副本")
	}
	input.NameZH = "修改后的系统模板"
	if _, err := uc.SaveFeeSettingTemplate(systemCtx, template.ID, input); err != nil {
		t.Fatal(err)
	}
	unchanged, err := d.db.FeeSetting.Get(context.Background(), copied.ID)
	if err != nil || unchanged.NameZh == input.NameZH {
		t.Fatal("模板变更不应更新公司副本")
	}
	other, err := d.db.Organization.Create().SetCode("FC_OTHER").SetName("另一公司").SetKind("company").SetBaseCurrency("CNY").Save(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	otherCtx := feeCatalogBranchContext(other.ID, system, access.FinanceFeeSettingUpdate)
	if _, err := uc.UpdateFeeSetting(otherCtx, other.ID, uuid.New(), copied.ID, feeSettingInputForTest("FC_TEMPLATE", unit, category, taxID)); !errors.Is(err, biz.ErrFeeSettingNotFound) {
		t.Fatalf("跨公司更新必须不可见: %v", err)
	}
	result, err := uc.ListFeeSettings(companyCtx, company, biz.FeeCatalogListOptions{Page: 1, PageSize: 200})
	if err != nil || result.Total != 1 {
		t.Fatalf("公司目录: %v %v", result, err)
	}
	// 相同税务名称、不同默认值不能静默复用；失败须回滚整个公司初始化。
	conflict := *input
	conflict.FeeCode = "FC_CONFLICT"
	conflict.TaxableService.DefaultTaxRate = decimal.NewFromInt(9)
	if _, err := uc.SaveFeeSettingTemplate(systemCtx, uuid.Nil, &conflict); err != nil {
		t.Fatal(err)
	}
	err = d.WithTx(context.Background(), func(tx *ent.Tx) error {
		return initializeCompanyFeeSettings(context.Background(), tx.Client(), other.ID)
	})
	if !errors.Is(err, biz.ErrFeeCatalogReferenceInvalid) {
		t.Fatalf("冲突应明确失败: %v", err)
	}
	count, _ := d.db.FeeSetting.Query().Where(feesettingent.OrganizationIDEQ(other.ID)).Count(context.Background())
	if count != 0 {
		t.Fatal("失败留下半初始化科目")
	}
}
