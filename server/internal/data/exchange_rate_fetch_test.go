package data

import (
	"testing"

	"github.com/shopspring/decimal"
)

// 新浪中行专线 JSON fixture：bank=boc 条目携带 xh_sell_price / xh_buy_price，
// 数值已按 1 外币折 CNY 归一化（无需 /100）。
const sinaBankForexFixture = `{"result":{"status":{"code":0},"data":[
  {"bank":"boc","symbol":"USDCNY","xh_sell_price":6.7523,"xh_buy_price":6.7117},
  {"bank":"boc","symbol":"HKDCNY","xh_sell_price":0.8721,"xh_buy_price":0.8655},
  {"bank":"boc","symbol":"EURCNY","xh_sell_price":"7.9120","xh_buy_price":"7.8460"},
  {"bank":"abc","symbol":"USDCNY","xh_sell_price":9.99,"xh_buy_price":9.99},
  {"bank":"bocom","symbol":"USDCNY","xh_sell_price":8.88,"xh_buy_price":8.88},
  {"bank":"boc","symbol":"XADCNY","xh_sell_price":1,"xh_buy_price":1}
]}}`

const sinaBankForexBadFixture = `not-json`

func TestParseSinaBankForexQuotes(t *testing.T) {
	quotes, err := parseSinaBankForexQuotes([]byte(sinaBankForexFixture), currencySet([]string{"USD", "HKD", "EUR"}))
	if err != nil {
		t.Fatalf("解析新浪中行专线失败: %v", err)
	}
	if len(quotes) != 3 {
		t.Fatalf("应命中 3 个币种（非中行条目与未知币种被过滤），实际 %d", len(quotes))
	}
	usd := quotes["USD"]
	if usd.ARRate.StringFixed(4) != "6.7523" || usd.APRate.StringFixed(4) != "6.7117" {
		t.Fatalf("USD 卖出/买入价不符: %#v", usd)
	}
	// 基准价缺省为中间价口径。
	if !usd.Rate.Equal(decimal.RequireFromString("6.7523").Add(decimal.RequireFromString("6.7117")).Div(decimal.NewFromInt(2)).RoundBank(8)) {
		t.Fatalf("USD 基准价应为中间价: %s", usd.Rate)
	}
	// 字符串数值同样支持。
	if quotes["EUR"].ARRate.StringFixed(4) != "7.9120" {
		t.Fatalf("EUR 卖出价不符: %#v", quotes["EUR"])
	}
}

func TestParseSinaBankForexQuotesRejectsGarbage(t *testing.T) {
	if _, err := parseSinaBankForexQuotes([]byte(sinaBankForexBadFixture), currencySet([]string{"USD"})); err == nil {
		t.Fatal("非 JSON 载荷应报解析错误")
	}
}

// 中行官方牌价页 HTML fixture：`<table id="priceTable">`，列序为 货币名称/现汇买入价/
// 现钞买入价/现汇卖出价/现钞卖出价/中行折算价/发布时间，报价基准为 100 外币。
const bocOfficialPriceFixture = `<html><body>
<table id="priceTable"><tbody>
<tr><td>货币名称</td><td>现汇买入价</td><td>现钞买入价</td><td>现汇卖出价</td><td>现钞卖出价</td><td>中行折算价</td><td>发布时间</td></tr>
<tr><td>美元</td><td>671.17</td><td>671.17</td><td>675.23</td><td>675.23</td><td>672.30</td><td>2026.09.14 10:00:01</td></tr>
<tr><td>港币</td><td>86.55</td><td>86.55</td><td>87.21</td><td>87.21</td><td>86.80</td><td>2026.09.14 10:00:01</td></tr>
<tr><td>未知币种</td><td>1</td><td>1</td><td>1</td><td>1</td><td>1</td><td>2026.09.14</td></tr>
</tbody></table>
</body></html>`

func TestParseBOCOfficialPriceTableNormalizesPer100(t *testing.T) {
	quotes, err := parseBOCOfficialPriceTable([]byte(bocOfficialPriceFixture), currencySet([]string{"USD", "HKD"}))
	if err != nil {
		t.Fatalf("解析中行官方牌价页失败: %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("应命中 2 个币种，实际 %d", len(quotes))
	}
	usd := quotes["USD"]
	// 官方页报价基准为 100 外币，须按 /100 归一化。
	if usd.ARRate.StringFixed(4) != "6.7523" || usd.APRate.StringFixed(4) != "6.7117" {
		t.Fatalf("USD /100 归一化不符: %#v", usd)
	}
	// 中行折算价直接作为基准价。
	if usd.Rate.StringFixed(4) != "6.7230" {
		t.Fatalf("USD 基准价应取中行折算价: %s", usd.Rate)
	}
	if quotes["HKD"].APRate.StringFixed(4) != "0.8655" {
		t.Fatalf("HKD 现汇买入价不符: %#v", quotes["HKD"])
	}
}

func TestParseBOCOfficialPriceTableRejectsMissingTable(t *testing.T) {
	if _, err := parseBOCOfficialPriceTable([]byte("<html><body>no table</body></html>"), currencySet([]string{"USD"})); err == nil {
		t.Fatal("缺少 priceTable 应报解析错误")
	}
}

// 新浪国际直盘 fixture：fx_s{from}{to}，字段序为 名称,买入价(Bid),卖出价(Ask),...，
// 报价已归一化（1 外币折目标币）。
const sinaDirectQuotesFixture = `var hq_str_fx_susdhkd="美元港币,7.8222,7.8226,7.8180,7.8250";` + "\n" +
	`var hq_str_fx_seurhkd="欧元港币,9.5010,9.5060,9.4950,9.5100";` + "\n" +
	`var hq_str_fx_susdcny="美元人民币,6.7100,6.7500,6.7000,6.7600";` + "\n" +
	`var hq_str_fx_seurusd="欧元美元,1.0800,1.0810,1.0790,1.0820";`

func TestParseSinaDirectQuotesUsesBidAsk(t *testing.T) {
	quotes, err := parseSinaDirectQuotes([]byte(sinaDirectQuotesFixture), "HKD", currencySet([]string{"USD", "EUR", "JPY"}))
	if err != nil {
		t.Fatalf("解析国际直盘失败: %v", err)
	}
	if len(quotes) != 2 {
		t.Fatalf("应命中 2 个直盘币种，实际 %d", len(quotes))
	}
	usd := quotes["USD"]
	// Bid→ap（应付），Ask→ar（应收）。
	if usd.ARRate.StringFixed(4) != "7.8226" || usd.APRate.StringFixed(4) != "7.8222" {
		t.Fatalf("USD 直盘 Ask/Bid 映射不符: %#v", usd)
	}
	if usd.Detail == "" || usd.Detail != "国际直盘 USD→HKD（Ask→应收，Bid→应付）" {
		t.Fatalf("直盘换算路径应明示: %s", usd.Detail)
	}
	// 缺失币种（JPY）不出现在结果中。
	if _, ok := quotes["JPY"]; ok {
		t.Fatal("未请求的币种不应出现")
	}
}

func TestParseSinaDirectQuotesIgnoresWrongBase(t *testing.T) {
	quotes, err := parseSinaDirectQuotes([]byte(sinaDirectQuotesFixture), "SGD", currencySet([]string{"USD", "EUR"}))
	if err != nil {
		t.Fatalf("解析国际直盘失败: %v", err)
	}
	if len(quotes) != 0 {
		t.Fatalf("目标本币不符的行情应被忽略，实际 %#v", quotes)
	}
}
