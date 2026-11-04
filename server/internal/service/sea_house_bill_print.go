package service

import (
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func seaHouseBillPrintDetailsToAPI(input *biz.SeaHouseBillPrintDetails) *v1.SeaHouseBillPrintDetails {
	if input == nil {
		return nil
	}
	return &v1.SeaHouseBillPrintDetails{
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

func seaHouseBillPrintDetailsFromAPI(input *v1.SeaHouseBillPrintDetails) *biz.SeaHouseBillPrintDetails {
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
