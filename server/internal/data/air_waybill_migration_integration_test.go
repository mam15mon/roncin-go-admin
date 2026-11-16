package data

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

const airContentMigration = "20261116140000_air_waybill_content_and_mode.sql"

type airContentMigrationFixture struct {
	d                                                        *Data
	f                                                        *airExportFixture
	orderID, masterID, houseID, linkID, versionID, historyID uuid.UUID
	oldContent                                               string
	masterHistory                                            []byte
	migrationDir                                             string
}

func newAirContentMigrationFixture(t *testing.T) *airContentMigrationFixture {
	t.Helper()
	d, cleanup := getIntegrationDataBeforeMigration(t, airContentMigration)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	o := mustMigrationFixture(insertMigrationFixture[ent.Order](ctx, d, "orders", d.db.Order.Create().SetOrganizationID(f.organizationID).SetOrderNo("AIR-MIGRATION").SetIdempotencyKey(uuid.NewString()).SetCustomerID(f.partnerID).SetBusinessType(orderent.BusinessTypeAE).SetTradeDirection(orderent.TradeDirectionExport).SetPaymentTerm(orderent.PaymentTermPREPAID).Mutation()))
	x := &airContentMigrationFixture{d: d, f: f, orderID: o.ID, masterID: uuid.New(), houseID: uuid.New(), linkID: uuid.New(), versionID: uuid.New(), historyID: uuid.New(), oldContent: "  旧正文\n\"引号\"\\反斜杠<>&\u2028\u2029" + strings.Repeat("长", 3500)}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, e := d.sqlDB.ExecContext(ctx, q, args...); e != nil {
			t.Fatal(e)
		}
	}
	exec(`INSERT INTO air_master_waybills(id,created_at,updated_at,business_type,organization_id,master_no,airline_id,origin_airport_id,destination_airport_id,version,notes) VALUES($1,'2026-10-01T01:02:03Z','2026-10-02T01:02:03Z','AE',$2,'78112345675',$3,$4,$5,7,'运输备注独立保留')`, x.masterID, f.organizationID, f.airlineID, f.originID, f.destinationID)
	exec(`INSERT INTO air_master_waybill_order_links(id,created_at,updated_at,organization_id,master_waybill_id,order_id,status,started_at,version) VALUES($1,'2026-10-01T01:02:03Z','2026-10-02T01:02:03Z',$2,$3,$4,'ACTIVE','2026-10-01T01:02:03Z',4)`, x.linkID, f.organizationID, x.masterID, o.ID)
	exec(`INSERT INTO air_house_waybills(id,created_at,updated_at,organization_id,master_waybill_id,order_id,house_no,status,content,version) VALUES($1,'2026-10-01T01:02:03Z','2026-10-02T01:02:03Z',$2,$3,$4,'H-OLD','RELEASED',$5,3)`, x.houseID, f.organizationID, x.masterID, o.ID, x.oldContent)
	exec(`INSERT INTO air_house_waybill_versions(id,created_at,organization_id,order_id,house_waybill_id,version,house_no,status,content,actor_id,reason) VALUES($1,'2026-10-01T01:02:03Z',$2,$3,$4,3,'H-OLD','RELEASED',$5,$6,'旧不可变原因')`, x.versionID, f.organizationID, o.ID, x.houseID, x.oldContent, f.actorID)
	x.masterHistory = []byte(`{"MasterNo":"78112345675","Notes":"原始主单历史\n保留","Status":"DRAFT"}`)
	exec(`INSERT INTO air_master_waybill_histories(id,created_at,updated_at,organization_id,master_waybill_id,version,action,reason,actor_id,content) VALUES($1,'2026-10-01T01:02:03Z','2026-10-02T01:02:03Z',$2,$3,7,'CREATE','旧主单历史',$4,$5)`, x.historyID, f.organizationID, x.masterID, f.actorID, x.masterHistory)
	// 只验收本组迁移；后续组演进不改变此处历史保留的比较边界。
	x.migrationDir = t.TempDir()
	files, e := filepath.Glob(filepath.Join("..", "..", "migrations", "*.sql"))
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range files {
		if filepath.Base(file) > airContentMigration {
			continue
		}
		b, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(x.migrationDir, filepath.Base(file)), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	return x
}
func airMigrationSnapshot(t *testing.T, d *Data, table, omitted string) string {
	t.Helper()
	var got string
	q := `SELECT coalesce(jsonb_agg(to_jsonb(r)` + omitted + ` ORDER BY id),'[]'::jsonb)::text FROM ` + table + ` r`
	if e := d.sqlDB.QueryRowContext(context.Background(), q).Scan(&got); e != nil {
		t.Fatal(e)
	}
	return got
}

