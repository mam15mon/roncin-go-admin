package data

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

func TestCompanyNameEnMigrationPostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261104100000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	bj, cd, department := uuid.New(), uuid.New(), uuid.New()
	stamp := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	_, err := data.sqlDB.ExecContext(ctx, `INSERT INTO organizations (id,created_at,updated_at,code,name,kind,parent_id,base_currency,search_keywords) VALUES ($1,$4,$4,'BJ','北京公司','company',NULL,'CNY','BEIJING'),($2,$4,$4,'CD','成都公司','company',NULL,'CNY','CHENGDU'),($3,$4,$4,'DEPT','部门','department',$1,NULL,'BUMEN')`, bj, cd, department, stamp)
	if err != nil {
		t.Fatal(err)
	}
	if err = migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	beforeRepeat := map[uuid.UUID]time.Time{}
	for _, id := range []uuid.UUID{bj, cd, department} {
		item, err := data.db.Organization.Get(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if id == bj || id == cd {
			expected := "RONCIN (BEIJING) SUPPLY CHAIN MANAGEMENT CO., LTD"
			if id == cd {
				expected = "RONCIN (CHENGDU) SUPPLY CHAIN MANAGEMENT CO., LTD"
			}
			if item.NameEn == nil || *item.NameEn != expected || !strings.Contains(item.SearchKeywords, expected) {
				t.Fatalf("公司回填或检索失败: %#v", item)
			}
		} else if item.NameEn != nil {
			t.Fatalf("不应为部门填写公司英文名: %#v", item)
		}
		if !item.UpdatedAt.Equal(stamp) {
			t.Fatalf("迁移改变原业务更新时间: %#v", item)
		}
		beforeRepeat[id] = item.UpdatedAt
	}
	if err = migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	for id, stamp := range beforeRepeat {
		item, err := data.db.Organization.Get(ctx, id)
		if err != nil || !item.UpdatedAt.Equal(stamp) {
			t.Fatalf("重复迁移不幂等: %#v %v", item, err)
		}
	}
}

func TestCompanyNameEnAdminRoundTripPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	if err := data.WithTx(ctx, func(tx *ent.Tx) error {
		_, err := BootstrapAdministrator(ctx, tx, BootstrapAdministratorInput{Username: "name.admin", DisplayName: "名称测试管理员", PlainPassword: "test-password-123", HashPassword: func(string) (string, error) { return "test-hash", nil }, OrganizationCode: "SYS", OrganizationName: "系统管理"})
		return err
	}); err != nil {
		t.Fatal(err)
	}
	system, err := data.db.Organization.Query().Where(organizationent.CodeEQ("SYS")).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewAdminRepo(data)
	audit := func(action string) *biz.AuditEvent {
		return &biz.AuditEvent{Action: action, Result: "success", Details: map[string]string{}}
	}
	uc := biz.NewAdminUsecase(repo, data)
	actor := uuid.New()
	company, err := uc.CreateOrganization(ctx, actor, &biz.AdminOrganization{Code: "EXAMPLE", Name: "人工名称测试公司", NameEn: "  Example Company's Official CO., LTD  ", Kind: biz.OrganizationKindCompany, BaseCurrency: "CNY"})
	if err != nil || company.NameEn != "Example Company's Official CO., LTD" {
		t.Fatalf("创建英文名失败: %#v %v", company, err)
	}
	name := company.NameEn
	company, err = uc.UpdateOrganization(ctx, actor, system.ID, company.ID, "改中文名称", true, "CNY", name)
	if err != nil || company.NameEn != name {
		t.Fatalf("普通修改丢英文名: %#v %v", company, err)
	}
	listed, err := repo.ListOrganizations(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, org := range listed {
		if org.ID == company.ID {
			found = org.NameEn == name
		}
	}
	if !found {
		t.Fatal("列表未回显英文名")
	}
	entity, err := data.db.Organization.Get(ctx, company.ID)
	if err != nil || !strings.Contains(entity.SearchKeywords, strings.ToUpper(name)) {
		t.Fatalf("英文检索键缺失: %#v %v", entity, err)
	}
	// 200 字符边界必须贯穿 Ent 校验，不能退化为 200 个 UTF-8 字节。
	unicodeName := strings.Repeat("É", 200)
	unicodeCompany, err := uc.CreateOrganization(ctx, actor, &biz.AdminOrganization{Code: "UNICODE-NAME", Name: "多字节正式名称测试公司", NameEn: unicodeName, Kind: biz.OrganizationKindCompany, BaseCurrency: "CNY"})
	if err != nil || unicodeCompany.NameEn != unicodeName {
		t.Fatalf("200字符创建被字节长度误拒绝: %#v %v", unicodeCompany, err)
	}
	unicodeSaved, err := uc.UpdateOrganization(ctx, actor, system.ID, company.ID, "改中文名称", true, "CNY", unicodeName)
	if err != nil || unicodeSaved.NameEn != unicodeName {
		t.Fatalf("200字符更新被字节长度误拒绝: %#v %v", unicodeSaved, err)
	}
	if _, err = uc.UpdateOrganization(ctx, actor, system.ID, company.ID, "改中文名称", true, "CNY", unicodeName+"É"); err == nil {
		t.Fatal("201字符更新未拒绝")
	}
	entity, err = data.db.Organization.Get(ctx, company.ID)
	if err != nil || entity.NameEn == nil || *entity.NameEn != unicodeName {
		t.Fatalf("越界请求改变已保存名称: %#v %v", entity, err)
	}
	changed, err := uc.UpdateOrganization(ctx, actor, system.ID, company.ID, "改中文名称", true, "CNY", "New Official CO., LTD")
	if err != nil || changed.NameEn != "New Official CO., LTD" {
		t.Fatalf("更新英文名失败: %#v %v", changed, err)
	}
	logs, err := data.db.AuditLog.Query().Where(auditlogent.ActionEQ("admin.organization.update")).All(ctx)
	if err != nil {
		t.Fatal(err)
	}
	audited := false
	for _, row := range logs {
		var details map[string]string
		if json.Unmarshal(row.Details, &details) == nil && details["organization.name_en"] == changed.NameEn {
			audited = true
		}
	}
	if !audited {
		t.Fatal("审计未记录正式英文名称")
	}
	// 仓储范围和审计失败仍需完整回滚名称写入。
	outsider, err := data.db.Organization.Create().SetCode("OTHER").SetName("其他公司").SetKind(organizationent.KindCompany).SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	input := &biz.AdminOrganization{ID: company.ID, Name: company.Name, NameEn: "FORBIDDEN", Kind: biz.OrganizationKindCompany, Enabled: true, BaseCurrency: "CNY"}
	if _, err := repo.UpdateOrganization(ctx, outsider.ID, input, audit("admin.organization.update")); err == nil {
		t.Fatal("跨公司更改未拒绝")
	}
	input.NameEn = "ROLLBACK"
	badAudit := audit("")
	if _, err := repo.UpdateOrganization(ctx, system.ID, input, badAudit); err == nil {
		t.Fatal("非法审计未使更改失败")
	}
	entity, err = data.db.Organization.Get(ctx, company.ID)
	if err != nil || entity.NameEn == nil || *entity.NameEn != changed.NameEn {
		t.Fatalf("失败出现部分写入: %#v %v", entity, err)
	}
	company, err = uc.UpdateOrganization(ctx, actor, system.ID, company.ID, "改中文名称", true, "CNY", "")
	if err != nil || company.NameEn != "" {
		t.Fatalf("清空英文名失败: %#v %v", company, err)
	}
	entity, err = data.db.Organization.Get(ctx, company.ID)
	if err != nil || entity.NameEn != nil || strings.Contains(entity.SearchKeywords, "New Official") {
		t.Fatalf("清空未持久化或残留旧检索: %#v %v", entity, err)
	}
	// 默认种子保留人工值，之后主动清空也不能再补回。
	seed := func() {
		t.Helper()
		if err := data.WithTx(ctx, func(tx *ent.Tx) error { _, err := CreateDefaultBranchCompanies(ctx, tx, system.ID); return err }); err != nil {
			t.Fatal(err)
		}
	}
	for _, seedCode := range []string{"BJ", "CD"} {
		item, err := data.db.Organization.Query().Where(organizationent.CodeEQ(seedCode)).Only(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if item.NameEn == nil {
			t.Fatalf("新建默认公司 %s 没有已确认英文名", seedCode)
		}
		manualName := "MANUAL " + seedCode + " CO"
		if _, err = data.db.Organization.UpdateOneID(item.ID).SetNameEn(manualName).Save(ctx); err != nil {
			t.Fatal(err)
		}
		seed()
		item, err = data.db.Organization.Get(ctx, item.ID)
		if err != nil || item.NameEn == nil || *item.NameEn != manualName {
			t.Fatalf("种子覆盖 %s 人工名: %#v %v", seedCode, item, err)
		}
		if _, err = data.db.Organization.UpdateOneID(item.ID).ClearNameEn().Save(ctx); err != nil {
			t.Fatal(err)
		}
		seed()
		item, err = data.db.Organization.Get(ctx, item.ID)
		if err != nil || item.NameEn != nil {
			t.Fatalf("种子覆盖 %s 主动清空: %#v %v", seedCode, item, err)
		}
	}
}

func TestHblIssuerNameEnUsesRealCompanyPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	fixture := newOrderPostgresFixture(t, data)
	if _, err := data.db.Organization.UpdateOneID(fixture.organizationID).SetNameEn("ORDER OWNER CO., LTD").Save(ctx); err != nil {
		t.Fatal(err)
	}
	order, err := fixture.newUsecase().Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewSeaDocumentRepo(data)
	docs, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if docs.HouseBill.PrintTemplate != biz.SeaHouseBillPrintTemplateRoncinSelf || docs.HouseBill.IssuerOrganizationNameEn != "ORDER OWNER CO., LTD" {
		t.Fatalf("任意公司未取得模板/名称: %#v", docs.HouseBill)
	}
	saved, err := repo.UpdateSeaHouseBill(ctx, fixture.organizationID, fixture.actorID, order.ID, docs.HouseBill.ID, docs.HouseBill.Version, docs.LinkVersion, &biz.SeaHouseBillInput{HouseNo: docs.HouseBill.HouseNo, IssuerSource: docs.HouseBill.IssuerSource, Content: docs.HouseBill.Content}, &biz.AuditEvent{UserID: &fixture.actorID, OrganizationID: &fixture.organizationID, Result: "success"})
	if err != nil || saved.IssuerOrganizationNameEn != "ORDER OWNER CO., LTD" {
		t.Fatalf("写后响应遗漏英文名: %#v %v", saved, err)
	}
	issuer, err := data.db.Organization.Create().SetCode("ACTUAL-ISSUER").SetName("真实签发公司").SetNameEn("ACTUAL ISSUER CO., LTD").SetKind(organizationent.KindCompany).SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = data.db.SeaHouseBill.UpdateOneID(saved.ID).SetIssuerOrganizationID(issuer.ID).Save(ctx); err != nil {
		t.Fatal(err)
	}
	before, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if before.HouseBill.IssuerOrganizationNameEn != "ACTUAL ISSUER CO., LTD" {
		t.Fatal("抬头误用订单或工作台公司")
	}
	if _, err = data.db.Organization.UpdateOneID(issuer.ID).SetNameEn("UPDATED ISSUER CO., LTD").SetEnabled(false).Save(ctx); err != nil {
		t.Fatal(err)
	}
	after, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.HouseBill.IssuerOrganizationNameEn != "UPDATED ISSUER CO., LTD" || after.HouseBill.Version != before.HouseBill.Version || after.HouseBill.PrintTemplate != biz.SeaHouseBillPrintTemplateRoncinSelf {
		t.Fatalf("主数据变更写入HBL或被新增停用限制: %#v", after.HouseBill)
	}
	if _, err = data.db.Organization.UpdateOneID(issuer.ID).ClearNameEn().Save(ctx); err != nil {
		t.Fatal(err)
	}
	missing, err := repo.GetSeaOrderDocuments(ctx, fixture.organizationID, order.ID)
	if err != nil || missing.HouseBill.IssuerOrganizationNameEn != "" || missing.HouseBill.PrintTemplate != biz.SeaHouseBillPrintTemplateRoncinSelf {
		t.Fatalf("缺名时不应阻止读取或伪造抬头: %#v %v", missing, err)
	}
}
