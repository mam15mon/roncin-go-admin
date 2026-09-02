package data

import (
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

func TestOrderConsolidatedMasterFilterRequiresMultipleOrders(t *testing.T) {
	table := entsql.Table(orderent.Table)
	selector := entsql.Dialect(dialect.Postgres).Select(table.C(orderent.FieldID)).From(table)
	orderConsolidatedMasterContainsFold("MBL-001")(selector)

	query, args := selector.Query()
	for _, fragment := range []string{
		`JOIN "sea_master_bills"`,
		`LOWER("filter_mbl"."master_no") LIKE`,
		`COUNT(*) FROM "sea_master_bill_order_links"`,
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("查询未包含 %q: %s", fragment, query)
		}
	}
	if len(args) != 1 || args[0] != "%mbl-001%" {
		t.Fatalf("args = %#v", args)
	}
}
