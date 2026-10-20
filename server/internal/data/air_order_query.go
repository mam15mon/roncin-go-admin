package data

import (
	"strings"

	entsql "entgo.io/ent/dialect/sql"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// airOrderNumberContainsFold 从真实关联搜索号码，不以客户业务号猜测主单成员。
func airOrderNumberContainsFold(keyword string, house, consolidated bool) predicate.Order {
	keyword = strings.TrimSpace(keyword)
	if !house {
		keyword = strings.ReplaceAll(keyword, "-", "")
	}
	return predicate.Order(func(selector *entsql.Selector) {
		selector.Where(entsql.P(func(b *entsql.Builder) {
			b.WriteString(`EXISTS (SELECT 1 FROM "air_master_waybill_order_links" AS "air_filter_link" JOIN "air_master_waybills" AS "air_filter_master" ON "air_filter_master"."id" = "air_filter_link"."master_waybill_id" `)
			if house {
				b.WriteString(`JOIN "air_house_waybills" AS "air_filter_house" ON "air_filter_house"."order_id" = "air_filter_link"."order_id" `)
			}
			b.WriteString(`WHERE "air_filter_link"."order_id" = `).Ident(selector.C(orderent.FieldID)).WriteString(` AND "air_filter_link"."status" = 'ACTIVE' AND LOWER(`)
			if house {
				b.WriteString(`"air_filter_house"."house_no"`)
			} else {
				b.WriteString(`"air_filter_master"."master_no"`)
			}
			b.WriteString(`) LIKE `).Arg("%" + strings.ToLower(keyword) + "%")
			if consolidated {
				b.WriteString(` AND (SELECT COUNT(*) FROM "air_master_waybill_order_links" AS "air_sub_link" WHERE "air_sub_link"."master_waybill_id" = "air_filter_master"."id" AND "air_sub_link"."status" = 'ACTIVE') > 1`)
			}
			b.WriteString(`)`)
		}))
	})
}

// writeOrderTransportTimeExpression 使用 AE 权威主单时刻，SE 保持已有订单投影筛选。
func writeOrderTransportTimeExpression(b *entsql.Builder, s *entsql.Selector, field string) *entsql.Builder {
	b.WriteString(`NULLIF(CASE WHEN `).Ident(s.C(orderent.FieldBusinessType)).WriteString(` = 'AE' THEN (SELECT "air_date_master".`).Ident(field).WriteString(` FROM "air_master_waybill_order_links" AS "air_date_link" JOIN "air_master_waybills" AS "air_date_master" ON "air_date_master"."id" = "air_date_link"."master_waybill_id" WHERE "air_date_link"."order_id" = `).Ident(s.C(orderent.FieldID)).WriteString(` ORDER BY "air_date_link"."started_at" DESC LIMIT 1) ELSE `).Ident(s.C(field)).WriteString(` END, '')::timestamptz`)
	return b
}
