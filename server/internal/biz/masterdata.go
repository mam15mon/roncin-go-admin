package biz

import (
	"context"
	"math/big"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrMasterDataInvalidArgument      = errors.BadRequest("MASTER_DATA_INVALID_ARGUMENT", "主数据字段不合法")
	ErrMasterDataNotFound             = errors.NotFound("MASTER_DATA_NOT_FOUND", "主数据不存在")
	ErrMasterDataCodeExists           = errors.Conflict("MASTER_DATA_CODE_EXISTS", "主数据编码已存在")
	ErrMasterDataInvalidKind          = errors.BadRequest("MASTER_DATA_INVALID_KIND", "主数据类型不合法")
	ErrMasterDataHeadquartersRequired = errors.Forbidden("MASTER_DATA_HEADQUARTERS_REQUIRED", "主数据只能由总部维护")
)

type MasterDataKind string

const (
	MasterDataKindCurrency      MasterDataKind = "currency"
	MasterDataKindCountry       MasterDataKind = "country"
	MasterDataKindRegion        MasterDataKind = "region"
	MasterDataKindContainerSpec MasterDataKind = "container_spec"
	MasterDataKindServiceType   MasterDataKind = "service_type"
	MasterDataKindCargoCategory MasterDataKind = "cargo_category"
	MasterDataKindAbnormalCase  MasterDataKind = "abnormal_case"
)

func (kind MasterDataKind) Valid() bool {
	switch kind {
	case MasterDataKindCurrency, MasterDataKindCountry, MasterDataKindRegion, MasterDataKindContainerSpec, MasterDataKindServiceType, MasterDataKindCargoCategory, MasterDataKindAbnormalCase:
		return true
	default:
		return false
	}
}

type MasterDataItem struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Kind           MasterDataKind
	Code           string
	Name           string
	NameEN         *string
	ParentCode     *string
	TEUFactor      *string
	Attributes     MasterDataAttributes
	Source         string
	SortOrder      int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func DefaultOrderOptions() []MasterDataItem {
	containerSpecs := []struct{ code, teuFactor string }{
		{"20GP", "1"},
		{"40HQ", "2"},
		{"20HC", "1"},
		{"20OT", "1"},
		{"20FR", "1"},
		{"20RF", "1"},
		{"20TK", "1"},
		{"20HT", "1"},
		{"20RH", "1"},
		{"40FR", "2"},
		{"40GP", "2"},
		{"40PF", "2"},
		{"40RF", "2"},
		{"40OT", "2"},
		{"40RH", "2"},
		{"45HC", "2.25"},
		{"12GP", "0.6"},
		{"40HC", "2"},
	}
	serviceTypes := []struct{ code, name string }{
		{"BOOKING", "订舱"},
		{"TRUCKING", "拖车"},
		{"STUFFING", "内装"},
		{"CUSTOMS_EXPORT", "报关"},
		{"CUSTOMS_IMPORT", "清关"},
		{"OVERSEA_SEGMENT", "海外段"},
		{"INSURANCE", "保险"},
		{"PALLET_CHARTER", "包板"},
		{"CONTAINER_LEASE", "租箱"},
		{"FUMIGATION", "熏蒸"},
		{"DOC_BUY", "买单"},
		{"CERTIFICATE", "办证"},
		{"DOC_PREP", "制单"},
		{"DANGEROUS_SERVICE", "危险品"},
		{"OVERWEIGHT_SERVICE", "超重"},
		{"DOCUMENT_EXCHANGE", "换单"},
		{"WAREHOUSING", "仓储"},
		{"INSPECTION", "报检"},
		{"CONTAINER_PURCHASE", "买箱"},
	}
	cargoCategories := []struct{ code, name string }{
		{"GENERAL", "普货"},
		{"REEFER", "冷藏货物"},
		{"OVERSIZE", "超限货"},
		{"DANGEROUS", "危险品"},
		{"BREAK_BULK_PIECE", "散杂件货"},
	}

	items := make([]MasterDataItem, 0, len(containerSpecs)+len(serviceTypes)+len(cargoCategories))
	for index, item := range containerSpecs {
		teuFactor := item.teuFactor
		items = append(items, MasterDataItem{Kind: MasterDataKindContainerSpec, Code: item.code, Name: item.code, TEUFactor: &teuFactor, Source: "system", SortOrder: (index + 1) * 10, Enabled: true})
	}
	for index, item := range serviceTypes {
		items = append(items, MasterDataItem{Kind: MasterDataKindServiceType, Code: item.code, Name: item.name, Source: "system", SortOrder: (index + 1) * 10, Enabled: true})
	}
	for index, item := range cargoCategories {
		items = append(items, MasterDataItem{Kind: MasterDataKindCargoCategory, Code: item.code, Name: item.name, Source: "system", SortOrder: (index + 1) * 10, Enabled: true})
	}
	return items
}

