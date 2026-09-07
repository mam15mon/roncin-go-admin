package main

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	scacPattern      = regexp.MustCompile(`^[A-Z]{2,4}$`)
	countryPattern   = regexp.MustCompile(`^[A-Z]{2}$`)
	prefixPattern    = regexp.MustCompile(`^[A-Z]{4}$`)
	httpProtocolExpr = regexp.MustCompile(`^https?://`)
)

// topLinersZH 映射中国国际海运货代常用主流船公司的官方/行业通用中文名称。
var topLinersZH = map[string]string{
	"COS": "中远海运集运",
	"MSK": "马士基航运",
	"MSC": "地中海航运",
	"CMA": "达飞轮船",
	"HLC": "赫伯罗特",
	"ONE": "海洋网联船务",
	"EMC": "长荣海运",
	"HMM": "韩新海运",
	"YML": "阳明海运",
	"ZIM": "以星综合航运",
	"WHL": "万海航运",
	"OOL": "东方海外货柜航运",
	"SIT": "海丰集运",
	"PIL": "太平洋船务",
	"KMT": "高丽海运",
	"TSL": "德翔海运",
	"SNK": "长锦商船",
	"CUL": "中联航运",
	"RCL": "宏海箱运",
	"ESL": "阿联酋航运",
	"MAT": "美森轮船",
	"ASL": "亚海航运",
	"CNC": "正利航业",
	"ANL": "澳国航运",
	"APL": "美国总统轮船",
	"HAS": "中谷海运",
	"ANT": "安通控股",
	"SML": "森罗商船",
	"IAL": "运达航运",
	"DJS": "锦江航运",
	"HED": "海德航运",
	"PAN": "泛洋海运",
	"SIN": "中国外运集运",
	"SJJ": "上港锦江",
	"MAR": "玛鲁巴航运",
	"ACL": "大西洋集装箱航运",
	"UAS": "阿拉伯联合国家轮船",
	"MCC": "马士基海陆(亚洲)",
	"SGL": "马士基海陆(欧地)",
	"SLD": "马士基海陆(美洲)",
	"SJL": "萨哈克集装箱航运",
	"EGH": "长荣海运(香港)",
	"EMA": "长荣海运(亚洲)",
	"EMS": "长荣海运(新加坡)",
	"HTM": "初野海运",
	"COE": "中远海运(欧洲)",
	"CSE": "鑫金海航运",
	"CSH": "达飞短海航运",
	"CLN": "正利航业",
	"CMN": "摩洛哥航运",
	"CAV": "智利南美邮船",
	"TRK": "土耳其图尔肯航运",
	"UFE": "联羽航运",
}

// topAlliances 映射国际集装箱班轮主要航运联盟。
var topAlliances = map[string]string{
	"COS": "Ocean Alliance",
	"CMA": "Ocean Alliance",
	"EMC": "Ocean Alliance",
	"OOL": "Ocean Alliance",
	"COE": "Ocean Alliance",
	"CSE": "Ocean Alliance",
	"CNC": "Ocean Alliance",
	"ANL": "Ocean Alliance",
	"APL": "Ocean Alliance",
	"EGH": "Ocean Alliance",
	"EMA": "Ocean Alliance",
	"EMS": "Ocean Alliance",
	"MSK": "Gemini Cooperation",
	"HLC": "Gemini Cooperation",
	"MCC": "Gemini Cooperation",
	"SGL": "Gemini Cooperation",
	"SLD": "Gemini Cooperation",
	"ONE": "Premier Alliance",
	"YML": "Premier Alliance",
	"HMM": "Premier Alliance",
	"MSC": "Independent",
	"ZIM": "Independent",
	"WHL": "Independent",
	"SIT": "Independent",
	"PIL": "Independent",
	"KMT": "Independent",
	"TSL": "Independent",
	"SNK": "Independent",
	"CUL": "Independent",
	"RCL": "Independent",
	"ESL": "Independent",
	"MAT": "Independent",
}

// topContainerPrefixes 映射各大船公司专有 BIC 集装箱箱主前缀。
// 保证每个组织内各前缀不重复冲突。
var topContainerPrefixes = map[string][]string{
	"COS": {"COSU", "CBHU"},
	"MSK": {"MSKU", "MAEU"},
	"MSC": {"MSCU", "MEDU"},
	"CMA": {"CMAU"},
	"HLC": {"HLCU", "TGHU"},
	"ONE": {"ONEY"},
	"EMC": {"EGLV", "EMCU"},
	"OOL": {"OOLU"},
	"ZIM": {"ZIMU"},
	"WHL": {"WHLU"},
	"SIT": {"SITU"},
	"PIL": {"PCIU"},
	"KMT": {"KMTC"},
	"HMM": {"HDMU"},
	"YML": {"YMLU"},
	"TSL": {"TSLU"},
	"SNK": {"SKLU"},
	"CUL": {"CULU"},
	"ESL": {"ESLU"},
	"MAT": {"MATS"},
	"ASL": {"ASLU"},
}

