package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	ruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
)

func TestPartnerAggregateSettlementWriteAccessPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := t.Context()
	f := newCreditPartnerFixture(t, data)
	repo := NewPartnerRepo(data)
	rule := func() *biz.PartnerSettlementRule {
		return &biz.PartnerSettlementRule{StatementMode: biz.PartnerStatementSingle, SettlementMethod: biz.PartnerSettlementByTicket, SettlementCurrency: "CNY", PaymentTermsDays: new(30), IsActive: true}
	}
	input := func(code string) *biz.Partner {
		return &biz.Partner{Code: code, LegalName: code, NormalizedName: code, Enabled: true, Roles: []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true, SettlementRule: rule()}}}
	}
	audit := func() *biz.AuditEvent {
		return &biz.AuditEvent{OrganizationID: &f.organizationID, Action: "partner.access.test", Result: "success", Details: map[string]string{}}
	}
	noAccess := biz.PartnerSettlementRuleWriteAccess{}
	createOnly := biz.PartnerSettlementRuleWriteAccess{Create: true}
	updateOnly := biz.PartnerSettlementRuleWriteAccess{Update: true}
	beforeAudit, err := data.db.AuditLog.Query().Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, access := range []biz.PartnerSettlementRuleWriteAccess{noAccess, updateOnly} {
		if _, err := repo.Create(ctx, f.organizationID, input("DENIED-CREATE"), access, audit()); !errors.Is(err, biz.ErrPermissionDenied) {
			t.Fatalf("无创建规则权限必须拒绝内嵌创建: %v", err)
		}
		if count, err := data.db.Partner.Query().Where(partnerent.CodeEQ("DENIED-CREATE")).Count(ctx); err != nil || count != 0 {
			t.Fatalf("越权创建未完整回滚: %d %v", count, err)
		}
	}
	if count, err := data.db.AuditLog.Query().Count(ctx); err != nil || count != beforeAudit {
		t.Fatalf("失败不得写成功审计: %d %v", count, err)
	}
	created, err := repo.Create(ctx, f.organizationID, input("ALLOWED-CREATE"), createOnly, audit())
	if err != nil {
		t.Fatal(err)
	}
	originalName := created.LegalName
	originalRuleID := created.Roles[0].SettlementRule.ID
	for _, access := range []biz.PartnerSettlementRuleWriteAccess{noAccess, createOnly} {
		changed := *created
		changed.LegalName = "越权名称"
		changed.NormalizedName = changed.LegalName
		changed.Roles = []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true, SettlementRule: rule()}}
		changed.Roles[0].SettlementRule.PaymentTermsDays = new(90)
		if _, err := repo.Update(ctx, f.organizationID, created.ID, &changed, access, audit()); !errors.Is(err, biz.ErrPermissionDenied) {
			t.Fatalf("无更新规则权限必须拒绝: %v", err)
		}
		stored, err := repo.Get(ctx, f.organizationID, created.ID)
		if err != nil || stored.LegalName != originalName || *stored.Roles[0].SettlementRule.PaymentTermsDays != 30 {
			t.Fatalf("越权修改必须原子回滚: %+v %v", stored, err)
		}
	}
	ordinary := *created
	ordinary.LegalName = "普通主档修改"
	ordinary.NormalizedName = ordinary.LegalName
	ordinary.Roles = []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true}}
	if _, err := repo.Update(ctx, f.organizationID, created.ID, &ordinary, noAccess, audit()); err != nil {
		t.Fatalf("普通主档修改不要求规则权限: %v", err)
	}
	storedRule, err := data.db.PartnerSettlementRule.Get(ctx, originalRuleID)
	if err != nil || *storedRule.PaymentTermsDays != 30 {
		t.Fatalf("普通保存不得改规则: %+v %v", storedRule, err)
	}
	changed := *created
	changed.Roles = []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true, SettlementRule: rule()}}
	changed.Roles[0].SettlementRule.PaymentTermsDays = new(60)
	if _, err := repo.Update(ctx, f.organizationID, created.ID, &changed, updateOnly, audit()); err != nil {
		t.Fatalf("更新权限可改已有规则: %v", err)
	}
	// 现有档案没有规则时，聚合 Update 依实际新增动作要求 Create。
	empty := input("NO-RULE")
	empty.Roles[0].SettlementRule = nil
	emptyCreated, err := repo.Create(ctx, f.organizationID, empty, noAccess, audit())
	if err != nil {
		t.Fatal(err)
	}
	emptyCreated.Roles[0].SettlementRule = rule()
	if _, err := repo.Update(ctx, f.organizationID, emptyCreated.ID, emptyCreated, updateOnly, audit()); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("Update-only 不得新增规则: %v", err)
	}
	if _, err := repo.Update(ctx, f.organizationID, emptyCreated.ID, emptyCreated, createOnly, audit()); err != nil {
		t.Fatalf("Create-only 可为已有档案补建规则: %v", err)
	}
	// 批量先创建普通档案，再命中无权更新的已有规则，必须整批回滚。
	newItem := input("IMPORT-NEW")
	newItem.Roles[0].SettlementRule = nil
	priorAudit, err := data.db.AuditLog.Query().Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Import(ctx, f.organizationID, biz.PartnerImportUpsert, []*biz.Partner{newItem, &changed}, createOnly, audit()); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("导入更新规则必须按真实动作鉴权: %v", err)
	}
	if count, err := data.db.Partner.Query().Where(partnerent.CodeEQ("IMPORT-NEW")).Count(ctx); err != nil || count != 0 {
		t.Fatalf("导入前序创建未回滚: %d %v", count, err)
	}
	if count, err := data.db.AuditLog.Query().Count(ctx); err != nil || count != priorAudit {
		t.Fatalf("失败导入不得写成功审计: %d %v", count, err)
	}
	if _, err := repo.Import(ctx, f.organizationID, biz.PartnerImportUpsert, []*biz.Partner{input("IMPORT-DENIED")}, noAccess, audit()); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("导入新建规则必须鉴权: %v", err)
	}
	// 停用规则仍属于实际更新；有多条规则仍保持明确拒绝，不静默选择一条。
	if err := data.db.PartnerSettlementRule.UpdateOneID(originalRuleID).SetIsActive(false).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Update(ctx, f.organizationID, created.ID, &changed, createOnly, audit()); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("停用规则不能当成无规则: %v", err)
	}
	roleID := storedRule.PartnerRoleID
	if _, err := data.db.PartnerSettlementRule.Create().SetPartnerRoleID(roleID).SetStatementMode("single").SetSettlementMethod("prepaid").SetSettlementCurrency("CNY").Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Update(ctx, f.organizationID, created.ID, &changed, updateOnly, audit()); !errors.Is(err, biz.ErrPartnerSettlementRuleInvalidArgument) {
		t.Fatalf("多条规则不得静默更新: %v", err)
	}
	if count, err := data.db.PartnerSettlementRule.Query().Where(ruleent.PartnerRoleIDEQ(roleID)).Count(ctx); err != nil || count != 2 {
		t.Fatalf("多规则拒绝后不应变更: %d %v", count, err)
	}
}

