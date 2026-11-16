package data

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/platform/searchtext"
)

type searchKeywordBackfillSpec struct {
	table        string
	columns      []string
	targetColumn string
}

var selectorSearchKeywordBackfills = []searchKeywordBackfillSpec{
	{table: "organizations", columns: []string{"code", "name", "name_en"}},
	{table: "currencies", columns: []string{"name"}},
	{table: "partners", columns: []string{"legal_name"}},
	{table: "partner_alias", columns: []string{"alias_name"}},
	{table: "master_data_items", columns: []string{"name", "name_en"}},
	{table: "ports", columns: []string{"name_zh", "name_en"}},
	{table: "airports", columns: []string{"name_zh", "name_en", "city_name_zh", "city_name_en"}},
	{table: "airlines", columns: []string{"name_zh", "name_en"}},
	{table: "shipping_lines", columns: []string{"name_zh", "name_en"}},
	{table: "administrative_regions", columns: []string{"name"}},
	{table: "users", columns: []string{"display_name", "dingtalk_name"}},
	{table: "fee_settings", columns: []string{"name_zh", "name_en", "alias_name"}},
	{table: "fee_setting_templates", columns: []string{"name_zh", "name_en", "alias_name"}},
	{table: "order_fees", columns: []string{"fee_name", "fee_name_en"}},
	{table: "billing_units", columns: []string{"name"}},
	{table: "taxable_services", columns: []string{"name", "short_name"}},
	{table: "finance_bills", columns: []string{"settlement_party_name"}},
	{table: "finance_cashflows", columns: []string{"settlement_party_name"}},
	{table: "finance_invoices", columns: []string{"settlement_party_name"}},
	{table: "finance_nettings", columns: []string{"settlement_party_name"}},
	{table: "finance_verifications", columns: []string{"settlement_party_name"}},
	{table: "finance_commissions", columns: []string{"employee_name", "rule_name"}},
	{table: "finance_commission_rules", columns: []string{"name"}},
	{table: "partner_contacts", columns: []string{"name"}},
	{table: "enterprise_resource_addresses", columns: []string{"address_detail", "contact_name"}},
	{table: "enterprise_resource_parties", columns: []string{"company_name", "address", "contact_name"}},
	{table: "enterprise_resource_remarks", columns: []string{"content"}},
	{table: "enterprise_resources", columns: []string{"short_name"}},
	{table: "orders", columns: []string{"vessel_voyage", "goods_description"}},
	{table: "orders", columns: []string{"shipper_short_name"}, targetColumn: "shipper_search_keywords"},
	{table: "orders", columns: []string{"consignee_short_name"}, targetColumn: "consignee_search_keywords"},
}

// BackfillSelectorSearchKeywords 为迁移前已有的候选项数据补齐拼音检索键。
// 表名与列名来自代码内固定清单，不接收外部输入。
func BackfillSelectorSearchKeywords(ctx context.Context, db *sql.DB) error {
	for _, spec := range selectorSearchKeywordBackfills {
		if err := backfillSelectorSearchKeywords(ctx, db, spec); err != nil {
			return err
		}
	}
	return nil
}

func backfillSelectorSearchKeywords(ctx context.Context, db *sql.DB, spec searchKeywordBackfillSpec) error {
	targetColumn := spec.targetColumn
	if targetColumn == "" {
		targetColumn = "search_keywords"
	}
	quotedColumns := make([]string, 0, len(spec.columns))
	for _, column := range spec.columns {
		quotedColumns = append(quotedColumns, fmt.Sprintf("COALESCE(\"%s\", '')", column))
	}
	query := fmt.Sprintf("SELECT \"id\", %s FROM \"%s\" WHERE \"%s\" = ''", strings.Join(quotedColumns, ", "), spec.table, targetColumn)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("读取 %s 拼音检索回填数据: %w", spec.table, err)
	}
	type pendingUpdate struct {
		id       string
		keywords string
	}
	pending := make([]pendingUpdate, 0)
	for rows.Next() {
		values := make([]string, len(spec.columns))
		destinations := make([]any, 0, len(values)+1)
		var id string
		destinations = append(destinations, &id)
		for index := range values {
			destinations = append(destinations, &values[index])
		}
		if err := rows.Scan(destinations...); err != nil {
			_ = rows.Close()
			return fmt.Errorf("扫描 %s 拼音检索回填数据: %w", spec.table, err)
		}
		pending = append(pending, pendingUpdate{id: id, keywords: searchtext.Build(values...)})
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("关闭 %s 拼音检索回填结果: %w", spec.table, err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("遍历 %s 拼音检索回填数据: %w", spec.table, err)
	}
	const batchSize = 500
	for start := 0; start < len(pending); start += batchSize {
		end := start + batchSize
		if end > len(pending) {
			end = len(pending)
		}
		placeholders := make([]string, 0, end-start)
		arguments := make([]any, 0, (end-start)*2)
		for index, item := range pending[start:end] {
			parameter := index*2 + 1
			placeholders = append(placeholders, fmt.Sprintf("($%d::uuid, $%d::text)", parameter, parameter+1))
			arguments = append(arguments, item.id, item.keywords)
		}
		update := fmt.Sprintf(
			"UPDATE \"%s\" AS target SET \"%s\" = source.keywords FROM (VALUES %s) AS source(id, keywords) WHERE target.\"id\" = source.id AND target.\"%s\" = ''",
			spec.table, targetColumn,
			strings.Join(placeholders, ", "), targetColumn,
		)
		if _, err := db.ExecContext(ctx, update, arguments...); err != nil {
			return fmt.Errorf("批量回填 %s 拼音检索键: %w", spec.table, err)
		}
	}
	return nil
}
