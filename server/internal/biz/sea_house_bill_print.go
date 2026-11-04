package biz

import (
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
)

const SeaHouseBillPrintTemplateRoncinBeijing = "RONCIN_BEIJING"

// ResolveSeaHouseBillPrintTemplate 根据真实签发公司代码确定本期适用模板。
func ResolveSeaHouseBillPrintTemplate(source SeaHouseBillIssuerSource, issuerCode string) string {
	if source == SeaHouseBillIssuerSourceSelfOrganization && issuerCode == "BJ" {
		return SeaHouseBillPrintTemplateRoncinBeijing
	}
	return ""
}

// SeaHouseBillPrintDetails 保存本票印字，不承载共享航程或另一份提单正文。
type SeaHouseBillPrintDetails struct {
	HeaderContactText    *string
	BookingNoText        *string
	ExportReferencesText *string
	FooterReferenceText  *string
	PreCarriageText      *string
	PlaceOfReceiptText   *string
	DomesticRoutingText  *string
	PointOfOriginText    *string
	OnwardRoutingText    *string
	VesselVoyageText     *string
	PortOfLoadingText    *string
	PortOfDischargeText  *string
	PlaceOfDeliveryText  *string
	ContainersText       *string
	OriginalCount        *int32
	ShippedOnBoardDate   *string
	IssuePlaceText       *string
	IssueDate            *string
	DeliveryContactText  *string
}

// ValidateSeaHouseBillPrintDetails 校验印字字段，空值保持未填写语义。
func ValidateSeaHouseBillPrintDetails(input *SeaHouseBillPrintDetails) (*SeaHouseBillPrintDetails, error) {
	if input == nil {
		return nil, nil
	}
	result := *input
	var err error
	normalize := func(label string, value *string, max int) (*string, error) {
		if value == nil {
			return nil, nil
		}
		text := strings.TrimSpace(*value)
		if utf8.RuneCountInString(text) > max {
			return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", fmt.Sprintf("%s不能超过%d个字符", label, max))
		}
		if strings.ContainsFunc(text, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' }) {
			return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", label+"包含无效控制字符")
		}
		if text == "" {
			return nil, nil
		}
		return &text, nil
	}
	if result.HeaderContactText, err = normalize("页眉联系方式", input.HeaderContactText, 3000); err != nil {
		return nil, err
	}
	if result.BookingNoText, err = normalize("订舱号印字", input.BookingNoText, 500); err != nil {
		return nil, err
	}
	if result.ExportReferencesText, err = normalize("出口参考", input.ExportReferencesText, 3000); err != nil {
		return nil, err
	}
	if result.FooterReferenceText, err = normalize("页脚参考", input.FooterReferenceText, 500); err != nil {
		return nil, err
	}
	if result.PreCarriageText, err = normalize("前程运输", input.PreCarriageText, 500); err != nil {
		return nil, err
	}
	if result.PlaceOfReceiptText, err = normalize("收货地", input.PlaceOfReceiptText, 500); err != nil {
		return nil, err
	}
	if result.DomesticRoutingText, err = normalize("内陆运输指示", input.DomesticRoutingText, 3000); err != nil {
		return nil, err
	}
	if result.PointOfOriginText, err = normalize("原产地", input.PointOfOriginText, 500); err != nil {
		return nil, err
	}
	if result.OnwardRoutingText, err = normalize("后程运输", input.OnwardRoutingText, 3000); err != nil {
		return nil, err
	}
	if result.VesselVoyageText, err = normalize("船名航次印字", input.VesselVoyageText, 500); err != nil {
		return nil, err
	}
	if result.PortOfLoadingText, err = normalize("装货港印字", input.PortOfLoadingText, 500); err != nil {
		return nil, err
	}
	if result.PortOfDischargeText, err = normalize("卸货港印字", input.PortOfDischargeText, 500); err != nil {
		return nil, err
	}
	if result.PlaceOfDeliveryText, err = normalize("交货地印字", input.PlaceOfDeliveryText, 500); err != nil {
		return nil, err
	}
	if result.ContainersText, err = normalize("箱号明细", input.ContainersText, 12000); err != nil {
		return nil, err
	}
	if result.ShippedOnBoardDate, err = normalize("装船日期", input.ShippedOnBoardDate, 10); err != nil {
		return nil, err
	}
	if result.IssuePlaceText, err = normalize("签发地", input.IssuePlaceText, 500); err != nil {
		return nil, err
	}
	if result.IssueDate, err = normalize("签发日期", input.IssueDate, 10); err != nil {
		return nil, err
	}
	if result.DeliveryContactText, err = normalize("交货联系人", input.DeliveryContactText, 3000); err != nil {
		return nil, err
	}
	if input.OriginalCount != nil && (*input.OriginalCount < 0 || *input.OriginalCount > 99) {
		return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", "正本份数必须在0至99之间")
	}
	for _, date := range []struct {
		label string
		value *string
	}{{"装船日期", result.ShippedOnBoardDate}, {"签发日期", result.IssueDate}} {
		if date.value == nil {
			continue
		}
		parsed, err := time.Parse("2006-01-02", *date.value)
		if err != nil || parsed.Format("2006-01-02") != *date.value {
			return nil, errors.BadRequest("SEA_HOUSE_BILL_INVALID_ARGUMENT", date.label+"必须为合法的YYYY-MM-DD日期")
		}
	}
	return &result, nil
}
