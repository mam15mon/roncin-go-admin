package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

const orderMasterDirectionMigration = "20261116120000_order_master_business_identity.sql"

// 前态只执行正式迁移前缀；主单用当时的 SQL 列形构造，不删除现行约束。
type orderDirectionMigrationFixture struct {
	t                                         *testing.T
	data                                      *Data
	organizationID, partnerID, shippingLineID uuid.UUID
	airlineID, originID, destinationID        uuid.UUID
	migrationDir                              string
}

func newOrderDirectionMigrationFixture(t *testing.T) *orderDirectionMigrationFixture {
	t.Helper()
	d, cleanup := getIntegrationDataBeforeMigration(t, orderMasterDirectionMigration)
	t.Cleanup(cleanup)
	// 全行不变断言只执行至本次迁移，后续内容/模式演进由各自迁移测试验证。
	migrationDir := t.TempDir()
	files, err := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) > orderMasterDirectionMigration {
			continue
		}
		body, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(migrationDir, filepath.Base(file)), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	org := d.db.Organization.Create().SetCode("DIRECTION-COMPANY").SetName("方向迁移测试公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	partner := d.db.Partner.Create().SetOrganizationID(org.ID).SetCode("DIRECTION-PARTNER").SetLegalName("方向迁移测试客户").SetNormalizedName("方向迁移测试客户").SaveX(ctx)
	line := d.db.ShippingLine.Create().SetScacCode("DRCT").SetNameZh("方向迁移测试船公司").SetNameEn("Direction Carrier").SetCountryCode("CN").SaveX(ctx)
	airline := d.db.Airline.Create().SetIataCode("DZ").SetNameEn("Direction Airline").SetCountryCode("CN").SaveX(ctx)
	origin := d.db.Airport.Create().SetIataCode("DZO").SetNameEn("Direction Origin").SetCountryCode("CN").SaveX(ctx)
	destination := d.db.Airport.Create().SetIataCode("DZD").SetNameEn("Direction Destination").SetCountryCode("US").SaveX(ctx)
	return &orderDirectionMigrationFixture{t: t, data: d, migrationDir: migrationDir, organizationID: org.ID, partnerID: partner.ID, shippingLineID: line.ID, airlineID: airline.ID, originID: origin.ID, destinationID: destination.ID}
}

func (f *orderDirectionMigrationFixture) order(businessType, direction string) uuid.UUID {
	f.t.Helper()
	return f.data.db.Order.Create().SetOrganizationID(f.organizationID).SetOrderNo(uuid.NewString()).SetIdempotencyKey(uuid.NewString()).SetCustomerID(f.partnerID).
		SetBusinessType(orderent.BusinessType(businessType)).SetTradeDirection(orderent.TradeDirection(direction)).SetPaymentTerm(orderent.PaymentTermPREPAID).SaveX(context.Background()).ID
}

func (f *orderDirectionMigrationFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.data.sqlDB.ExecContext(context.Background(), query, args...); err != nil {
		f.t.Fatal(err)
	}
}

func (f *orderDirectionMigrationFixture) seaMaster(no string) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO sea_master_bills(id,created_at,updated_at,organization_id,shipping_line_id,master_no,normalized_master_no,status,version,shipper_text)
		VALUES($1,'2026-10-01T01:02:03Z','2026-10-02T01:02:03Z',$2,$3,$4,$4,'DRAFT',7,'迁移前海运长文本与换行保留')`, id, f.organizationID, f.shippingLineID, no)
	return id
}

func (f *orderDirectionMigrationFixture) airMaster(no string) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO air_master_waybills(id,created_at,updated_at,organization_id,airline_id,origin_airport_id,destination_airport_id,master_no,version,notes)
		VALUES($1,'2026-10-01T01:02:03Z','2026-10-02T01:02:03Z',$2,$3,$4,$5,$6,8,'迁移前空运备注完整保留')`, id, f.organizationID, f.airlineID, f.originID, f.destinationID, no)
	return id
}

