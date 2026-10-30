package data

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
)

// 独立维护入口退役只回收编辑权限，不删除订单创建链路保存的人员分工。
func TestOrderPersonnelEditorRetirementPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	fixture := newOrderPostgresFixture(t, data)
	created, err := fixture.newUsecase().Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
	if err != nil {
		t.Fatal(err)
	}
	repo := NewOrderPersonnelRepo(data)
	beforePersonnel, err := repo.List(ctx, fixture.organizationID, created.ID)
	if err != nil || len(beforePersonnel) != 4 {
		t.Fatalf("订单创建未保留创建人和三岗分工: count=%d err=%v", len(beforePersonnel), err)
	}
	readSnapshot := func(query string) string {
		t.Helper()
		var snapshot string
		if err := data.sqlDB.QueryRowContext(ctx, query, created.ID).Scan(&snapshot); err != nil {
			t.Fatal(err)
		}
		return snapshot
	}
	queries := map[string]string{
		"订单":   `SELECT row_to_json(o)::text FROM orders AS o WHERE id=$1`,
		"人员分工": `SELECT jsonb_agg(to_jsonb(p) ORDER BY p.id)::text FROM order_personnels AS p WHERE order_id=$1`,
	}
	snapshots := make(map[string]string, len(queries))
	for name, query := range queries {
		snapshots[name] = readSnapshot(query)
	}
	if _, err := SyncPermissionManifest(ctx, data.sqlDB); err != nil {
		t.Fatal(err)
	}
	liveKeys := []string{
		access.OrderPermission(access.OrderBusinessSE, access.OrderRead),
		access.OrderPermission(access.OrderBusinessSE, access.OrderCreate),
		access.OrderPermission(access.OrderBusinessSE, access.OrderPersonnelRead),
	}
	grants, err := data.db.Permission.Query().Where(permissionent.KeyIn(liveKeys...)).All(ctx)
	if err != nil || len(grants) != len(liveKeys) {
		t.Fatalf("准备相邻权限失败: count=%d err=%v", len(grants), err)
	}
	var retiredIDs []uuid.UUID
	for _, kind := range []string{"se", "si", "ae", "ai", "land", "rail"} {
		for _, operation := range []string{"assign", "remove"} {
			retired, err := data.db.Permission.Create().SetKey("business.order." + kind + ".personnel." + operation).
				SetName("退役人员维护权限").SetGroup("测试").SetDescription("独立维护入口退役前授权").Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			retiredIDs = append(retiredIDs, retired.ID)
			grants = append(grants, retired)
		}
	}
	role, err := data.db.Role.Create().SetOrganizationID(fixture.organizationID).
		SetCode("personnel-editor-retirement").SetName("保留人员读取角色").AddPermissions(grants...).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := role.Update().ClearPermissions().Exec(ctx); err != nil {
			t.Error(err)
		}
		if err := data.db.Role.DeleteOneID(role.ID).Exec(ctx); err != nil {
			t.Error(err)
		}
	})

	summary, err := SyncPermissionManifest(ctx, data.sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Removed != len(retiredIDs) {
		t.Fatalf("应精确回收12个独立维护权限: got=%d want=%d", summary.Removed, len(retiredIDs))
	}
	if count, err := data.db.Permission.Query().Where(permissionent.IDIn(retiredIDs...)).Count(ctx); err != nil || count != 0 {
		t.Fatalf("退役权限仍存在: count=%d err=%v", count, err)
	}
	if count, err := role.QueryPermissions().Count(ctx); err != nil || count != len(liveKeys) {
		t.Fatalf("退役授权未清理或相邻授权被移除: count=%d err=%v", count, err)
	}
	for _, key := range liveKeys {
		if exists, err := role.QueryPermissions().Where(permissionent.KeyEQ(key)).Exist(ctx); err != nil || !exists {
			t.Fatalf("相邻权限 %s 未保留: exists=%v err=%v", key, exists, err)
		}
	}
	second, err := SyncPermissionManifest(ctx, data.sqlDB)
	if err != nil {
		t.Fatal(err)
	}
	if second.Created != 0 || second.Updated != 0 || second.Removed != 0 || second.Attached != 0 {
		t.Fatalf("重复权限同步不幂等: %+v", second)
	}
	for name, query := range queries {
		if after := readSnapshot(query); after != snapshots[name] {
			t.Fatalf("回收独立维护权限改写了%s的整行数据", name)
		}
	}
	afterPersonnel, err := repo.List(ctx, fixture.organizationID, created.ID)
	if err != nil || !reflect.DeepEqual(beforePersonnel, afterPersonnel) {
		t.Fatalf("人员读取在独立维护入口退役后改变: err=%v", err)
	}
}
