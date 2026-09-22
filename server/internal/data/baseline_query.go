package data

import (
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// B 型（基线+本地）统一查询谓词工厂。
//
// B 型表 organization_id 可空：NULL=集团基线行（全网可见），非空=本组织行。
// 同码数据本组织行无条件覆盖基线行（Shadowing）：被同码本组织行覆盖的基线行
// 整行不可见，NOT EXISTS 反连接下推到 SQL，保证分页计数、排序与去重结果一致。
//
// 表名与列名全部来自下方各实体的固定常量清单，不接收外部输入；大小写不敏感比对
// 沿承既有 writeIndustryLocalCodePrecedence 的 UPPER 语义。

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

// portReferenceScope 按已保存 UUID 解析引用，保留被本地同码行遮蔽的共享基线。
// 候选列表与新写入仍使用 portBaselineScope，其他公司的私有港口不可见。
func portReferenceScope(organizationID uuid.UUID) predicate.Port {
	return port.Or(port.OrganizationIDIsNil(), port.OrganizationIDEQ(organizationID))
}
