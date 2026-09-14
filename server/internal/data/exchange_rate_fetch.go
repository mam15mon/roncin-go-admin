package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

const (
	// sinaBankForexURL 新浪财经中行专线外汇接口：返回清洗规范的 JSON，数值已按
	// 1 外币折合本币归一化（免除 /100 换算），需携带 Referer 突破防盗链。
	sinaBankForexURL = "https://vip.stock.finance.sina.com.cn/forex/api/openapi.php/ForexService.getBankForex"
	// sinaQuoteURL 新浪外汇行情接口：以 fx_s{from}{to} 直盘符号批量查询。
	sinaQuoteURL = "https://hq.sinajs.cn/list="
	// bocOfficialPriceURL 中国银行官方公开外汇牌价页（HTML 兜底源），
	// 报价基准为 100 外币，需按 /100 换算。
	bocOfficialPriceURL = "https://www.boc.cn/sourcedb/whpj/"
	// sinaReferer 新浪外汇接口防盗链要求的 Referer。
	sinaReferer = "https://finance.sina.com.cn"
	// exchangeQuoteTimeout 外部牌价抓取超时；同步为交互式操作，不可长挂。
	exchangeQuoteTimeout = 10 * time.Second
)

// exchangeRateQuoteProvider 实现外部牌价抓取：CNY 本币走新浪中行专线主源 +
// 中行官方牌价页兜底；国际直盘走新浪 fx_s 行情。抓取失败原样返回错误，
// 是否启用备选源由 biz 层编排并在预览中明示，绝不静默兜底。
type exchangeRateQuoteProvider struct {
	client *http.Client
}

func NewExchangeRateQuoteProvider() biz.ExchangeRateQuoteProvider {
	return &exchangeRateQuoteProvider{client: &http.Client{Timeout: exchangeQuoteTimeout}}
}

func (p *exchangeRateQuoteProvider) FetchCNYBankQuotes(ctx context.Context, currencies []string) (*biz.ExchangeRateQuoteSet, error) {
	wanted := currencySet(currencies)
	payload, err := p.fetch(ctx, sinaBankForexURL, sinaReferer)
	if err == nil {
		quotes, parseErr := parseSinaBankForexQuotes(payload, wanted)
		if parseErr == nil && len(quotes) > 0 {
			return &biz.ExchangeRateQuoteSet{Source: "新浪财经中行专线", Quotes: quotes}, nil
		}
	}
	// 主源失败/无命中时启用官方牌价页兜底（biz 层会在预览中标注备选来源）。
	htmlPayload, htmlErr := p.fetch(ctx, bocOfficialPriceURL, "")
	if htmlErr != nil {
		return nil, fmt.Errorf("%w: 主源与官方兜底源均抓取失败", biz.ErrExchangeRateQuoteUnavailable)
	}
	quotes, parseErr := parseBOCOfficialPriceTable(htmlPayload, wanted)
	if parseErr != nil || len(quotes) == 0 {
		return nil, fmt.Errorf("%w: 中行官方牌价解析失败", biz.ErrExchangeRateQuoteUnavailable)
	}
	return &biz.ExchangeRateQuoteSet{Source: "中国银行官方牌价", FallbackUsed: true, Quotes: quotes}, nil
}

func (p *exchangeRateQuoteProvider) FetchDirectQuotes(ctx context.Context, baseCurrency string, currencies []string) (*biz.ExchangeRateQuoteSet, error) {
	symbols := make([]string, 0, len(currencies))
	for _, code := range currencies {
		symbols = append(symbols, "fx_s"+strings.ToLower(code+baseCurrency))
	}
	payload, err := p.fetch(ctx, sinaQuoteURL+url.QueryEscape(strings.Join(symbols, ",")), sinaReferer)
	if err != nil {
		return nil, fmt.Errorf("%w: 国际直盘行情抓取失败", biz.ErrExchangeRateQuoteUnavailable)
	}
	quotes, parseErr := parseSinaDirectQuotes(payload, baseCurrency, currencySet(currencies))
	if parseErr != nil || len(quotes) == 0 {
		return nil, fmt.Errorf("%w: 国际直盘行情解析失败", biz.ErrExchangeRateQuoteUnavailable)
	}
	return &biz.ExchangeRateQuoteSet{Source: "国际直盘行情", Quotes: quotes}, nil
}

func (p *exchangeRateQuoteProvider) fetch(ctx context.Context, target, referer string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; RoncinAdmin/1.0)")
	if referer != "" {
		request.Header.Set("Referer", referer)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("牌价源 HTTP %d", response.StatusCode)
	}
	return io.ReadAll(io.LimitReader(response.Body, 4<<20))
}