func TestAirWaybillMigrationPreservesTextAndHistoryPostgres(t *testing.T) {
	x := newAirContentMigrationFixture(t)
	ctx := context.Background()
	before := map[string]string{}
	for table, omit := range map[string]string{"air_master_waybills": "-'waybill_content'-'content_hash'", "air_master_waybill_order_links": "-'document_mode'", "air_house_waybills": "-'content'-'content_hash'", "air_house_waybill_versions": "-'content'-'content_hash'", "air_master_waybill_histories": "-'content_hash'", "orders": ""} {
		before[table] = airMigrationSnapshot(t, x.d, table, omit)
	}
	for range 2 {
		if e := migration.Apply(ctx, x.d.sqlDB, x.migrationDir); e != nil {
			t.Fatal(e)
		}
	}
	for table, omit := range map[string]string{"air_master_waybills": "-'waybill_content'-'content_hash'", "air_master_waybill_order_links": "-'document_mode'", "air_house_waybills": "-'content'-'content_hash'", "air_house_waybill_versions": "-'content'-'content_hash'", "air_master_waybill_histories": "-'content_hash'", "orders": ""} {
		if got := airMigrationSnapshot(t, x.d, table, omit); got != before[table] {
			t.Fatalf("迁移改写%s既有列", table)
		}
	}
	house := x.d.db.AirHouseWaybill.GetX(ctx, x.houseID)
	history := x.d.db.AirHouseWaybillVersion.GetX(ctx, x.versionID)
	master := x.d.db.AirMasterWaybill.GetX(ctx, x.masterID)
	masterHistory := x.d.db.AirMasterWaybillHistory.GetX(ctx, x.historyID)
	for _, content := range [][]byte{house.Content, history.Content} {
		var v biz.AirWaybillContent
		if e := json.Unmarshal(content, &v); e != nil || v.Remarks != x.oldContent {
			t.Fatalf("迁移丢失旧文本：%v", e)
		}
	}
	expected := biz.AirWaybillContentHash(&biz.AirWaybillContent{Remarks: x.oldContent})
	if house.ContentHash != expected || history.ContentHash != expected {
		t.Fatalf("SQL迁移哈希与Go不一致：%s %s %s", house.ContentHash, history.ContentHash, expected)
	}
	if master.Notes != "运输备注独立保留" || master.ContentHash != biz.AirWaybillContentHash(nil) || string(masterHistory.Content) != string(x.masterHistory) || masterHistory.ContentHash != biz.AirWaybillContentHash(nil) {
		t.Fatal("迁移串改运输备注或主单历史原载体")
	}
	if x.d.db.AirMasterWaybillOrderLink.GetX(ctx, x.linkID).DocumentMode != "HOUSE" {
		t.Fatal("有效旧关系未回填HOUSE")
	}
	var revisions int
	if e := x.d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM schema_migrations WHERE version=$1`, strings.TrimSuffix(airContentMigration, ".sql")).Scan(&revisions); e != nil || revisions != 1 {
		t.Fatalf("迁移重复执行错误：%d %v", revisions, e)
	}
}

func TestAirWaybillMigrationStopsInvalidLegacyRelationsPostgres(t *testing.T) {
	for _, scenario := range []string{"缺少分单", "历史订单不符", "历史号码不符"} {
		t.Run(scenario, func(t *testing.T) {
			x := newAirContentMigrationFixture(t)
			ctx := context.Background()
			switch scenario {
			case "缺少分单":
				if _, e := x.d.sqlDB.ExecContext(ctx, `DELETE FROM air_house_waybill_versions`); e != nil {
					t.Fatal(e)
				}
				if _, e := x.d.sqlDB.ExecContext(ctx, `DELETE FROM air_house_waybills`); e != nil {
					t.Fatal(e)
				}
			case "历史订单不符":
				if _, e := x.d.sqlDB.ExecContext(ctx, `UPDATE air_house_waybill_versions SET order_id=$1`, uuid.New()); e != nil {
					t.Fatal(e)
				}
			case "历史号码不符":
				if _, e := x.d.sqlDB.ExecContext(ctx, `UPDATE air_house_waybill_versions SET house_no='H-WRONG'`); e != nil {
					t.Fatal(e)
				}
			}
			before := airMigrationSnapshot(t, x.d, "air_house_waybills", "")
			if e := migration.Apply(ctx, x.d.sqlDB, x.migrationDir); e == nil || !strings.Contains(e.Error(), "停止模式/正文迁移") {
				t.Fatalf("异常存量未显式停止：%v", e)
			}
			if got := airMigrationSnapshot(t, x.d, "air_house_waybills", ""); got != before {
				t.Fatal("失败迁移部分改写正文")
			}
			var columns int
			if e := x.d.sqlDB.QueryRowContext(ctx, `SELECT count(*) FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='air_master_waybill_order_links' AND column_name='document_mode'`).Scan(&columns); e != nil || columns != 0 {
				t.Fatalf("失败迁移部分提交DDL：%d %v", columns, e)
			}
		})
	}
}
