package data

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	resourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	billent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

const businessTagRetirementVersion = "20261109100000_retire_business_tags"

// 带真实标签关联的正式迁移前态升级后，完整业务行、非标签资源与历史审计保持不变。
func TestBusinessTagRetirementUpgradePostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261109100000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	client := data.db
	org, err := client.Organization.Create().SetCode("TAG-RETIRE").SetName("标签退役公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := client.User.Create().SetDisplayName("保留员工").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	partner, err := client.Partner.Create().SetOrganizationID(org.ID).SetCode("KEEP").SetLegalName("保留客商").SetNormalizedName("保留客商").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	order, err := client.Order.Create().SetOrganizationID(org.ID).SetCustomerID(partner.ID).SetOrderNo("SE-TAG-RETIRE").SetIdempotencyKey(uuid.NewString()).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetPaymentTerm(orderent.PaymentTermPREPAID).SetVersion(7).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fee, err := client.OrderFee.Create().SetOrderID(order.ID).SetSettlementPartyID(partner.ID).SetIdempotencyKey(uuid.NewString()).SetDirection(feeent.DirectionRECEIVABLE).SetFeeCode("TEST").SetFeeName("保留海运费").SetBillingUnit("票").SetQuantity("1").SetUnitPrice("123.45").SetTotalAmount("123.45").SetNetAmount("123.45").SetTaxAmount("0").SetCurrency("CNY").SetExchangeRate("1").SetExchangeRateSource(feeent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-11-09").SetBaseCurrency("CNY").SetBaseCurrencyAmount("123.45").SetVersion(3).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	bill, err := withTestFinanceBillSettlementAccountSnapshot(client.FinanceBill.Create(), uuid.New(), "CNY").SetOrganizationID(org.ID).SetExchangeRate("1").SetExchangeRateSource(billent.ExchangeRateSourceSYSTEM).SetExchangeRateDate("2026-11-09").SetBillNo("KEEP-BILL").SetIdempotencyKey(uuid.NewString()).SetDirection(billent.DirectionRECEIVABLE).SetSettlementPartyID(partner.ID).SetSettlementPartyName(partner.LegalName).SetCurrency("CNY").SetBaseCurrency("CNY").SetTotalAmount("123.45").SetNetAmount("123.45").SetTaxAmount("0").SetBaseCurrencyAmount("123.45").SetFeeCount(1).SetBillDate("2026-11-09").SetVersion(4).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.FinanceBillLine.Create().SetBillID(bill.ID).SetOrderFeeID(fee.ID).SetOrderID(order.ID).SetOrderNo(order.OrderNo).SetQuantity(fee.Quantity).SetUnitPrice(fee.UnitPrice).SetFeeCode(fee.FeeCode).SetFeeName(fee.FeeName).SetTotalAmount(fee.TotalAmount).SetNetAmount(fee.NetAmount).SetTaxAmount(fee.TaxAmount).SetCurrency(fee.Currency).SetExchangeRate(fee.ExchangeRate).SetBaseCurrency(fee.BaseCurrency).SetBaseCurrencyAmount(fee.BaseCurrencyAmount).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.AuditLog.Create().SetOrganizationID(org.ID).SetUserID(actor.ID).SetAction("enterprise_resource.tag.create").SetResult(auditlog.ResultSuccess).SetResourceType("TAG").SetResourceID(uuid.NewString()).SetDetails([]byte(`{"历史事实":"保留标签审计"}`)).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tagID, groupID := uuid.New(), uuid.New()
	execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resources(id,created_at,updated_at,organization_id,resource_type,short_name) VALUES($1,now(),now(),$2,'TAG','待退役标签')`, tagID, org.ID)
	execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_tag_groups(id,created_at,updated_at,organization_id,name,normalized_name) VALUES($1,now(),now(),$2,'待退役组','待退役组')`, groupID, org.ID)
	execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_tags(id,created_at,updated_at,organization_id,resource_id,group_id,normalized_name) VALUES($1,now(),now(),$2,$3,$4,'待退役标签')`, uuid.New(), org.ID, tagID, groupID)
	for _, link := range []struct {
		table, owner string
		id           uuid.UUID
	}{{"order_enterprise_tags", "order_id", order.ID}, {"order_fee_enterprise_tags", "order_fee_id", fee.ID}, {"finance_bill_enterprise_tags", "finance_bill_id", bill.ID}} {
		execTagRetirementSQL(t, data.sqlDB, `INSERT INTO `+link.table+`(id,created_at,updated_at,organization_id,tag_resource_id,`+link.owner+`) VALUES($1,now(),now(),$2,$3,$4)`, uuid.New(), org.ID, tagID, link.id)
	}
	execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_partners(id,created_at,updated_at,resource_id,partner_id,resource_type) VALUES($1,now(),now(),$2,$3,'TAG')`, uuid.New(), tagID, partner.ID)
	for _, kind := range []resourceent.ResourceType{resourceent.ResourceTypeADDRESS, resourceent.ResourceTypeREMARK, resourceent.ResourceTypeIMAGE, resourceent.ResourceTypeSHIPPER, resourceent.ResourceTypeCONSIGNEE, resourceent.ResourceTypeNOTIFY_PARTY, resourceent.ResourceTypeENGLISH_CARGO_NAME, resourceent.ResourceTypeHS_CODE, resourceent.ResourceTypeMARKS} {
		resource, err := client.EnterpriseResource.Create().SetOrganizationID(org.ID).SetResourceType(kind).SetShortName(string(kind)).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_partners(id,created_at,updated_at,resource_id,partner_id,resource_type) VALUES($1,now(),now(),$2,$3,$4)`, uuid.New(), resource.ID, partner.ID, string(kind))
		switch kind {
		case resourceent.ResourceTypeADDRESS:
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_addresses(id,created_at,updated_at,resource_id,country_code,address_detail) VALUES($1,now(),now(),$2,'CN','保留地址')`, uuid.New(), resource.ID)
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_assignees(id,created_at,updated_at,resource_id,user_id) VALUES($1,now(),now(),$2,$3)`, uuid.New(), resource.ID, actor.ID)
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_address_types(id,created_at,updated_at,resource_id,address_type) VALUES($1,now(),now(),$2,'PICKUP')`, uuid.New(), resource.ID)
		case resourceent.ResourceTypeREMARK:
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_remarks(id,created_at,updated_at,resource_id,remark_type,content) VALUES($1,now(),now(),$2,'BOOKING','保留备注')`, uuid.New(), resource.ID)
		case resourceent.ResourceTypeIMAGE:
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_images(id,created_at,updated_at,resource_id,file_name,mime_type,file_size,object_key,checksum,uploaded_by) VALUES($1,now(),now(),$2,'保留图片.png','image/png',10,'keep/image','checksum',$3)`, uuid.New(), resource.ID, actor.ID)
		case resourceent.ResourceTypeSHIPPER, resourceent.ResourceTypeCONSIGNEE, resourceent.ResourceTypeNOTIFY_PARTY:
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_parties(id,created_at,updated_at,resource_id,organization_id,resource_type,company_name) VALUES($1,now(),now(),$2,$3,$4,'保留单证主体')`, uuid.New(), resource.ID, org.ID, string(kind))
		default:
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resource_shipping_texts(id,created_at,updated_at,resource_id,content) VALUES($1,now(),now(),$2,'保留出货文本')`, uuid.New(), resource.ID)
		}
	}
	if _, err := SyncPermissionManifest(ctx, data.sqlDB); err != nil {
		t.Fatal(err)
	}
	retired, err := client.Permission.Create().SetKey("system.finance.fee.tag").SetName("旧标签权限").SetGroup("旧财务标签").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	liveKeys := []string{access.FinanceBillRead, access.FinanceBillUpdate, access.EnterpriseResourceRead, access.EnterpriseResourceUpdate, access.OrderPermission(access.OrderBusinessSE, access.OrderRead), access.OrderPermission(access.OrderBusinessSE, access.OrderUpdate), access.OrderPermission(access.OrderBusinessSE, access.OrderFeeRead), access.OrderPermission(access.OrderBusinessSE, access.OrderFeeUpdate)}
	grants, err := client.Permission.Query().Where(permissionent.KeyIn(liveKeys...)).All(ctx)
	if err != nil || len(grants) != len(liveKeys) {
		t.Fatalf("保留权限准备失败: count=%d err=%v", len(grants), err)
	}
	role, err := client.Role.Create().SetOrganizationID(org.ID).SetCode("operator").SetName("保留角色").AddPermissions(append(grants, retired)...).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	preserved := []string{"organizations", "users", "partners", "orders", "order_fees", "finance_bills", "finance_bill_lines", "audit_logs", "roles", "enterprise_resources", "enterprise_resource_partners", "enterprise_resource_addresses", "enterprise_resource_remarks", "enterprise_resource_images", "enterprise_resource_parties", "enterprise_resource_shipping_texts", "enterprise_resource_assignees", "enterprise_resource_address_types"}
	before := tagRetirementSnapshots(t, data.sqlDB, preserved, true)
	for attempt := 0; attempt < 2; attempt++ {
		err := migration.ApplyWithOptions(ctx, data.sqlDB, filepath.Join("..", "..", "migrations"), migration.Options{PostStep: func(conn *sql.Conn) error { _, err := SyncPermissionManifest(ctx, conn); return err }})
		if err != nil {
			t.Fatal(err)
		}
		assertBusinessTagsRetired(t, data)
		after := tagRetirementSnapshots(t, data.sqlDB, preserved, false)
		for table, snapshot := range before {
			if after[table] != snapshot {
				t.Fatalf("退役迁移改写非目标表 %s", table)
			}
		}
		if exists, err := client.Permission.Query().Where(permissionent.IDEQ(retired.ID)).Exist(ctx); err != nil || exists {
			t.Fatalf("旧权限未清理: exists=%v err=%v", exists, err)
		}
		for _, key := range liveKeys {
			if exists, err := role.QueryPermissions().Where(permissionent.KeyEQ(key)).Exist(ctx); err != nil || !exists {
				t.Fatalf("有效授权 %s 被删除: %v", key, err)
			}
		}
		if count, err := role.QueryPermissions().Count(ctx); err != nil || count != len(liveKeys) {
			t.Fatalf("授权计数错误 count=%d err=%v", count, err)
		}
	}
}

