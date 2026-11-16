package data

import entsql "entgo.io/ent/dialect/sql"

// industryReferenceCodeRelevance 在分页前优先排列精确代码、代码前缀，再沿用目录排序。
// 名称与拼音仍由各目录原有谓词筛选；Ent 谓词负责绑定参数并转义 LIKE 通配符。
func industryReferenceCodeRelevance(keyword string, fields ...string) func(*entsql.Selector) {
	return func(selector *entsql.Selector) {
		if keyword == "" {
			return
		}
		exact := make([]*entsql.Predicate, 0, len(fields))
		prefix := make([]*entsql.Predicate, 0, len(fields))
		for _, field := range fields {
			exact = append(exact, entsql.EqualFold(selector.C(field), keyword))
			prefix = append(prefix, entsql.HasPrefixFold(selector.C(field), keyword))
		}
		selector.OrderExpr(entsql.ExprFunc(func(builder *entsql.Builder) {
			builder.WriteString("CASE WHEN ").Join(entsql.Or(exact...)).
				WriteString(" THEN 0 WHEN ").Join(entsql.Or(prefix...)).
				WriteString(" THEN 1 ELSE 2 END")
		}))
	}
}
