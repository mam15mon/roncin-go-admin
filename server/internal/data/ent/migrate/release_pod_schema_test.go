package migrate

import (
	"testing"

	"entgo.io/ent/dialect/sql/schema"
)

func TestOrderReleasePodSchemaMetadataSupportsSeaDocuments(t *testing.T) {
	for _, name := range []string{"shipping_document_id", "sea_master_bill_id", "sea_house_bill_id"} {
		if column := requireColumn(t, OrderReleasePodsColumns, name); !column.Nullable {
			t.Fatalf("放货记录单证引用列 %s 必须可空", name)
		}
	}

	const checkName = "order_release_pods_document_reference_check"
	const checkExpression = "num_nonnulls(shipping_document_id, sea_master_bill_id, sea_house_bill_id) <= 1"
	if OrderReleasePodsTable.Annotation == nil || OrderReleasePodsTable.Annotation.Checks[checkName] != checkExpression {
		t.Fatalf("放货记录单证互斥 CHECK 元数据缺失或不一致: %#v", OrderReleasePodsTable.Annotation)
	}

	for _, symbol := range []string{
		"order_release_pods_sea_master_bills_release_pods",
		"order_release_pods_sea_house_bills_release_pods",
	} {
		foreignKey := requireForeignKey(t, OrderReleasePodsTable.ForeignKeys, symbol)
		if foreignKey.OnDelete != schema.NoAction {
			t.Fatalf("放货记录真实海运单证外键 %s 的删除策略 = %v，期望 NO ACTION", symbol, foreignKey.OnDelete)
		}
	}
}
