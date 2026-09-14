package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	"github.com/shopspring/decimal"
)

// fee_catalog_integration_test.go — 阶段三费用科目两级的真实库定向验证：
// B 型选择器穿透（同码本地行覆盖基线行 + 分页计数一致）、引用校验三分支
//（计费单位/费用大类查 A 型全局、税务名称查 C 型本组织行、全部要求 enabled）、
// 组织归属写权限（总部落基线行、分公司落本组织行）与基线行禁改。

// feeCatalogHeadquartersContext 构造总部根节点工作台主体上下文（持有指定权限码）。
func feeCatalogHeadquartersContext(orgID uuid.UUID, permissions ...string) context.Context {
	keys := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		keys[permission] = struct{}{}
	}
	return biz.WithPrincipal(context.Background(), &biz.Principal{
		Organization:      biz.Organization{ID: orgID, Kind: biz.OrganizationKindHeadquarters},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: orgID, Kind: biz.OrganizationKindHeadquarters}},
		RoleGrants:        []biz.RoleGrant{{RoleCode: "administrator", DataScope: biz.DataScopeAll, Permissions: keys}},
	})
}

// feeCatalogBranchContext 构造挂在总部之下的公司工作台主体上下文（持权限码但
// 组织身份非总部）。
func feeCatalogBranchContext(branchID, headquartersID uuid.UUID, permissions ...string) context.Context {
	keys := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		keys[permission] = struct{}{}
	}
	return biz.WithPrincipal(context.Background(), &biz.Principal{
		Organization: biz.Organization{ID: branchID, Kind: biz.OrganizationKindCompany},
		OrganizationNodes: []biz.OrganizationScopeNode{
			{ID: headquartersID, Kind: biz.OrganizationKindHeadquarters},
			{ID: branchID, ParentID: &headquartersID, Kind: biz.OrganizationKindCompany},
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
		FeeCode:           code,
		NameZH:            "费用科目测试-" + code,
		ChargeCategoryID:  chargeCategoryID,
		DefaultCurrency:   "CNY",
		BillingUnitID:     billingUnitID,
		TaxRate:           decimal.RequireFromString("6.00"),
		TaxableServiceID:  taxableServiceID,
		SortOrder:         100,
	}
}

// TestFeeCatalogBaselineShadowingPostgres 验证费用设置列表（费用选择器同源谓词）
// 的 Shadowing 穿透：同码本地行整行覆盖基线行、分页计数与去重一致、组织视图互不
// 干扰，以及两条 NULL 基线同码被数据库部分唯一索引拒绝并映射业务错误。
func TestFeeCatalogBaselineShadowingPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	hqID, branchID := newCrossCalcOrgTree(t, data)
	billingUnitID, hqTaxableID, branchTaxableID, chargeCategoryID := newFeeCatalogReferences(t, data, hqID, branchID)
	repo := NewFeeCatalogRepo(data)
	hqCtx := feeCatalogHeadquartersContext(hqID, access.FinanceFeeSettingCreate)
	branchCtx := feeCatalogBranchContext(branchID, hqID, access.FinanceFeeSettingCreate)
	usecase := biz.NewFeeCatalogUsecase(repo)

	baseline, err := usecase.CreateFeeSetting(hqCtx, hqID, uuid.New(), feeSettingInputForTest("FC_OCEAN", billingUnitID, chargeCategoryID, hqTaxableID))
	if err != nil {
		t.Fatalf("总部创建基线行: %v", err)
	}
	if _, err := usecase.CreateFeeSetting(hqCtx, hqID, uuid.New(), feeSettingInputForTest("FC_DOC", billingUnitID, chargeCategoryID, hqTaxableID)); err != nil {
		t.Fatalf("总部创建第二条基线行: %v", err)
	}
	if _, err := usecase.CreateFeeSetting(hqCtx, hqID, uuid.New(), feeSettingInputForTest("FC_OCEAN", billingUnitID, chargeCategoryID, hqTaxableID)); !errors.Is(err, biz.ErrFeeSettingCodeExists) {
		t.Fatalf("两条 NULL 基线同码应被部分唯一索引拒绝并映射业务错误，实际 %v", err)
	}
	local, err := usecase.CreateFeeSetting(branchCtx, branchID, uuid.New(), feeSettingInputForTest("FC_OCEAN", billingUnitID, chargeCategoryID, branchTaxableID))
	if err != nil {
		t.Fatalf("分公司创建同码本地行: %v", err)
	}
	if local.OrganizationID == nil || *local.OrganizationID != branchID {
		t.Fatalf("分公司创建应落本组织行，实际归属 %v", local.OrganizationID)
	}
	if baseline.OrganizationID != nil {
		t.Fatalf("总部创建应落 NULL 基线行，实际归属 %v", baseline.OrganizationID)
	}

	t.Run("分公司视图同码本地行覆盖基线行", func(t *testing.T) {
		result, err := repo.ListFeeSettings(context.Background(), branchID, biz.FeeCatalogListOptions{Page: 1, PageSize: 200})
		if err != nil {
			t.Fatalf("分公司列表: %v", err)
		}
		if result.Total != 2 {
			t.Fatalf("分公司视图应只含本地 FC_OCEAN 与基线 FC_DOC 共 2 行，实际 total=%d", result.Total)
		}
		for _, item := range result.Items {
			if item.FeeCode == "FC_OCEAN" && item.ID != local.ID {
				t.Fatalf("同码基线行应被本地行整行覆盖，不应出现旧基线行 ID %s", item.ID)
			}
		}
		found := false
		for _, item := range result.Items {
			if item.ID == local.ID {
				found = true
				if item.OrganizationID == nil || *item.OrganizationID != branchID {
					t.Fatalf("命中行应为本组织行，实际 %v", item.OrganizationID)
				}
			}
		}
		if !found {
			t.Fatal("分公司视图应包含本地 FC_OCEAN 行")
		}
	})

	t.Run("总部视图不被分公司本地行覆盖", func(t *testing.T) {
		result, err := repo.ListFeeSettings(context.Background(), hqID, biz.FeeCatalogListOptions{Page: 1, PageSize: 200})
		if err != nil {
			t.Fatalf("总部列表: %v", err)
		}
		if result.Total != 2 {
			t.Fatalf("总部视图应含两条基线行，实际 total=%d", result.Total)
		}
		for _, item := range result.Items {
			if item.OrganizationID != nil {
				t.Fatalf("总部视图不应出现他组织行 %s", item.FeeCode)
			}
		}
	})

	t.Run("分页计数与去重结果一致", func(t *testing.T) {
		result, err := repo.ListFeeSettings(context.Background(), branchID, biz.FeeCatalogListOptions{Page: 1, PageSize: 1})
		if err != nil {
			t.Fatalf("分页查询: %v", err)
		}
		if result.Total != 2 || len(result.Items) != 1 {
			t.Fatalf("pageSize=1 应返回 total=2 且单行，实际 total=%d len=%d", result.Total, len(result.Items))
		}
		result, err = repo.ListFeeSettings(context.Background(), branchID, biz.FeeCatalogListOptions{Keyword: "FC_OCEAN", Page: 1, PageSize: 200})
		if err != nil {
			t.Fatalf("关键字查询: %v", err)
		}
		if result.Total != 1 || len(result.Items) != 1 || result.Items[0].ID != local.ID {
			t.Fatalf("同码关键字搜索应只命中本地行，实际 total=%d", result.Total)
		}
	})
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

// TestFeeCatalogOwnershipWritePostgres 验证行归属写权限：总部写 NULL 基线行、
// 分公司写本组织行、无权限分公司被拒；基线行对分公司禁改、他组织行互不可见、
// 归属行各自可改。
func TestFeeCatalogOwnershipWritePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	hqID, branchID := newCrossCalcOrgTree(t, data)
	billingUnitID, hqTaxableID, branchTaxableID, chargeCategoryID := newFeeCatalogReferences(t, data, hqID, branchID)
	repo := NewFeeCatalogRepo(data)
	usecase := biz.NewFeeCatalogUsecase(repo)
	hqCtx := feeCatalogHeadquartersContext(hqID, access.FinanceFeeSettingCreate, access.FinanceFeeSettingUpdate)
	branchCtx := feeCatalogBranchContext(branchID, hqID, access.FinanceFeeSettingCreate, access.FinanceFeeSettingUpdate)
	actor := uuid.New()

	baseline, err := usecase.CreateFeeSetting(hqCtx, hqID, actor, feeSettingInputForTest("FC_OWN_BASE", billingUnitID, chargeCategoryID, hqTaxableID))
	if err != nil {
		t.Fatalf("总部创建基线行: %v", err)
	}
	local, err := usecase.CreateFeeSetting(branchCtx, branchID, actor, feeSettingInputForTest("FC_OWN_LOCAL", billingUnitID, chargeCategoryID, branchTaxableID))
	if err != nil {
		t.Fatalf("分公司创建本地行: %v", err)
	}

	t.Run("无权限分公司写本组织行被拒", func(t *testing.T) {
		noPermissionCtx := feeCatalogBranchContext(branchID, hqID)
		if _, err := usecase.CreateFeeSetting(noPermissionCtx, branchID, actor, feeSettingInputForTest("FC_NO_PERM", billingUnitID, chargeCategoryID, branchTaxableID)); !errors.Is(err, biz.ErrPermissionDenied) {
			t.Fatalf("无权限码应拒绝，实际 %v", err)
		}
	})

	t.Run("基线行对分公司禁改", func(t *testing.T) {
		update := feeSettingInputForTest("FC_OWN_BASE", billingUnitID, chargeCategoryID, branchTaxableID)
		update.ID = baseline.ID
		if _, err := usecase.UpdateFeeSetting(branchCtx, branchID, actor, baseline.ID, update); !errors.Is(err, biz.ErrFeeSettingNotFound) {
			t.Fatalf("分公司改基线行应按不存在拒绝，实际 %v", err)
		}
	})

	t.Run("分公司他组织行不可见", func(t *testing.T) {
		_, otherBranchID := newCrossCalcOrgTree(t, data)
		otherCtx := feeCatalogBranchContext(otherBranchID, hqID, access.FinanceFeeSettingUpdate)
		update := feeSettingInputForTest("FC_OWN_LOCAL", billingUnitID, chargeCategoryID, branchTaxableID)
		update.ID = local.ID
		if _, err := usecase.UpdateFeeSetting(otherCtx, otherBranchID, actor, local.ID, update); !errors.Is(err, biz.ErrFeeSettingNotFound) {
			t.Fatalf("他组织改分公司本地行应按不存在拒绝，实际 %v", err)
		}
	})

	t.Run("总部可改基线行且归属不变", func(t *testing.T) {
		update := feeSettingInputForTest("FC_OWN_BASE", billingUnitID, chargeCategoryID, hqTaxableID)
		update.NameZH = "费用科目测试-基线行已更新"
		update.Enabled = false
		updated, err := usecase.UpdateFeeSetting(hqCtx, hqID, actor, baseline.ID, update)
		if err != nil {
			t.Fatalf("总部改基线行: %v", err)
		}
		if updated.OrganizationID != nil || updated.NameZH != "费用科目测试-基线行已更新" || updated.Enabled {
			t.Fatalf("基线行更新结果不符: 归属 %v 名称 %s 启用 %v", updated.OrganizationID, updated.NameZH, updated.Enabled)
		}
	})

	t.Run("分公司可改本组织行", func(t *testing.T) {
		update := feeSettingInputForTest("FC_OWN_LOCAL", billingUnitID, chargeCategoryID, branchTaxableID)
		update.SortOrder = 42
		updated, err := usecase.UpdateFeeSetting(branchCtx, branchID, actor, local.ID, update)
		if err != nil {
			t.Fatalf("分公司改本组织行: %v", err)
		}
		if updated.OrganizationID == nil || *updated.OrganizationID != branchID || updated.SortOrder != 42 {
			t.Fatalf("本组织行更新结果不符: 归属 %v 排序 %d", updated.OrganizationID, updated.SortOrder)
		}
		// 归属不可迁移：数据库中该行 organization_id 必须仍是本组织。
		persisted, err := data.db.FeeSetting.Query().Where(feesettingent.IDEQ(local.ID)).Only(context.Background())
		if err != nil {
			t.Fatalf("回读本组织行: %v", err)
		}
		if persisted.OrganizationID == nil || *persisted.OrganizationID != branchID {
			t.Fatalf("更新后归属应保持本组织，实际 %v", persisted.OrganizationID)
		}
	})
}