func (f *orderDirectionMigrationFixture) link(kind string, masterID, orderID uuid.UUID, ended bool) {
	f.t.Helper()
	status := "ACTIVE"
	var endedAt any
	if ended {
		status, endedAt = "ENDED", "2026-10-03T01:02:03Z"
	}
	if kind == "sea" {
		voyage := f.data.db.SeaTransportExecution.Create().SetOrganizationID(f.organizationID).SetShippingLineID(f.shippingLineID).SetVesselName("迁移前航程").SaveX(context.Background())
		f.exec(`INSERT INTO sea_master_bill_order_links(id,created_at,updated_at,organization_id,master_bill_id,order_id,transport_execution_id,document_structure,status,started_at,ended_at,ended_reason,version)
			VALUES($1,now(),now(),$2,$3,$4,$5,'HOUSE',$6,'2026-10-01T01:02:03Z',$7,'迁移前历史关系',3)`, uuid.New(), f.organizationID, masterID, orderID, voyage.ID, status, endedAt)
	} else {
		f.exec(`INSERT INTO air_master_waybill_order_links(id,created_at,updated_at,organization_id,master_waybill_id,order_id,status,started_at,ended_at,ended_reason,version)
			VALUES($1,now(),now(),$2,$3,$4,$5,'2026-10-01T01:02:03Z',$6,'迁移前历史关系',3)`, uuid.New(), f.organizationID, masterID, orderID, status, endedAt)
	}
}

func (f *orderDirectionMigrationFixture) seaVersion(masterID uuid.UUID) uuid.UUID {
	f.t.Helper()
	id := uuid.New()
	f.exec(`INSERT INTO sea_master_bill_versions(id,created_at,organization_id,master_bill_id,shipping_line_id,version_no,source_entity_version,master_no,normalized_master_no,status,content_hash,source,shipper_text)
		SELECT $1,'2026-10-01T01:02:03Z',organization_id,id,shipping_line_id,1,version,master_no,normalized_master_no,status,'历史内容指纹','ORDER_LOCK','原始不可变版本正文' FROM sea_master_bills WHERE id=$2`, id, masterID)
	return id
}

func (f *orderDirectionMigrationFixture) snapshot(table string) string {
	f.t.Helper()
	var value string
	// table 只取本文件固定表名；移除新增身份后比较全部原始列，包含时间、版本和正文。
	query := fmt.Sprintf(`SELECT coalesce(jsonb_agg(to_jsonb(r)-'business_type' ORDER BY id),'[]'::jsonb)::text FROM %s r`, table)
	if err := f.data.sqlDB.QueryRowContext(context.Background(), query).Scan(&value); err != nil {
		f.t.Fatal(err)
	}
	return value
}

