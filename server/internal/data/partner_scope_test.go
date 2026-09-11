package data

import (
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

func TestPartnerOrganizationScopePredicateUsesExplicitOrganizationIDs(t *testing.T) {
	tianjinID := uuid.New()
	beijingID := uuid.New()
	table := entsql.Table(partnerent.Table)
	selector := entsql.Dialect(dialect.Postgres).Select(table.C(partnerent.FieldID)).From(table)
	partnerOrganizationScopePredicate([]uuid.UUID{tianjinID, beijingID})(selector)
	query, args := selector.Query()
	if !strings.Contains(query, `"partners"."organization_id" IN ($1, $2)`) {
		t.Fatalf("合作伙伴组织谓词应由数据库显式过滤，query=%s", query)
	}
	if len(args) != 2 || args[0] != tianjinID || args[1] != beijingID {
		t.Fatalf("合作伙伴组织谓词参数=%v，期望天津和北京组织", args)
	}
}