func TestBusinessTagRetirementRejectsInvalidReferencesPostgres(t *testing.T) {
	for _, scenario := range []struct{ name, statement, want string }{
		{"标签被备注引用", `INSERT INTO enterprise_resource_remarks(id,created_at,updated_at,resource_id,remark_type,content) VALUES($1,now(),now(),$2,'BOOKING','异常标签备注')`, "非标签明细引用"},
		{"冗余标签类型指向备注", `INSERT INTO enterprise_resource_partners(id,created_at,updated_at,resource_id,partner_id,resource_type) VALUES($1,now(),now(),$2,$3,'TAG')`, "资源类型不一致"},
		{"标签资源客商关联标成备注", `INSERT INTO enterprise_resource_partners(id,created_at,updated_at,resource_id,partner_id,resource_type) VALUES($1,now(),now(),$2,$3,'REMARK')`, "资源类型不一致"},
		{"未知表引用标签组", "", "2BP01"},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			data, cleanup := getIntegrationDataBeforeMigration(t, "20261109100000")
			t.Cleanup(cleanup)
			ctx := context.Background()
			org, err := data.db.Organization.Create().SetCode("INVALID").SetName("异常测试").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			tagID, groupID := uuid.New(), uuid.New()
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_resources(id,created_at,updated_at,organization_id,resource_type,short_name) VALUES($1,now(),now(),$2,'TAG','待保留异常标签')`, tagID, org.ID)
			execTagRetirementSQL(t, data.sqlDB, `INSERT INTO enterprise_tag_groups(id,created_at,updated_at,organization_id,name,normalized_name) VALUES($1,now(),now(),$2,'异常组','异常组')`, groupID, org.ID)
			switch scenario.name {
			case "冗余标签类型指向备注", "标签资源客商关联标成备注":
				partner, err := data.db.Partner.Create().SetOrganizationID(org.ID).SetLegalName("异常关联客商").SetNormalizedName("异常关联客商").Save(ctx)
				if err != nil {
					t.Fatal(err)
				}
				resource, err := data.db.EnterpriseResource.Create().SetOrganizationID(org.ID).SetResourceType(resourceent.ResourceTypeREMARK).SetShortName("保留备注").Save(ctx)
				if err != nil {
					t.Fatal(err)
				}
				targetID := resource.ID
				if scenario.name == "标签资源客商关联标成备注" {
					targetID = tagID
				}
				execTagRetirementSQL(t, data.sqlDB, scenario.statement, uuid.New(), targetID, partner.ID)
			case "未知表引用标签组":
				execTagRetirementSQL(t, data.sqlDB, `CREATE TABLE unknown_tag_reference(group_id uuid REFERENCES enterprise_tag_groups(id))`)
				execTagRetirementSQL(t, data.sqlDB, `INSERT INTO unknown_tag_reference(group_id) VALUES($1)`, groupID)
			default:
				execTagRetirementSQL(t, data.sqlDB, scenario.statement, uuid.New(), tagID)
			}
			tables := []string{"enterprise_resources", "enterprise_tag_groups", "enterprise_tags", "order_enterprise_tags", "order_fee_enterprise_tags", "finance_bill_enterprise_tags", "enterprise_resource_partners", "enterprise_resource_remarks"}
			before := tagRetirementSnapshots(t, data.sqlDB, tables, false)
			err = migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations"))
			if err == nil || !strings.Contains(err.Error(), scenario.want) {
				t.Fatalf("异常引用必须拒绝迁移: %v", err)
			}
			after := tagRetirementSnapshots(t, data.sqlDB, tables, false)
			for table, snapshot := range before {
				if after[table] != snapshot {
					t.Fatalf("失败迁移未回滚 %s", table)
				}
			}
			var revisions int
			if err := data.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version=$1`, businessTagRetirementVersion).Scan(&revisions); err != nil || revisions != 0 {
				t.Fatalf("失败迁移不得录入版本: count=%d err=%v", revisions, err)
			}
		})
	}
}

func TestBusinessTagRetirementColdStartPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	assertBusinessTagsRetired(t, data)
}

func execTagRetirementSQL(t *testing.T, db *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), query, args...); err != nil {
		t.Fatal(err)
	}
}

func tagRetirementSnapshots(t *testing.T, db *sql.DB, tables []string, excludeTags bool) map[string]string {
	t.Helper()
	snapshots := make(map[string]string, len(tables))
	for _, table := range tables {
		filter := ""
		if excludeTags {
			switch table {
			case "enterprise_resources":
				filter = " WHERE t.resource_type <> 'TAG'"
			case "enterprise_resource_partners":
				filter = " WHERE t.resource_id IN (SELECT id FROM enterprise_resources WHERE resource_type <> 'TAG')"
			}
		}
		var snapshot string
		if err := db.QueryRowContext(context.Background(), `SELECT COALESCE(jsonb_agg(to_jsonb(t) ORDER BY id),'[]'::jsonb)::text FROM "`+table+`" t`+filter).Scan(&snapshot); err != nil {
			t.Fatal(err)
		}
		snapshots[table] = snapshot
	}
	return snapshots
}

func assertBusinessTagsRetired(t *testing.T, data *Data) {
	t.Helper()
	for _, table := range []string{"enterprise_tag_groups", "enterprise_tags", "order_enterprise_tags", "order_fee_enterprise_tags", "finance_bill_enterprise_tags"} {
		var exists bool
		if err := data.sqlDB.QueryRowContext(context.Background(), `SELECT to_regclass(current_schema()||'.'||$1) IS NOT NULL`, table).Scan(&exists); err != nil || exists {
			t.Fatalf("退役表 %s 仍存在: %v", table, err)
		}
	}
	var count int
	if err := data.sqlDB.QueryRowContext(context.Background(), `SELECT count(*) FROM enterprise_resources WHERE resource_type='TAG'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("标签资源未清理: count=%d err=%v", count, err)
	}
	if err := data.sqlDB.QueryRowContext(context.Background(), `SELECT count(*) FROM enterprise_resource_partners WHERE resource_type='TAG'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("标签客商关联未清理: count=%d err=%v", count, err)
	}
}

// 编译期持续保证标签退役不改变保留的企业资源接口实现。
var _ biz.EnterpriseResourceRepo = (*enterpriseResourceRepo)(nil)