// DefaultCountryOptions 返回系统预置的全球常用国家与地区主数据字典（覆盖全球 50+ 主要贸易伙伴国）。
func DefaultCountryOptions() []MasterDataItem {
	rawCountries := []struct {
		code         string
		name         string
		nameEn       string
		continent    string
		currencyCode string
	}{
		// 亚洲 (Asia)
		{"CN", "中国", "China", "Asia", "CNY"},
		{"HK", "中国香港", "Hong Kong", "Asia", "HKD"},
		{"MO", "中国澳门", "Macao", "Asia", "MOP"},
		{"TW", "中国台湾", "Taiwan", "Asia", "TWD"},
		{"JP", "日本", "Japan", "Asia", "JPY"},
		{"KR", "韩国", "South Korea", "Asia", "KRW"},
		{"SG", "新加坡", "Singapore", "Asia", "SGD"},
		{"MY", "马来西亚", "Malaysia", "Asia", "MYR"},
		{"TH", "泰国", "Thailand", "Asia", "THB"},
		{"VN", "越南", "Vietnam", "Asia", "VND"},
		{"ID", "印度尼西亚", "Indonesia", "Asia", "IDR"},
		{"PH", "菲律宾", "Philippines", "Asia", "PHP"},
		{"IN", "印度", "India", "Asia", "INR"},
		{"PK", "巴基斯坦", "Pakistan", "Asia", "PKR"},
		{"BD", "孟加拉国", "Bangladesh", "Asia", "BDT"},
		{"AE", "阿联酋", "United Arab Emirates", "Asia", "AED"},
		{"SA", "沙特阿拉伯", "Saudi Arabia", "Asia", "SAR"},
		{"QA", "卡塔尔", "Qatar", "Asia", "QAR"},
		{"TR", "土耳其", "Turkey", "Asia", "TRY"},
		{"IL", "以色列", "Israel", "Asia", "ILS"},
		{"KZ", "哈萨克斯坦", "Kazakhstan", "Asia", "KZT"},
		{"UZ", "乌兹别克斯坦", "Uzbekistan", "Asia", "UZS"},

		// 欧洲 (Europe)
		{"DE", "德国", "Germany", "Europe", "EUR"},
		{"GB", "英国", "United Kingdom", "Europe", "GBP"},
		{"FR", "法国", "France", "Europe", "EUR"},
		{"IT", "意大利", "Italy", "Europe", "EUR"},
		{"NL", "荷兰", "Netherlands", "Europe", "EUR"},
		{"BE", "比利时", "Belgium", "Europe", "EUR"},
		{"ES", "西班牙", "Spain", "Europe", "EUR"},
		{"PL", "波兰", "Poland", "Europe", "PLN"},
		{"RU", "俄罗斯", "Russia", "Europe", "RUB"},
		{"CH", "瑞士", "Switzerland", "Europe", "CHF"},
		{"SE", "瑞典", "Sweden", "Europe", "SEK"},
		{"NO", "挪威", "Norway", "Europe", "NOK"},
		{"DK", "丹麦", "Denmark", "Europe", "DKK"},
		{"FI", "芬兰", "Finland", "Europe", "EUR"},
		{"AT", "奥地利", "Austria", "Europe", "EUR"},
		{"CZ", "捷克", "Czech Republic", "Europe", "CZK"},
		{"HU", "匈牙利", "Hungary", "Europe", "HUF"},
		{"GR", "希腊", "Greece", "Europe", "EUR"},
		{"PT", "葡萄牙", "Portugal", "Europe", "EUR"},
		{"IE", "爱尔兰", "Ireland", "Europe", "EUR"},

		// 北美洲 (North America)
		{"US", "美国", "United States", "North America", "USD"},
		{"CA", "加拿大", "Canada", "North America", "CAD"},
		{"MX", "墨西哥", "Mexico", "North America", "MXN"},
		{"PA", "巴拿马", "Panama", "North America", "USD"},

		// 南美洲 (South America)
		{"BR", "巴西", "Brazil", "South America", "BRL"},
		{"CL", "智利", "Chile", "South America", "CLP"},
		{"AR", "阿根廷", "Argentina", "South America", "ARS"},
		{"PE", "秘鲁", "Peru", "South America", "PEN"},
		{"CO", "哥伦比亚", "Colombia", "South America", "COP"},

		// 大洋洲 (Oceania)
		{"AU", "澳大利亚", "Australia", "Oceania", "AUD"},
		{"NZ", "新西兰", "New Zealand", "Oceania", "NZD"},

		// 非洲 (Africa)
		{"EG", "埃及", "Egypt", "Africa", "EGP"},
		{"ZA", "南非", "South Africa", "Africa", "ZAR"},
		{"NG", "尼日利亚", "Nigeria", "Africa", "NGN"},
		{"KE", "肯尼亚", "Kenya", "Africa", "KES"},
		{"MA", "摩洛哥", "Morocco", "Africa", "MAD"},
		{"GH", "加纳", "Ghana", "Africa", "GHS"},
	}

	items := make([]MasterDataItem, 0, len(rawCountries))
	for index, item := range rawCountries {
		nameEn := item.nameEn
		continent := item.continent
		currencyCode := item.currencyCode
		items = append(items, MasterDataItem{
			Kind:      MasterDataKindCountry,
			Code:      item.code,
			Name:      item.name,
			NameEN:    &nameEn,
			Source:    "system",
			SortOrder: (index + 1) * 10,
			Enabled:   true,
			Attributes: MasterDataAttributes{
				Continent:    &continent,
				CurrencyCode: &currencyCode,
			},
		})
	}
	return items
}

