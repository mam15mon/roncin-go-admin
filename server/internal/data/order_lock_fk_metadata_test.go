package data

import (
	"testing"

	entsqlschema "entgo.io/ent/dialect/sql/schema"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent/migrate"
)

// TestOrderLockRecordUserFKNoAction 断言生成迁移元数据中锁定历史用户外键保持
// NO ACTION：锁定事实不可变，删除用户不得清空 MANUAL 锁定记录的 locked_by 或
// 自动锁定触发审计 triggered_by，且必须与正式迁移链的删除策略同源。
// unlocked_by 属可逆审计引用，保持 SetNull 不在本断言范围外漂移。
func TestOrderLockRecordUserFKNoAction(t *testing.T) {
	expected := map[string]entsqlschema.ReferenceOption{
		"order_lock_records_users_order_lock_records":                entsqlschema.NoAction,
		"order_lock_records_users_auto_triggered_order_lock_records": entsqlschema.NoAction,
		"order_lock_records_users_unlocked_order_lock_records":       entsqlschema.SetNull,
	}
	found := map[string]entsqlschema.ReferenceOption{}
	for _, fk := range migrate.OrderLockRecordsTable.ForeignKeys {
		if _, ok := expected[fk.Symbol]; ok {
			found[fk.Symbol] = fk.OnDelete
		}
	}
	for symbol, want := range expected {
		got, ok := found[symbol]
		if !ok {
			t.Fatalf("生成元数据缺少外键 %s", symbol)
		}
		if got != want {
			t.Fatalf("外键 %s 删除策略 = %v，期望 %v", symbol, got, want)
		}
	}
}