func currencySet(codes []string) map[string]struct{} {
	set := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		set[strings.ToUpper(strings.TrimSpace(code))] = struct{}{}
	}
	return set
}

// sinaBankForexResponse 新浪中行专线 JSON 结构（宽松解析：仅锚定所需字段）。
type sinaBankForexResponse struct {
	Result struct {
		Data []map[string]any `json:"data"`
	} `json:"result"`
}

// parseSinaBankForexQuotes 解析新浪中行专线 JSON：筛选 bank=boc（中国银行）条目，
// xh_sell_price（现汇卖出价）→ 建议 ar_rate，xh_buy_price（现汇买入价）→ 建议
// ap_rate；数值已归一化，直接使用。币种取自 symbol/code 字段前三位（如 USDCNY→USD）。
func parseSinaBankForexQuotes(payload []byte, wanted map[string]struct{}) (map[string]biz.ExchangeRateQuote, error) {
	var parsed sinaBankForexResponse
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return nil, err
	}
	quotes := make(map[string]biz.ExchangeRateQuote)
	for _, item := range parsed.Result.Data {
		if !isBOCBankEntry(item) {
			continue
		}
		code := bankForexCurrencyCode(item)
		if _, ok := wanted[code]; !ok {
			continue
		}
		sell, sellOK := bankForexDecimal(item, "xh_sell_price")
		buy, buyOK := bankForexDecimal(item, "xh_buy_price")
		if !sellOK || !buyOK || !sell.IsPositive() || !buy.IsPositive() {
			continue
		}
		quotes[code] = biz.ExchangeRateQuote{
			ARRate: sell,
			APRate: buy,
			Rate:   sell.Add(buy).Div(decimal.NewFromInt(2)).RoundBank(8),
			Detail: "中国银行现汇买卖价（新浪中行专线，已归一化）",
		}
	}
	return quotes, nil
}