// explicitCountryByCode 为缺少明确地址文本或知名代码的船司显式指定总部国家。
var explicitCountryByCode = map[string]string{
	"OOL": "HK", // 东方海外 (OOCL) 香港
	"RCL": "TH", // 宏海箱运 (RCL) 泰国
	"CAV": "CL", // 智利南美邮船 (CSAV) 智利
	"EIM": "IS", // Eimskip 冰岛
	"GUA": "PY", // Guaran Feeder 巴拉圭
	"INS": "PY", // Independencia 巴拉圭
	"TFP": "PY", // Transporte Fluvial 巴拉圭
	"VEL": "UY", // Velmaren 乌拉圭
	"VEN": "VE", // Venavega 委内瑞拉
	"WRN": "BN", // Warisan 汶莱
	"RAL": "GL", // Royal Arctic 格陵兰
	"HED": "CN", // 河北海德 中国
	"LSS": "CN", // 连云港海运 中国
	"KWL": "HK", // 卡瓦船务 香港
	"AAS": "HK", // Andaman Asia 香港
	"OKL": "CN", // 必胜物流 中国
	"MAR": "AR", // Maruba 阿根廷
	"CFS": "US", // Caribbean Feeder Service 美国
	"CMC": "US", // Crowley Maritime 美国
	"FHF": "US", // Fjord Havn Feeders 美国
	"KJC": "US", // Kuk Jae Transportation 美国
	"KOS": "US", // King Ocean Services 美国
	"PAS": "US", // Pasha Lines 美国
	"UFE": "DK", // Unifeeder 丹麦
	"TRK": "TR", // Turkon Line 土耳其
	"DBD": "TR", // Turkish Cargo Lines 土耳其
	"AKN": "TR", // AKKON Lines 土耳其
	"SET": "IT", // Setramar 意大利
	"TAR": "IT", // Tarros 意大利
	"GNC": "IT", // Grimaldi 意大利
	"CSI": "IT", // Cosiarma 意大利
	"ELS": "SG", // Eng Lee Shipping 新加坡
	"APS": "SG", // Alpine Shipping 新加坡
	"RLS": "SG", // Routes Logistics 新加坡
	"WFL": "MY", // Winfast Lines 马来西亚
	"SAI": "VN", // Saigon Shipping 越南
	"ECG": "GB", // Ellerman City Liner 英国
	"GEE": "GB", // Geest Line 英国
	"MLM": "GB", // Mann Lines 英国
	"HST": "DE", // Hugo Stinnes 德国
	"RTS": "DE", // RTSB Line 德国
	"ESG": "DE", // Eshipping Gateway 德国
	"HKS": "FI", // Hacklin Seatrans 芬兰
	"FCS": "ES", // Fred-Olsen 西班牙
	"HAD": "IR", // Hafez Darya 伊朗
	"HDS": "IR", // Hafiz Darya 伊朗
	"ISL": "IR", // IRISL 伊朗
	"ISC": "IL", // ISCONT 以色列
	"HYS": "KR", // Hae Yang Shipping 韩国
	"LBA": "BR", // Libra 巴西
	"AND": "BR", // Andes 巴西
	"ALI": "BR", // Alianca 巴西
	"KMA": "TT", // Coastal Shipping 特立尼达和多巴哥
	"OAC": "ZA", // Ocean Africa 南非
	"OBS": "NZ", // Ocean Bridge 新西兰
	"PFL": "NZ", // Pacific Forum Line 新西兰
	"PLS": "US", // PNS Logistics 美国
	"PSL": "PT", // PSL Navegacao 葡萄牙
	"QCL": "IN", // Quest Container Line 印度
	"RHE": "DE", // Rhenus Logistics 德国
	"CNA": "DZ", // CNAN 阿尔及利亚
	"ECL": "NO", // Euro Container Lines 挪威
	"NGP": "PG", // New Guinea Pacific 巴布亚新几内亚
	"NLS": "GR", // Neptune Lines 希腊
}