type MasterDataAttributes struct {
	Continent    *string
	CurrencyCode *string
	RegionLevel  *int
}

type MasterDataListOptions struct {
	Page     int
	PageSize int
	Kind     MasterDataKind
	Keyword  string
	Enabled  *bool
}

type MasterDataList = PagedList[*MasterDataItem]

type MasterDataImportMode string

const (
	MasterDataImportModeCreateOnly MasterDataImportMode = "create_only"
	MasterDataImportModeUpsert     MasterDataImportMode = "upsert"
)

func (mode MasterDataImportMode) Valid() bool {
	return mode == MasterDataImportModeCreateOnly || mode == MasterDataImportModeUpsert
}

type MasterDataImportInput struct {
	Kind   MasterDataKind
	Source string
	Mode   MasterDataImportMode
	Items  []*MasterDataItem
}

type MasterDataImportResult struct {
	Items   []*MasterDataItem
	Created int
	Updated int
}

type MasterDataRepo interface {
	List(context.Context, uuid.UUID, MasterDataListOptions) (*MasterDataList, error)
	ListEnabled(context.Context, uuid.UUID) ([]*MasterDataItem, error)
	Create(context.Context, uuid.UUID, *MasterDataItem, *AuditEvent) (*MasterDataItem, error)
	Update(context.Context, uuid.UUID, uuid.UUID, *MasterDataItem, *AuditEvent) (*MasterDataItem, error)
	Import(context.Context, uuid.UUID, MasterDataImportMode, []*MasterDataItem, *AuditEvent) (*MasterDataImportResult, error)
}

type MasterDataUsecase struct {
	repo MasterDataRepo
}

func NewMasterDataUsecase(repo MasterDataRepo) *MasterDataUsecase {
	return &MasterDataUsecase{repo: repo}
}

func (uc *MasterDataUsecase) List(ctx context.Context, organizationID uuid.UUID, options MasterDataListOptions) (*MasterDataList, error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrMasterDataInvalidArgument
	}
	if options.Kind != "" && !options.Kind.Valid() {
		return nil, ErrMasterDataInvalidKind
	}
	options.Keyword = strings.TrimSpace(options.Keyword)
	return uc.repo.List(ctx, organizationID, options)
}

func (uc *MasterDataUsecase) ListOptions(ctx context.Context, organizationID uuid.UUID) ([]*MasterDataItem, error) {
	if organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	return uc.repo.ListEnabled(ctx, organizationID)
}