func TestOrderMasterDirectionMigrationPreservesHistoricalRowsPostgres(t *testing.T) {
	f := newOrderDirectionMigrationFixture(t)
	types := []struct{ kind, businessType, direction, no string }{
		{"sea", "SE", "export", "SEA-EXPORT"}, {"sea", "SI", "import", "SEA-IMPORT"},
		{"air", "AE", "export", "78112345678"}, {"air", "AI", "import", "78187654321"},
	}
	want := map[uuid.UUID]string{}
	for _, item := range types {
		var masterID uuid.UUID
		if item.kind == "sea" {
			masterID = f.seaMaster(item.no)
			want[f.seaVersion(masterID)] = item.businessType
		} else {
			masterID = f.airMaster(item.no)
		}
		want[masterID] = item.businessType
		// 进口仅保留已结束关系，证明身份不依赖当前活动成员。
		f.link(item.kind, masterID, f.order(item.businessType, item.direction), item.direction == "import")
	}
	tables := []string{"sea_master_bills", "air_master_waybills", "sea_master_bill_versions", "sea_master_bill_order_links", "air_master_waybill_order_links", "orders", "sea_transport_executions"}
	before := map[string]string{}
	for _, table := range tables {
		before[table] = f.snapshot(table)
	}
	for range 2 {
		if err := migration.Apply(context.Background(), f.data.sqlDB, f.migrationDir); err != nil {
			t.Fatal(err)
		}
		for _, table := range tables {
			if got := f.snapshot(table); got != before[table] {
				t.Fatalf("方向迁移或重放改写了 %s 原有列: before=%s after=%s", table, before[table], got)
			}
		}
	}
	rows, err := f.data.sqlDB.QueryContext(context.Background(), `SELECT id,business_type FROM sea_master_bills UNION ALL SELECT id,business_type FROM air_master_waybills UNION ALL SELECT id,business_type FROM sea_master_bill_versions`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var actual string
		if err := rows.Scan(&id, &actual); err != nil {
			t.Fatal(err)
		}
		if actual != want[id] {
			t.Fatalf("未按全部历史关联回填方向: id=%s got=%s want=%s", id, actual, want[id])
		}
		delete(want, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(want) != 0 {
		t.Fatalf("迁移丢失主单或历史版本: %v", want)
	}
	var revisions int
	if err := f.data.sqlDB.QueryRowContext(context.Background(), `SELECT count(*) FROM schema_migrations WHERE version=$1`, strings.TrimSuffix(orderMasterDirectionMigration, ".sql")).Scan(&revisions); err != nil || revisions != 1 {
		t.Fatalf("重复正式迁移应只记录一次: count=%d err=%v", revisions, err)
	}
}

func TestOrderMasterDirectionMigrationRejectsUnclassifiableRowsPostgres(t *testing.T) {
	for _, kind := range []string{"sea", "air"} {
		scenarios := []string{"孤立主单", "未知类型", "混合进出口", "类型方向不符", "跨公司订单", "跨公司关联"}
		if kind == "sea" {
			scenarios = append(scenarios, "历史版本公司不符")
		}
		for _, scenario := range scenarios {
			t.Run(kind+"/"+scenario, func(t *testing.T) {
				f := newOrderDirectionMigrationFixture(t)
				exportType, importType, table := "SE", "SI", "sea_master_bills"
				var masterID uuid.UUID
				if kind == "sea" {
					masterID = f.seaMaster("SEA-INVALID")
				} else {
					exportType, importType, table = "AE", "AI", "air_master_waybills"
					masterID = f.airMaster("78112345678")
				}
				if scenario != "孤立主单" {
					businessType, direction := exportType, "export"
					if scenario == "未知类型" {
						businessType = "LAND"
					}
					if scenario == "类型方向不符" {
						direction = "import"
					}
					orderID := f.order(businessType, direction)
					f.link(kind, masterID, orderID, true)
					if scenario == "混合进出口" {
						f.link(kind, masterID, f.order(importType, "import"), false)
					}
					if scenario == "历史版本公司不符" {
						versionID := f.seaVersion(masterID)
						other := f.data.db.Organization.Create().SetCode("VERSION-COMPANY").SetName("历史版本其他公司").SetKind("company").SetBaseCurrency("CNY").SaveX(context.Background())
						f.exec(`UPDATE sea_master_bill_versions SET organization_id=$1 WHERE id=$2`, other.ID, versionID)
					}
					if strings.HasPrefix(scenario, "跨公司") {
						other := f.data.db.Organization.Create().SetCode("OTHER-COMPANY").SetName("其他公司").SetKind("company").SetBaseCurrency("CNY").SaveX(context.Background())
						if scenario == "跨公司订单" {
							f.exec(`UPDATE orders SET organization_id=$1 WHERE id=$2`, other.ID, orderID)
						} else if kind == "sea" {
							f.exec(`UPDATE sea_master_bill_order_links SET organization_id=$1 WHERE master_bill_id=$2`, other.ID, masterID)
						} else {
							f.exec(`UPDATE air_master_waybill_order_links SET organization_id=$1 WHERE master_waybill_id=$2`, other.ID, masterID)
						}
					}
				}
				before := f.snapshot(table)
				err := migration.Apply(context.Background(), f.data.sqlDB, f.migrationDir)
				if err == nil || !strings.Contains(err.Error(), "主单") {
					t.Fatalf("异常历史应显式拒绝而非默认出口: %v", err)
				}
				if f.snapshot(table) != before {
					t.Fatal("失败迁移留下半完成回填")
				}
				var columns, revisions int
				if err := f.data.sqlDB.QueryRowContext(context.Background(), `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name IN ('sea_master_bills','air_master_waybills','sea_master_bill_versions') AND column_name='business_type'`).Scan(&columns); err != nil || columns != 0 {
					t.Fatalf("失败迁移没有整体回滚列: count=%d err=%v", columns, err)
				}
				if err := f.data.sqlDB.QueryRowContext(context.Background(), `SELECT count(*) FROM schema_migrations WHERE version=$1`, strings.TrimSuffix(orderMasterDirectionMigration, ".sql")).Scan(&revisions); err != nil || revisions != 0 {
					t.Fatalf("失败迁移不应记录 revision: count=%d err=%v", revisions, err)
				}
			})
		}
	}
}

func TestOrderMasterDirectionDatabaseConstraintsPostgres(t *testing.T) {
	f := newOrderDirectionMigrationFixture(t)
	seaID, airID := f.seaMaster("SAME-NUMBER"), f.airMaster("78112345678")
	f.link("sea", seaID, f.order("SE", "export"), false)
	f.link("air", airID, f.order("AE", "export"), false)
	versionID := f.seaVersion(seaID)
	if err := migration.Apply(context.Background(), f.data.sqlDB, f.migrationDir); err != nil {
		t.Fatal(err)
	}
	other := f.data.db.Organization.Create().SetCode("SECOND-COMPANY").SetName("第二公司").SetKind("company").SetBaseCurrency("CNY").SaveX(context.Background())
	for _, item := range []struct {
		table, importType, invalidType string
		id                             uuid.UUID
	}{
		{"sea_master_bills", "SI", "AE", seaID}, {"air_master_waybills", "AI", "SE", airID},
	} {
		t.Run(item.table, func(t *testing.T) {
			clone := func(patch map[string]any, wantSQLState string) {
				t.Helper()
				patch["id"] = uuid.NewString()
				body, err := json.Marshal(patch)
				if err != nil {
					t.Fatal(err)
				}
				query := fmt.Sprintf(`INSERT INTO %s SELECT (jsonb_populate_record(NULL::%s,to_jsonb(r)||$2::jsonb)).* FROM %s r WHERE id=$1`, item.table, item.table, item.table)
				_, err = f.data.sqlDB.ExecContext(context.Background(), query, item.id, body)
				if wantSQLState == "" {
					if err != nil {
						t.Fatalf("相同号码应按方向或公司独立: %v", err)
					}
					return
				}
				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) || pgErr.Code != wantSQLState {
					t.Fatalf("数据库约束未拒绝，want SQLSTATE=%s got=%v", wantSQLState, err)
				}
			}
			clone(map[string]any{"business_type": item.importType}, "")
			clone(map[string]any{"organization_id": other.ID}, "")
			clone(map[string]any{}, "23505")
			clone(map[string]any{"business_type": item.invalidType}, "23514")
			clone(map[string]any{"business_type": nil}, "23502")
		})
	}
	for _, table := range []string{"sea_master_bills", "air_master_waybills", "sea_master_bill_versions"} {
		var nullable string
		var defaultValue *string
		if err := f.data.sqlDB.QueryRowContext(context.Background(), `SELECT is_nullable,column_default FROM information_schema.columns WHERE table_schema=current_schema() AND table_name=$1 AND column_name='business_type'`, table).Scan(&nullable, &defaultValue); err != nil || nullable != "NO" || defaultValue != nil {
			t.Fatalf("身份列须必填且无出口默认值: table=%s nullable=%s default=%v err=%v", table, nullable, defaultValue, err)
		}
	}
	for _, invalidType := range []any{nil, "AE"} {
		_, err := f.data.sqlDB.ExecContext(context.Background(), `UPDATE sea_master_bill_versions SET business_type=$1 WHERE id=$2`, invalidType, versionID)
		wantSQLState := "23514"
		if invalidType == nil {
			wantSQLState = "23502"
		}
		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) || pgErr.Code != wantSQLState {
			t.Fatalf("历史版本身份约束未拒绝: want=%s err=%v", wantSQLState, err)
		}
	}
}