var countryKeywords = []struct {
	keyword string
	country string
}{
	{"singapore", "SG"}, {"hong kong", "HK"}, {"kowloon", "HK"},
	{"shanghai", "CN"}, {"shenzhen", "CN"}, {"ningbo", "CN"}, {"qingdao", "CN"},
	{"tianjin", "CN"}, {"xiamen", "CN"}, {"guangzhou", "CN"}, {"dalian", "CN"},
	{"beijing", "CN"}, {"china", "CN"}, {"p.r.c", "CN"}, {"prc", "CN"},
	{"taiwan", "TW"}, {"taipei", "TW"},
	{"united states", "US"}, {"usa", "US"}, {"u.s.a", "US"}, {"westfield", "US"},
	{"chicago", "US"}, {"florida", "US"}, {"california", "US"}, {"new jersey", "US"},
	{"germany", "DE"}, {"hamburg", "DE"}, {"buxtehude", "DE"}, {"bremen", "DE"},
	{"deutschland", "DE"}, {"france", "FR"}, {"marseille", "FR"}, {"paris", "FR"},
	{"denmark", "DK"}, {"copenhagen", "DK"}, {"københavn", "DK"},
	{"japan", "JP"}, {"tokyo", "JP"}, {"korea", "KR"}, {"seoul", "KR"}, {"busan", "KR"},
	{"netherlands", "NL"}, {"rotterdam", "NL"}, {"united kingdom", "GB"}, {"london", "GB"},
	{"england", "GB"}, {"scotland", "GB"}, {"uae", "AE"}, {"u.a.e", "AE"}, {"dubai", "AE"},
	{"abu dhabi", "AE"}, {"emirates", "AE"}, {"switzerland", "CH"}, {"geneva", "CH"},
	{"indonesia", "ID"}, {"jakarta", "ID"}, {"malaysia", "MY"}, {"australia", "AU"},
	{"melbourne", "AU"}, {"sydney", "AU"}, {"brazil", "BR"}, {"brasil", "BR"},
	{"santos", "BR"}, {"sao paulo", "BR"}, {"turkey", "TR"}, {"turkiye", "TR"},
	{"türkiye", "TR"}, {"istanbul", "TR"}, {"spain", "ES"}, {"madrid", "ES"},
	{"barcelona", "ES"}, {"valencia", "ES"}, {"italy", "IT"}, {"genova", "IT"},
	{"israel", "IL"}, {"haifa", "IL"}, {"vietnam", "VN"}, {"thailand", "TH"},
	{"bangkok", "TH"}, {"qatar", "QA"}, {"doha", "QA"}, {"kuwait", "KW"},
	{"saudi arabia", "SA"}, {"russia", "RU"}, {"moscow", "RU"}, {"st.petersburg", "RU"},
	{"india", "IN"}, {"mumbai", "IN"}, {"delhi", "IN"}, {"malta", "MT"},
	{"valletta", "MT"}, {"cuba", "CU"}, {"egypt", "EG"}, {"chile", "CL"},
	{"canada", "CA"}, {"norway", "NO"}, {"sweden", "SE"}, {"finland", "FI"},
	{"belgium", "BE"}, {"antwerp", "BE"}, {"philippines", "PH"}, {"greece", "GR"},
	{"mauritius", "MU"}, {"ghana", "GH"}, {"tema", "GH"}, {"morocco", "MA"},
	{"new zealand", "NZ"}, {"pakistan", "PK"}, {"sri lanka", "LK"}, {"colombia", "CO"},
	{"cyprus", "CY"}, {"limassol", "CY"}, {"portugal", "PT"}, {"ireland", "IE"},
	{"south africa", "ZA"}, {"panama", "PA"}, {"mexico", "MX"}, {"argentina", "AR"},
	{"peru", "PE"}, {"ecuador", "EC"}, {"poland", "PL"}, {"jordan", "JO"},
	{"oman", "OM"}, {"bahrain", "BH"}, {"bangladesh", "BD"}, {"myanmar", "MM"},
	{"cambodia", "KH"}, {"georgia", "GE"}, {"monaco", "MC"}, {"algeria", "DZ"},
	{"kenya", "KE"}, {"nigeria", "NG"}, {"lebanon", "LB"}, {"lithuania", "LT"},
	{"latvia", "LV"}, {"estonia", "EE"}, {"slovenia", "SI"}, {"croatia", "HR"},
	{"romania", "RO"}, {"bulgaria", "BG"}, {"ukraine", "UA"}, {"togo", "TG"},
	{"benin", "BJ"}, {"cameroon", "CM"}, {"senegal", "SN"}, {"cote d'ivoire", "CI"},
	{"angola", "AO"}, {"mozambique", "MZ"}, {"tanzania", "TZ"}, {"djibouti", "DJ"},
	{"sudan", "SD"}, {"yemen", "YE"}, {"maldives", "MV"},
}

func resolveCountryCode(code, name, address, website string) string {
	if explicit, ok := explicitCountryByCode[code]; ok {
		return explicit
	}
	text := strings.ToLower(address + " " + name + " " + website)
	for _, entry := range countryKeywords {
		if strings.Contains(text, entry.keyword) {
			return entry.country
		}
	}
	// 兜底国家代码：联合国国际组织标号
	return "UN"
}

func resolveChineseName(code, nameEN string) string {
	if zh, ok := topLinersZH[code]; ok && zh != "" {
		return zh
	}
	return nameEN
}

func resolveAlliance(code string) *string {
	if alliance, ok := topAlliances[code]; ok && alliance != "" {
		return &alliance
	}
	return nil
}

func resolveContainerPrefixes(code string) []string {
	if prefixes, ok := topContainerPrefixes[code]; ok {
		return prefixes
	}
	return nil
}

func resolveTrackingURL(raw string) *string {
	clean := strings.TrimSpace(raw)
	if clean == "" || strings.EqualFold(clean, "not available") || clean == "-" {
		return nil
	}
	if !httpProtocolExpr.MatchString(clean) {
		clean = "https://" + clean
	}
	parsed, err := url.ParseRequestURI(clean)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil
	}
	return &clean
}
