package data

import (
	"encoding/json"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/schema"
)

func seaHouseBillPrintDetailsToPO(input *biz.SeaHouseBillPrintDetails) *schema.SeaHouseBillPrintDetails {
	if input == nil {
		return nil
	}
	return &schema.SeaHouseBillPrintDetails{
		HeaderContactText:    input.HeaderContactText,
		BookingNoText:        input.BookingNoText,
		ExportReferencesText: input.ExportReferencesText,
		FooterReferenceText:  input.FooterReferenceText,
		PreCarriageText:      input.PreCarriageText,
		PlaceOfReceiptText:   input.PlaceOfReceiptText,
		DomesticRoutingText:  input.DomesticRoutingText,
		PointOfOriginText:    input.PointOfOriginText,
		OnwardRoutingText:    input.OnwardRoutingText,
		VesselVoyageText:     input.VesselVoyageText,
		PortOfLoadingText:    input.PortOfLoadingText,
		PortOfDischargeText:  input.PortOfDischargeText,
		PlaceOfDeliveryText:  input.PlaceOfDeliveryText,
		ContainersText:       input.ContainersText,
		OriginalCount:        input.OriginalCount,
		ShippedOnBoardDate:   input.ShippedOnBoardDate,
		IssuePlaceText:       input.IssuePlaceText,
		IssueDate:            input.IssueDate,
		DeliveryContactText:  input.DeliveryContactText,
	}
}

func seaHouseBillPrintDetailsToBiz(input *schema.SeaHouseBillPrintDetails) *biz.SeaHouseBillPrintDetails {
	if input == nil {
		return nil
	}
	return &biz.SeaHouseBillPrintDetails{
		HeaderContactText:    input.HeaderContactText,
		BookingNoText:        input.BookingNoText,
		ExportReferencesText: input.ExportReferencesText,
		FooterReferenceText:  input.FooterReferenceText,
		PreCarriageText:      input.PreCarriageText,
		PlaceOfReceiptText:   input.PlaceOfReceiptText,
		DomesticRoutingText:  input.DomesticRoutingText,
		PointOfOriginText:    input.PointOfOriginText,
		OnwardRoutingText:    input.OnwardRoutingText,
		VesselVoyageText:     input.VesselVoyageText,
		PortOfLoadingText:    input.PortOfLoadingText,
		PortOfDischargeText:  input.PortOfDischargeText,
		PlaceOfDeliveryText:  input.PlaceOfDeliveryText,
		ContainersText:       input.ContainersText,
		OriginalCount:        input.OriginalCount,
		ShippedOnBoardDate:   input.ShippedOnBoardDate,
		IssuePlaceText:       input.IssuePlaceText,
		IssueDate:            input.IssueDate,
		DeliveryContactText:  input.DeliveryContactText,
	}
}

// seaHouseBillPrintTemplate 只依据真实签发主体，不依据订单显示名称猜测品牌。
func seaHouseBillPrintTemplate(item *ent.SeaHouseBill, issuerKind biz.OrganizationKind) string {
	if item.IssuerOrganizationID == nil {
		return ""
	}
	return biz.ResolveSeaHouseBillPrintTemplate(biz.SeaHouseBillIssuerSource(item.IssuerSource), issuerKind)
}

func setSeaHouseBillPrintDetailsUpdate(builder *ent.SeaHouseBillUpdateOne, details *biz.SeaHouseBillPrintDetails) {
	if details == nil {
		builder.ClearPrintDetails()
		return
	}
	builder.SetPrintDetails(seaHouseBillPrintDetailsToPO(details))
}

func diffHousePrintDetails(before, after *biz.SeaHouseBillPrintDetails) []*biz.SeaDocumentFieldDifference {
	if before == nil {
		before = &biz.SeaHouseBillPrintDetails{}
	}
	if after == nil {
		after = &biz.SeaHouseBillPrintDetails{}
	}
	result := []*biz.SeaDocumentFieldDifference{}
	addDiff(&result, "print_details.header_contact_text", "页眉联系方式", documentStringValue(before.HeaderContactText), documentStringValue(after.HeaderContactText))
	addDiff(&result, "print_details.booking_no_text", "订舱号印字", documentStringValue(before.BookingNoText), documentStringValue(after.BookingNoText))
	addDiff(&result, "print_details.export_references_text", "出口参考", documentStringValue(before.ExportReferencesText), documentStringValue(after.ExportReferencesText))
	addDiff(&result, "print_details.footer_reference_text", "页脚参考", documentStringValue(before.FooterReferenceText), documentStringValue(after.FooterReferenceText))
	addDiff(&result, "print_details.pre_carriage_text", "前程运输", documentStringValue(before.PreCarriageText), documentStringValue(after.PreCarriageText))
	addDiff(&result, "print_details.place_of_receipt_text", "收货地", documentStringValue(before.PlaceOfReceiptText), documentStringValue(after.PlaceOfReceiptText))
	addDiff(&result, "print_details.domestic_routing_text", "内陆运输指示", documentStringValue(before.DomesticRoutingText), documentStringValue(after.DomesticRoutingText))
	addDiff(&result, "print_details.point_of_origin_text", "原产地", documentStringValue(before.PointOfOriginText), documentStringValue(after.PointOfOriginText))
	addDiff(&result, "print_details.onward_routing_text", "后程运输", documentStringValue(before.OnwardRoutingText), documentStringValue(after.OnwardRoutingText))
	addDiff(&result, "print_details.vessel_voyage_text", "船名航次印字", documentStringValue(before.VesselVoyageText), documentStringValue(after.VesselVoyageText))
	addDiff(&result, "print_details.port_of_loading_text", "装货港印字", documentStringValue(before.PortOfLoadingText), documentStringValue(after.PortOfLoadingText))
	addDiff(&result, "print_details.port_of_discharge_text", "卸货港印字", documentStringValue(before.PortOfDischargeText), documentStringValue(after.PortOfDischargeText))
	addDiff(&result, "print_details.place_of_delivery_text", "交货地印字", documentStringValue(before.PlaceOfDeliveryText), documentStringValue(after.PlaceOfDeliveryText))
	addDiff(&result, "print_details.containers_text", "箱号明细", documentStringValue(before.ContainersText), documentStringValue(after.ContainersText))
	addDiff(&result, "print_details.original_count", "正本份数", int32Value(before.OriginalCount), int32Value(after.OriginalCount))
	addDiff(&result, "print_details.shipped_on_board_date", "装船日期", documentStringValue(before.ShippedOnBoardDate), documentStringValue(after.ShippedOnBoardDate))
	addDiff(&result, "print_details.issue_place_text", "签发地", documentStringValue(before.IssuePlaceText), documentStringValue(after.IssuePlaceText))
	addDiff(&result, "print_details.issue_date", "签发日期", documentStringValue(before.IssueDate), documentStringValue(after.IssueDate))
	addDiff(&result, "print_details.delivery_contact_text", "交货联系人", documentStringValue(before.DeliveryContactText), documentStringValue(after.DeliveryContactText))
	return result
}

func seaHouseBillPrintDetailsHash(input *schema.SeaHouseBillPrintDetails) string {
	if input == nil {
		return ""
	}
	// 固定 PO 只包含字符串和整数，编码不会失败，字段顺序确定。
	value, _ := json.Marshal(input)
	return string(value)
}
