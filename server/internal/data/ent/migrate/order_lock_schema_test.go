package migrate

import (
	"testing"

	"entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/schema/field"
)

func TestOrderLockSchemaMetadataSupportsAllBusinessTypes(t *testing.T) {
	lockBusinessType := requireColumn(t, OrderLockRecordsColumns, "business_type")
	if lockBusinessType.Type != field.TypeEnum || lockBusinessType.Nullable {
		t.Fatalf("锁记录 business_type 元数据异常: %#v", lockBusinessType)
	}
	wantEnums := []string{"SE", "SI", "AE", "AI", "LAND", "RAIL"}
	if !equalStrings(lockBusinessType.Enums, wantEnums) {
		t.Fatalf("锁记录 business_type 枚举 = %v，期望 %v", lockBusinessType.Enums, wantEnums)
	}
	for _, name := range []string{"master_bill_id", "master_bill_version_id"} {
		if column := requireColumn(t, OrderLockRecordsColumns, name); !column.Nullable {
			t.Fatalf("锁记录 %s 必须可空", name)
		}
	}

	requestBusinessType := requireColumn(t, OrderUnlockRequestsColumns, "business_type")
	if requestBusinessType.Type != field.TypeEnum || requestBusinessType.Nullable || !equalStrings(requestBusinessType.Enums, wantEnums) {
		t.Fatalf("解锁请求 business_type 元数据异常: %#v", requestBusinessType)
	}

	const checkName = "order_lock_records_business_type_document_refs_check"
	const checkExpression = "(business_type = 'SE' AND master_bill_id IS NOT NULL AND master_bill_version_id IS NOT NULL) OR (business_type IN ('SI', 'AE', 'AI', 'LAND', 'RAIL') AND master_bill_id IS NULL AND master_bill_version_id IS NULL)"
	if OrderLockRecordsTable.Annotation == nil || OrderLockRecordsTable.Annotation.Checks[checkName] != checkExpression {
		t.Fatalf("锁记录 CHECK 元数据缺失或不一致: %#v", OrderLockRecordsTable.Annotation)
	}

	for _, symbol := range []string{
		"order_lock_records_sea_master_bills_lock_records",
		"order_lock_records_sea_master_bill_versions_lock_records",
	} {
		foreignKey := requireForeignKey(t, OrderLockRecordsTable.ForeignKeys, symbol)
		if foreignKey.OnDelete != schema.NoAction {
			t.Fatalf("锁记录历史快照外键 %s 的删除策略 = %v，期望 NO ACTION", symbol, foreignKey.OnDelete)
		}
	}
}

func requireColumn(t *testing.T, columns []*schema.Column, name string) *schema.Column {
	t.Helper()
	for _, column := range columns {
		if column.Name == name {
			return column
		}
	}
	t.Fatalf("生成的迁移元数据缺少列 %s", name)
	return nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func requireForeignKey(t *testing.T, foreignKeys []*schema.ForeignKey, symbol string) *schema.ForeignKey {
	t.Helper()
	for _, foreignKey := range foreignKeys {
		if foreignKey.Symbol == symbol {
			return foreignKey
		}
	}
	t.Fatalf("生成的迁移元数据缺少外键 %s", symbol)
	return nil
}