// isBOCBankEntry 精确识别中国银行条目：bank/banklog 字段等值 boc（忽略大小写）
// 或中文名「中国银行」；子串匹配会误中 bocom（交通银行）等同前缀代码。
func isBOCBankEntry(item map[string]any) bool {
	for _, key := range []string{"bank", "banklog", "bank_code"} {
		value, ok := item[key].(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if strings.EqualFold(value, "boc") || value == "中国银行" {
			return true
		}
	}
	return false
}

func bankForexCurrencyCode(item map[string]any) string {
	for _, key := range []string{"symbol", "code", "currency", "pair"} {
		if value, ok := item[key].(string); ok && len(value) >= 3 {
			return strings.ToUpper(strings.TrimSpace(value[:3]))
		}
	}
	return ""
}

func bankForexDecimal(item map[string]any, key string) (decimal.Decimal, bool) {
	switch value := item[key].(type) {
	case string:
		parsed, err := decimal.NewFromString(strings.TrimSpace(value))
		return parsed, err == nil
	case float64:
		return decimal.NewFromFloat(value), true
	case json.Number:
		parsed, err := decimal.NewFromString(value.String())
		return parsed, err == nil
	default:
		return decimal.Decimal{}, false
	}
}

var (
	bocTableRowPattern    = regexp.MustCompile(`(?is)<tr[^>]*>(.*?)</tr>`)
	bocTableCellPattern   = regexp.MustCompile(`(?is)<td[^>]*>(.*?)</td>`)
	bocTagPattern         = regexp.MustCompile(`(?s)<[^>]+>`)
	bocPriceTablePattern  = regexp.MustCompile(`(?is)<table[^>]*id="priceTable"[^>]*>(.*?)</table>`)
	bocWhitespacePattern  = regexp.MustCompile(`\s+`)
	bocCurrencyNameToCode = map[string]string{
		"美元": "USD", "港币": "HKD", "欧元": "EUR", "日元": "JPY", "英镑": "GBP",
		"瑞士法郎": "CHF", "澳大利亚元": "AUD", "加拿大元": "CAD", "新加坡元": "SGD", "新西兰元": "NZD",
		"韩元": "KRW", "泰铢": "THB", "新台币": "TWD", "澳门元": "MOP", "林吉特": "MYR",
		"卢布": "RUB", "印度卢比": "INR", "菲律宾比索": "PHP", "印尼卢比": "IDR", "南非兰特": "ZAR",
		"瑞典克朗": "SEK", "挪威克朗": "NOK", "丹麦克朗": "DKK", "土耳其里拉": "TRY",
		"阿联酋迪拉姆": "AED", "沙特里亚尔": "SAR", "巴西里亚尔": "BRL", "巴西雷亚尔": "BRL",
		"匈牙利福林": "HUF", "波兰兹罗提": "PLN", "以色列谢克尔": "ILS", "埃及镑": "EGP",
	}
)

// parseBOCOfficialPriceTable 解析中行官方牌价页 `<table id="priceTable">`：
// 列序为 货币名称/现汇买入价/现钞买入价/现汇卖出价/现钞卖出价/中行折算价/发布时间，
// 报价基准为 100 外币，统一按 /100 换算归一化。
func parseBOCOfficialPriceTable(payload []byte, wanted map[string]struct{}) (map[string]biz.ExchangeRateQuote, error) {
	table := bocPriceTablePattern.FindSubmatch(payload)
	if table == nil {
		return nil, fmt.Errorf("官方牌价页缺少 priceTable")
	}
	quotes := make(map[string]biz.ExchangeRateQuote)
	for _, rowMatch := range bocTableRowPattern.FindAllSubmatch(table[1], -1) {
		cells := bocTableCellPattern.FindAllSubmatch(rowMatch[1], -1)
		if len(cells) < 6 {
			continue
		}
		texts := make([]string, len(cells))
		for index, cell := range cells {
			text := bocTagPattern.ReplaceAll(cell[1], nil)
			texts[index] = bocWhitespacePattern.ReplaceAllString(string(text), "")
		}
		code, ok := bocCurrencyNameToCode[texts[0]]
		if !ok {
			if _, wantedOK := wanted[strings.ToUpper(texts[0])]; !wantedOK {
				continue
			}
			code = strings.ToUpper(texts[0])
		}
		if _, wantedOK := wanted[code]; !wantedOK {
			continue
		}
		buy, buyOK := bocNormalizedPrice(texts[1])
		sell, sellOK := bocNormalizedPrice(texts[3])
		conversion, conversionOK := bocNormalizedPrice(texts[5])
		if !buyOK || !sellOK || !conversionOK {
			continue
		}
		quotes[code] = biz.ExchangeRateQuote{
			ARRate: sell,
			APRate: buy,
			Rate:   conversion,
			Detail: "中国银行官方牌价（现汇卖出/买入价与中行折算价，/100 归一化）",
		}
	}
	return quotes, nil
}

func bocNormalizedPrice(text string) (decimal.Decimal, bool) {
	if text == "" {
		return decimal.Decimal{}, false
	}
	value, err := decimal.NewFromString(text)
	if err != nil || !value.IsPositive() {
		return decimal.Decimal{}, false
	}
	return value.Div(decimal.NewFromInt(100)).RoundBank(8), true
}

var sinaQuoteLinePattern = regexp.MustCompile(`hq_str_fx_s([a-z]{6})="([^"]*)"`)

// parseSinaDirectQuotes 解析新浪外汇直盘行情（fx_s{from}{to}）：字段序为
// 名称,买入价(Bid),卖出价(Ask),...；Bid→建议 ap_rate，Ask→建议 ar_rate，
// 基准价取中间价口径。报价已归一化（1 外币折目标币）。
func parseSinaDirectQuotes(payload []byte, baseCurrency string, wanted map[string]struct{}) (map[string]biz.ExchangeRateQuote, error) {
	quotes := make(map[string]biz.ExchangeRateQuote)
	for _, match := range sinaQuoteLinePattern.FindAllSubmatch(payload, -1) {
		pair := string(match[1])
		to := strings.ToUpper(pair[3:])
		from := strings.ToUpper(pair[:3])
		if to != strings.ToUpper(baseCurrency) {
			continue
		}
		if _, ok := wanted[from]; !ok {
			continue
		}
		fields := strings.Split(string(match[2]), ",")
		if len(fields) < 3 {
			continue
		}
		bid, bidErr := decimal.NewFromString(strings.TrimSpace(fields[1]))
		ask, askErr := decimal.NewFromString(strings.TrimSpace(fields[2]))
		if bidErr != nil || askErr != nil || !bid.IsPositive() || !ask.IsPositive() {
			continue
		}
		quotes[from] = biz.ExchangeRateQuote{
			ARRate: ask,
			APRate: bid,
			Rate:   ask.Add(bid).Div(decimal.NewFromInt(2)).RoundBank(8),
			Detail: fmt.Sprintf("国际直盘 %s→%s（Ask→应收，Bid→应付）", from, to),
		}
	}
	return quotes, nil
}
