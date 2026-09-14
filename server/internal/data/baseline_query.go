package data

import (
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/exchangeratesetting"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// B 型（基线+本地）统一查询谓词工厂。
//
// B 型表 organization_id 可空：NULL=集团基线行（全网可见），非空=本组织行。
// 同码数据本组织行无条件覆盖基线行（Shadowing）。两种形态（design §2）：
//
//  1. 点查形态 BaselineOrLocal：`organization_id = ? OR organization_id IS NULL`，
//     配合 BaselineLocalFirstOrder 让本组织行排序在前，取首行；
//  2. 列表形态 BaselineShadowedByLocal：被同码本组织行覆盖的基线行整行不可见，
//     NOT EXISTS 反连接下推到 SQL，保证分页计数、排序与去重结果一致；
//     禁止在分页列表中使用点查的 ORDER BY + LIMIT 1 形态。
//
// 表名与列名全部来自下方各实体的固定常量清单，不接收外部输入；大小写不敏感比对
// 沿承既有 writeIndustryLocalCodePrecedence 的 UPPER 语义。

// baselineOrLocalWhere 点查谓词：本组织行或基线行都参与候选。
func baselineOrLocalWhere(organizationColumn string, organizationID uuid.UUID) func(*sql.Selector) {
	return func(selector *sql.Selector) {
		selector.Where(sql.Or(
			sql.EQ(selector.C(organizationColumn), organizationID),
			sql.IsNull(selector.C(organizationColumn)),
		))
	}
}

// baselineLocalFirstOrder 点查排序：本组织行优先于基线行（取首行即本组织覆盖结果）。
func baselineLocalFirstOrder(organizationColumn string, organizationID uuid.UUID) func(*sql.Selector) {
	return func(selector *sql.Selector) {
		selector.OrderExprFunc(func(builder *sql.Builder) {
			builder.WriteString("CASE WHEN ").Ident(selector.C(organizationColumn)).WriteString(" = ").Arg(organizationID).WriteString(" THEN 0 ELSE 1 END")
		})
	}
}

// baselineShadowedByLocalWhere 列表谓词：本组织行 + 未被同码本组织行覆盖的基线行。
func baselineShadowedByLocalWhere(table, organizationColumn, codeColumn string, organizationID uuid.UUID) func(*sql.Selector) {
	return func(selector *sql.Selector) {
		selector.Where(sql.Or(
			sql.EQ(selector.C(organizationColumn), organizationID),
			sql.And(
				sql.IsNull(selector.C(organizationColumn)),
				sql.P(func(builder *sql.Builder) {
					builder.WriteString("NOT EXISTS (SELECT 1 FROM ")
					builder.WriteString(table)
					builder.WriteString(" AS local_shadow WHERE local_shadow.organization_id = ")
					builder.Arg(organizationID)
					builder.WriteString(" AND UPPER(local_shadow.")
					builder.WriteString(codeColumn)
					builder.WriteString(") = UPPER(")
					builder.Ident(selector.C(codeColumn))
					builder.WriteString("))")
				}),
			),
		))
	}
}

// portBaselineScope 港口列表形态谓词（业务码 un_locode）。
func portBaselineScope(organizationID uuid.UUID) predicate.Port {
	return baselineShadowedByLocalWhere("ports", port.FieldOrganizationID, port.FieldUnLocode, organizationID)
}

// airportBaselineScope 机场列表形态谓词（业务码 iata_code）。
func airportBaselineScope(organizationID uuid.UUID) predicate.Airport {
	return baselineShadowedByLocalWhere("airports", airport.FieldOrganizationID, airport.FieldIataCode, organizationID)
}

// feeSettingBaselineScope 费用设置列表形态谓词（业务码 fee_code）。
func feeSettingBaselineScope(organizationID uuid.UUID) predicate.FeeSetting {
	return baselineShadowedByLocalWhere("fee_settings", feesetting.FieldOrganizationID, feesetting.FieldFeeCode, organizationID)
}

// exchangeRateBaselineScope 汇率点查谓词（业务码 from+to+生效期，唯一性由
// 部分唯一索引保证，这里按「本组织行或基线行」取候选，排序本组织优先）。
func exchangeRateBaselineScope(organizationID uuid.UUID) predicate.ExchangeRateSetting {
	return baselineOrLocalWhere(exchangeratesetting.FieldOrganizationID, organizationID)
}

// exchangeRateLocalFirstOrder 汇率点查排序：本组织行优先于基线行。
func exchangeRateLocalFirstOrder(organizationID uuid.UUID) func(*sql.Selector) {
	return baselineLocalFirstOrder(exchangeratesetting.FieldOrganizationID, organizationID)
}