func TestPartnerInvoiceProfileDefaultVersionPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	f := newCreditPartnerFixture(t, data)
	repo := NewPartnerInvoiceProfileRepo(data)
	audit := func() *biz.AuditEvent {
		return &biz.AuditEvent{OrganizationID: &f.organizationID, Action: "partner.invoice_profile.test", Result: "success", Details: map[string]string{}}
	}
	for _, mode := range []string{"create", "update"} {
		t.Run(mode, func(t *testing.T) {
			partnerID, _ := f.newCreditPartner("抬头" + mode)
			profile := func(title string, isDefault bool) *biz.PartnerInvoiceProfile {
				return &biz.PartnerInvoiceProfile{ID: uuid.New(), PartnerID: partnerID, InvoiceTitle: title, TaxpayerIdentificationNo: "913100000000000000", DefaultInvoiceType: biz.FinanceInvoiceNormal, IsDefault: isDefault, Enabled: true}
			}
			a, err := repo.Create(ctx, f.organizationID, profile("默认A", true), audit())
			if err != nil {
				t.Fatal(err)
			}
			other, err := repo.Create(ctx, f.organizationID, profile("非默认C", false), audit())
			if err != nil {
				t.Fatal(err)
			}
			stale := *a
			bInput := profile("切换B", mode == "create")
			b, err := repo.Create(ctx, f.organizationID, bInput, audit())
			if err != nil {
				t.Fatal(err)
			}
			if mode == "update" {
				b.IsDefault = true
				b, err = repo.Update(ctx, f.organizationID, b, b.Version, audit())
				if err != nil {
					t.Fatal(err)
				}
			}
			latest, err := data.db.PartnerInvoiceProfile.Get(ctx, a.ID)
			if err != nil {
				t.Fatal(err)
			}
			if latest.IsDefault || latest.Version != a.Version+1 {
				t.Fatalf("清旧默认必须推进版本: %+v", latest)
			}
			unchanged, err := data.db.PartnerInvoiceProfile.Get(ctx, other.ID)
			if err != nil || unchanged.Version != other.Version {
				t.Fatalf("未受影响项版本不能变: %+v %v", unchanged, err)
			}
			stale.RegisteredPhone = "旧表单改电话"
			if _, err := repo.Update(ctx, f.organizationID, &stale, stale.Version, audit()); !errors.Is(err, biz.ErrPartnerInvoiceProfileVersionConflict) {
				t.Fatalf("旧表单应冲突而非覆盖默认: %v", err)
			}
			current := partnerInvoiceProfileToBiz(latest)
			current.RegisteredPhone = "最新表单改电话"
			if _, err := repo.Update(ctx, f.organizationID, current, current.Version, audit()); err != nil {
				t.Fatalf("最新表单正常编辑应成功: %v", err)
			}
			currentB, err := data.db.PartnerInvoiceProfile.Get(ctx, b.ID)
			if err != nil || !currentB.IsDefault {
				t.Fatalf("旧表单失败不能改默认B: %+v %v", currentB, err)
			}
			// 清默认后新标题冲突，整个事务应回滚被清行及其版本。
			before := *currentB
			if _, err := repo.Create(ctx, f.organizationID, profile("默认A", true), audit()); err == nil {
				t.Fatal("重复标题应失败")
			}
			after, err := data.db.PartnerInvoiceProfile.Get(ctx, b.ID)
			if err != nil || !after.IsDefault || after.Version != before.Version {
				t.Fatalf("失败必须回滚默认与版本: %+v %v", after, err)
			}
		})
	}
}
