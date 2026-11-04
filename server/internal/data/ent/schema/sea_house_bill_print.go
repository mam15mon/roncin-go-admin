package schema

// SeaHouseBillPrintDetails 是 HBL 印字资料的固定 JSONB 持久化结构。
type SeaHouseBillPrintDetails struct {
	HeaderContactText    *string `json:"header_contact_text,omitempty"`
	BookingNoText        *string `json:"booking_no_text,omitempty"`
	ExportReferencesText *string `json:"export_references_text,omitempty"`
	FooterReferenceText  *string `json:"footer_reference_text,omitempty"`
	PreCarriageText      *string `json:"pre_carriage_text,omitempty"`
	PlaceOfReceiptText   *string `json:"place_of_receipt_text,omitempty"`
	DomesticRoutingText  *string `json:"domestic_routing_text,omitempty"`
	PointOfOriginText    *string `json:"point_of_origin_text,omitempty"`
	OnwardRoutingText    *string `json:"onward_routing_text,omitempty"`
	VesselVoyageText     *string `json:"vessel_voyage_text,omitempty"`
	PortOfLoadingText    *string `json:"port_of_loading_text,omitempty"`
	PortOfDischargeText  *string `json:"port_of_discharge_text,omitempty"`
	PlaceOfDeliveryText  *string `json:"place_of_delivery_text,omitempty"`
	ContainersText       *string `json:"containers_text,omitempty"`
	OriginalCount        *int32  `json:"original_count,omitempty"`
	ShippedOnBoardDate   *string `json:"shipped_on_board_date,omitempty"`
	IssuePlaceText       *string `json:"issue_place_text,omitempty"`
	IssueDate            *string `json:"issue_date,omitempty"`
	DeliveryContactText  *string `json:"delivery_contact_text,omitempty"`
}
