package data

import (
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partneralias"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerprofile"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// 财务业务编号检索同时支持往来单位代码、中英文名称、别名及已维护的拼音键。
// 公司范围由引用该单位的资金/账单主查询限定；不因单位停用丢失历史单据候选。
func financePartyKeywordPredicate(keyword string) predicate.Partner {
	return partner.Or(
		partner.CodeContainsFold(keyword), partner.LegalNameContainsFold(keyword), partner.SearchKeywordsContainsFold(keyword),
		partner.HasProfileWith(partnerprofile.NameEnContainsFold(keyword)),
		partner.HasAliasesWith(partneralias.Or(partneralias.AliasNameContainsFold(keyword), partneralias.SearchKeywordsContainsFold(keyword))),
	)
}
