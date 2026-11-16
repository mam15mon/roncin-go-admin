package biz

// 航空印字值独立于运输和经营费用，所有数值保持十进制文本。
type AirDeclaredValue struct {
	Kind   string `json:"kind,omitempty"`
	Amount string `json:"amount,omitempty"`
}
type AirRateLine struct {
	Pieces             string `json:"pieces,omitempty"`
	GrossWeightKg      string `json:"gross_weight_kg,omitempty"`
	RateClass          string `json:"rate_class,omitempty"`
	CommodityItemNo    string `json:"commodity_item_no,omitempty"`
	ChargeableWeightKg string `json:"chargeable_weight_kg,omitempty"`
	Rate               string `json:"rate,omitempty"`
	Amount             string `json:"amount,omitempty"`
	GoodsDescription   string `json:"goods_description,omitempty"`
}
type AirOtherCharge struct {
	Name        string `json:"name,omitempty"`
	BillingUnit string `json:"billing_unit,omitempty"`
	Quantity    string `json:"quantity,omitempty"`
	Currency    string `json:"currency,omitempty"`
	UnitPrice   string `json:"unit_price,omitempty"`
	Amount      string `json:"amount,omitempty"`
	DueTo       string `json:"due_to,omitempty"`
	Payment     string `json:"payment,omitempty"`
}
type AirChargeSummary struct {
	WeightCharge    string `json:"weight_charge,omitempty"`
	ValuationCharge string `json:"valuation_charge,omitempty"`
	Tax             string `json:"tax,omitempty"`
	OtherDueAgent   string `json:"other_due_agent,omitempty"`
	OtherDueCarrier string `json:"other_due_carrier,omitempty"`
	Total           string `json:"total,omitempty"`
}
type AirWaybillContent struct {
	ShipperAccount           string            `json:"shipper_account,omitempty"`
	ShipperText              string            `json:"shipper_text,omitempty"`
	ConsigneeAccount         string            `json:"consignee_account,omitempty"`
	ConsigneeText            string            `json:"consignee_text,omitempty"`
	NotifyPartyText          string            `json:"notify_party_text,omitempty"`
	SecondNotifyPartyText    string            `json:"second_notify_party_text,omitempty"`
	AgentText                string            `json:"agent_text,omitempty"`
	AgentIataCode            string            `json:"agent_iata_code,omitempty"`
	AgentAccount             string            `json:"agent_account,omitempty"`
	IssuedBy                 string            `json:"issued_by,omitempty"`
	DepartureText            string            `json:"departure_text,omitempty"`
	RouteTo                  string            `json:"route_to,omitempty"`
	FirstCarrier             string            `json:"first_carrier,omitempty"`
	RouteTo2                 string            `json:"route_to_2,omitempty"`
	Carrier2                 string            `json:"carrier_2,omitempty"`
	RouteTo3                 string            `json:"route_to_3,omitempty"`
	Carrier3                 string            `json:"carrier_3,omitempty"`
	DestinationText          string            `json:"destination_text,omitempty"`
	FlightText               string            `json:"flight_text,omitempty"`
	FlightDate               string            `json:"flight_date,omitempty"`
	AccountingInformation    string            `json:"accounting_information,omitempty"`
	ReferenceNumber          string            `json:"reference_number,omitempty"`
	Information1             string            `json:"information_1,omitempty"`
	Information2             string            `json:"information_2,omitempty"`
	Currency                 string            `json:"currency,omitempty"`
	ChargeCode               string            `json:"charge_code,omitempty"`
	WeightPayment            string            `json:"weight_payment,omitempty"`
	OtherPayment             string            `json:"other_payment,omitempty"`
	Sci                      string            `json:"sci,omitempty"`
	HandlingInformation      string            `json:"handling_information,omitempty"`
	Marks                    string            `json:"marks,omitempty"`
	GoodsDescription         string            `json:"goods_description,omitempty"`
	Clauses                  string            `json:"clauses,omitempty"`
	TransportTerms           string            `json:"transport_terms,omitempty"`
	OtherChargesText         string            `json:"other_charges_text,omitempty"`
	DestinationCurrency      string            `json:"destination_currency,omitempty"`
	CurrencyConversionRate   string            `json:"currency_conversion_rate,omitempty"`
	DestinationCollectCharge string            `json:"destination_collect_charge,omitempty"`
	ChargesAtDestination     string            `json:"charges_at_destination,omitempty"`
	TotalCollect             string            `json:"total_collect,omitempty"`
	ShipperCertification     string            `json:"shipper_certification,omitempty"`
	ShipperSignature         string            `json:"shipper_signature,omitempty"`
	CarrierExecution         string            `json:"carrier_execution,omitempty"`
	IssueDate                string            `json:"issue_date,omitempty"`
	IssuePlace               string            `json:"issue_place,omitempty"`
	CarrierSignature         string            `json:"carrier_signature,omitempty"`
	Remarks                  string            `json:"remarks,omitempty"`
	CarriageValue            *AirDeclaredValue `json:"carriage_value,omitempty"`
	CustomsValue             *AirDeclaredValue `json:"customs_value,omitempty"`
	InsuranceValue           *AirDeclaredValue `json:"insurance_value,omitempty"`
	RateLines                []*AirRateLine    `json:"rate_lines,omitempty"`
	OtherCharges             []*AirOtherCharge `json:"other_charges,omitempty"`
	Prepaid                  *AirChargeSummary `json:"prepaid,omitempty"`
	Collect                  *AirChargeSummary `json:"collect,omitempty"`
	AsArranged               bool              `json:"as_arranged,omitempty"`
}
