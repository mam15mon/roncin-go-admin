package data

import (
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
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

func TestOrderOrganizationScopePredicatePairsBusinessTypeWithOrganizationIDs(t *testing.T) {
	tianjinID := uuid.New()
	beijingID := uuid.New()
	shanghaiID := uuid.New()
	table := entsql.Table(orderent.Table)
	selector := entsql.Dialect(dialect.Postgres).Select(table.C(orderent.FieldID)).From(table)
	orderOrganizationScopePredicate([]biz.OrderOrganizationScope{
		{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{tianjinID, beijingID}},
		{BusinessType: biz.OrderBusinessAI, OrganizationIDs: []uuid.UUID{shanghaiID}},
	})(selector)

	query, args := selector.Query()
	for _, fragment := range []string{
		`"orders"."business_type" = $1`,
		`"orders"."organization_id" IN ($2, $3)`,
		`"orders"."business_type" = $4`,
		`"orders"."organization_id" IN ($5)`,
		`) OR (`,
	} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("查询未按业务类型与组织范围成对过滤，缺少 %q: %s", fragment, query)
		}
	}
	if len(args) != 5 || args[0] != orderent.BusinessTypeSE || args[1] != tianjinID || args[2] != beijingID || args[3] != orderent.BusinessTypeAI || args[4] != shanghaiID {
		t.Fatalf("scope predicate args = %#v", args)
	}
}