func (uc *MasterDataUsecase) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *MasterDataItem) (*MasterDataItem, error) {
	if organizationID == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	normalized, err := normalizeMasterDataItem(input, true)
	if err != nil {
		return nil, err
	}
	audit := &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "master_data.create", Result: "success", Details: map[string]string{"master_data.kind": string(normalized.Kind), "master_data.code": normalized.Code}}
	created, err := uc.repo.Create(ctx, organizationID, normalized, audit)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (uc *MasterDataUsecase) Import(ctx context.Context, organizationID, actorID uuid.UUID, input MasterDataImportInput) (*MasterDataImportResult, error) {
	input.Source = strings.TrimSpace(input.Source)
	if organizationID == uuid.Nil || !input.Kind.Valid() || !input.Mode.Valid() || input.Source == "" || utf8.RuneCountInString(input.Source) > 100 || len(input.Items) == 0 || len(input.Items) > 500 {
		return nil, ErrMasterDataInvalidArgument
	}
	items := make([]*MasterDataItem, 0, len(input.Items))
	seen := make(map[string]struct{}, len(input.Items))
	for _, item := range input.Items {
		if item == nil {
			return nil, ErrMasterDataInvalidArgument
		}
		copyItem := *item
		copyItem.Kind = input.Kind
		copyItem.Source = input.Source
		normalized, err := normalizeMasterDataItem(&copyItem, true)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[normalized.Code]; exists {
			return nil, ErrMasterDataInvalidArgument
		}
		seen[normalized.Code] = struct{}{}
		items = append(items, normalized)
	}
	audit := &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "master_data.import", Result: "success", Details: map[string]string{"master_data.kind": string(input.Kind), "source": input.Source, "mode": string(input.Mode)}}
	result, err := uc.repo.Import(ctx, organizationID, input.Mode, items, audit)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (uc *MasterDataUsecase) Update(ctx context.Context, organizationID, actorID, id uuid.UUID, input *MasterDataItem) (*MasterDataItem, error) {
	if id == uuid.Nil {
		return nil, ErrMasterDataInvalidArgument
	}
	normalized, err := normalizeMasterDataItem(input, false)
	if err != nil {
		return nil, err
	}
	audit := &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: "master_data.update", Result: "success", Details: map[string]string{"master_data.id": id.String(), "master_data.kind": string(normalized.Kind)}}
	updated, err := uc.repo.Update(ctx, organizationID, id, normalized, audit)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

func normalizeMasterDataItem(input *MasterDataItem, creating bool) (*MasterDataItem, error) {
	if input == nil {
		return nil, ErrMasterDataInvalidArgument
	}
	output := *input
	output.Code = strings.ToUpper(strings.TrimSpace(output.Code))
	output.Name = strings.TrimSpace(output.Name)
	output.Source = strings.TrimSpace(output.Source)
	if output.Source == "" {
		output.Source = "manual"
	}
	output.NameEN = normalizedOptionalString(output.NameEN)
	output.ParentCode = normalizedUpperOptionalString(output.ParentCode)
	output.TEUFactor = normalizedOptionalString(output.TEUFactor)
	if err := normalizeMasterDataAttributes(output.Kind, &output.Attributes); err != nil {
		return nil, err
	}
	if !output.Kind.Valid() || output.Name == "" || utf8.RuneCountInString(output.Name) > 200 || output.SortOrder < 0 || utf8.RuneCountInString(output.Source) > 100 {
		return nil, ErrMasterDataInvalidArgument
	}
	if creating && (output.Code == "" || utf8.RuneCountInString(output.Code) > 64) {
		return nil, ErrMasterDataInvalidArgument
	}
	if optionalStringTooLong(output.NameEN, 200) || optionalStringTooLong(output.ParentCode, 64) || optionalStringTooLong(output.TEUFactor, 32) {
		return nil, ErrMasterDataInvalidArgument
	}
	if output.ParentCode != nil && output.Kind != MasterDataKindRegion {
		return nil, ErrMasterDataInvalidArgument
	}
	if output.TEUFactor != nil {
		if output.Kind != MasterDataKindContainerSpec {
			return nil, ErrMasterDataInvalidArgument
		}
		value, ok := new(big.Rat).SetString(*output.TEUFactor)
		if !ok || value.Sign() <= 0 {
			return nil, ErrMasterDataInvalidArgument
		}
	}
	return &output, nil
}

func normalizeMasterDataAttributes(kind MasterDataKind, attributes *MasterDataAttributes) error {
	attributes.Continent = normalizedOptionalString(attributes.Continent)
	attributes.CurrencyCode = normalizedUpperOptionalString(attributes.CurrencyCode)
	if optionalStringTooLong(attributes.Continent, 32) || optionalStringTooLong(attributes.CurrencyCode, 3) {
		return ErrMasterDataInvalidArgument
	}
	if attributes.RegionLevel != nil && (*attributes.RegionLevel < 1 || *attributes.RegionLevel > 3) {
		return ErrMasterDataInvalidArgument
	}

	switch kind {
	case MasterDataKindCountry:
		if attributes.RegionLevel != nil {
			return ErrMasterDataInvalidArgument
		}
	case MasterDataKindRegion:
		if attributes.Continent != nil || attributes.CurrencyCode != nil {
			return ErrMasterDataInvalidArgument
		}
	default:
		if attributes.Continent != nil || attributes.CurrencyCode != nil || attributes.RegionLevel != nil {
			return ErrMasterDataInvalidArgument
		}
	}
	return nil
}

func optionalStringTooLong(value *string, maximum int) bool {
	return value != nil && utf8.RuneCountInString(*value) > maximum
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func normalizedUpperOptionalString(value *string) *string {
	normalized := normalizedOptionalString(value)
	if normalized == nil {
		return nil
	}
	upper := strings.ToUpper(*normalized)
	return &upper
}

var _ MasterDataRepo = (MasterDataRepo)(nil)
