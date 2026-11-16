package service

import (
	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func airDeclaredValueFromAPI(in *v1.AirDeclaredValue) *biz.AirDeclaredValue {
	if in == nil {
		return nil
	}
	out := &biz.AirDeclaredValue{}
	out.Kind = in.Kind
	out.Amount = in.Amount
	return out
}
func airDeclaredValueToAPI(in *biz.AirDeclaredValue) *v1.AirDeclaredValue {
	if in == nil {
		return nil
	}
	out := &v1.AirDeclaredValue{}
	out.Kind = in.Kind
	out.Amount = in.Amount
	return out
}
func airRateLineFromAPI(in *v1.AirRateLine) *biz.AirRateLine {
	if in == nil {
		return nil
	}
	out := &biz.AirRateLine{}
	out.Pieces = in.Pieces
	out.GrossWeightKg = in.GrossWeightKg
	out.RateClass = in.RateClass
	out.CommodityItemNo = in.CommodityItemNo
	out.ChargeableWeightKg = in.ChargeableWeightKg
	out.Rate = in.Rate
	out.Amount = in.Amount
	out.GoodsDescription = in.GoodsDescription
	return out
}
func airRateLineToAPI(in *biz.AirRateLine) *v1.AirRateLine {
	if in == nil {
		return nil
	}
	out := &v1.AirRateLine{}
	out.Pieces = in.Pieces
	out.GrossWeightKg = in.GrossWeightKg
	out.RateClass = in.RateClass
	out.CommodityItemNo = in.CommodityItemNo
	out.ChargeableWeightKg = in.ChargeableWeightKg
	out.Rate = in.Rate
	out.Amount = in.Amount
	out.GoodsDescription = in.GoodsDescription
	return out
}
func airOtherChargeFromAPI(in *v1.AirOtherCharge) *biz.AirOtherCharge {
	if in == nil {
		return nil
	}
	out := &biz.AirOtherCharge{}
	out.Name = in.Name
	out.BillingUnit = in.BillingUnit
	out.Quantity = in.Quantity
	out.Currency = in.Currency
	out.UnitPrice = in.UnitPrice
	out.Amount = in.Amount
	out.DueTo = in.DueTo
	out.Payment = in.Payment
	return out
}
func airOtherChargeToAPI(in *biz.AirOtherCharge) *v1.AirOtherCharge {
	if in == nil {
		return nil
	}
	out := &v1.AirOtherCharge{}
	out.Name = in.Name
	out.BillingUnit = in.BillingUnit
	out.Quantity = in.Quantity
	out.Currency = in.Currency
	out.UnitPrice = in.UnitPrice
	out.Amount = in.Amount
	out.DueTo = in.DueTo
	out.Payment = in.Payment
	return out
}
func airChargeSummaryFromAPI(in *v1.AirChargeSummary) *biz.AirChargeSummary {
	if in == nil {
		return nil
	}
	out := &biz.AirChargeSummary{}
	out.WeightCharge = in.WeightCharge
	out.ValuationCharge = in.ValuationCharge
	out.Tax = in.Tax
	out.OtherDueAgent = in.OtherDueAgent
	out.OtherDueCarrier = in.OtherDueCarrier
	out.Total = in.Total
	return out
}
func airChargeSummaryToAPI(in *biz.AirChargeSummary) *v1.AirChargeSummary {
	if in == nil {
		return nil
	}
	out := &v1.AirChargeSummary{}
	out.WeightCharge = in.WeightCharge
	out.ValuationCharge = in.ValuationCharge
	out.Tax = in.Tax
	out.OtherDueAgent = in.OtherDueAgent
	out.OtherDueCarrier = in.OtherDueCarrier
	out.Total = in.Total
	return out
}
func airContentFromAPI(in *v1.AirWaybillContent) *biz.AirWaybillContent {
	if in == nil {
		return nil
	}
	out := &biz.AirWaybillContent{}
	out.ShipperAccount = in.ShipperAccount
	out.ShipperText = in.ShipperText
	out.ConsigneeAccount = in.ConsigneeAccount
	out.ConsigneeText = in.ConsigneeText
	out.NotifyPartyText = in.NotifyPartyText
	out.SecondNotifyPartyText = in.SecondNotifyPartyText
	out.AgentText = in.AgentText
	out.AgentIataCode = in.AgentIataCode
	out.AgentAccount = in.AgentAccount
	out.IssuedBy = in.IssuedBy
	out.DepartureText = in.DepartureText
	out.RouteTo = in.RouteTo
	out.FirstCarrier = in.FirstCarrier
	out.RouteTo2 = in.RouteTo_2
	out.Carrier2 = in.Carrier_2
	out.RouteTo3 = in.RouteTo_3
	out.Carrier3 = in.Carrier_3
	out.DestinationText = in.DestinationText
	out.FlightText = in.FlightText
	out.FlightDate = in.FlightDate
	out.AccountingInformation = in.AccountingInformation
	out.ReferenceNumber = in.ReferenceNumber
	out.Information1 = in.Information_1
	out.Information2 = in.Information_2
	out.Currency = in.Currency
	out.ChargeCode = in.ChargeCode
	out.WeightPayment = in.WeightPayment
	out.OtherPayment = in.OtherPayment
	out.Sci = in.Sci
	out.HandlingInformation = in.HandlingInformation
	out.Marks = in.Marks
	out.GoodsDescription = in.GoodsDescription
	out.Clauses = in.Clauses
	out.TransportTerms = in.TransportTerms
	out.OtherChargesText = in.OtherChargesText
	out.DestinationCurrency = in.DestinationCurrency
	out.CurrencyConversionRate = in.CurrencyConversionRate
	out.DestinationCollectCharge = in.DestinationCollectCharge
	out.ChargesAtDestination = in.ChargesAtDestination
	out.TotalCollect = in.TotalCollect
	out.ShipperCertification = in.ShipperCertification
	out.ShipperSignature = in.ShipperSignature
	out.CarrierExecution = in.CarrierExecution
	out.IssueDate = in.IssueDate
	out.IssuePlace = in.IssuePlace
	out.CarrierSignature = in.CarrierSignature
	out.Remarks = in.Remarks
	out.CarriageValue = airDeclaredValueFromAPI(in.CarriageValue)
	out.CustomsValue = airDeclaredValueFromAPI(in.CustomsValue)
	out.InsuranceValue = airDeclaredValueFromAPI(in.InsuranceValue)
	for _, v := range in.RateLines {
		out.RateLines = append(out.RateLines, airRateLineFromAPI(v))
	}
	for _, v := range in.OtherCharges {
		out.OtherCharges = append(out.OtherCharges, airOtherChargeFromAPI(v))
	}
	out.Prepaid = airChargeSummaryFromAPI(in.Prepaid)
	out.Collect = airChargeSummaryFromAPI(in.Collect)
	out.AsArranged = in.AsArranged
	return out
}
func airContentToAPI(in *biz.AirWaybillContent) *v1.AirWaybillContent {
	if in == nil {
		return nil
	}
	out := &v1.AirWaybillContent{}
	out.ShipperAccount = in.ShipperAccount
	out.ShipperText = in.ShipperText
	out.ConsigneeAccount = in.ConsigneeAccount
	out.ConsigneeText = in.ConsigneeText
	out.NotifyPartyText = in.NotifyPartyText
	out.SecondNotifyPartyText = in.SecondNotifyPartyText
	out.AgentText = in.AgentText
	out.AgentIataCode = in.AgentIataCode
	out.AgentAccount = in.AgentAccount
	out.IssuedBy = in.IssuedBy
	out.DepartureText = in.DepartureText
	out.RouteTo = in.RouteTo
	out.FirstCarrier = in.FirstCarrier
	out.RouteTo_2 = in.RouteTo2
	out.Carrier_2 = in.Carrier2
	out.RouteTo_3 = in.RouteTo3
	out.Carrier_3 = in.Carrier3
	out.DestinationText = in.DestinationText
	out.FlightText = in.FlightText
	out.FlightDate = in.FlightDate
	out.AccountingInformation = in.AccountingInformation
	out.ReferenceNumber = in.ReferenceNumber
	out.Information_1 = in.Information1
	out.Information_2 = in.Information2
	out.Currency = in.Currency
	out.ChargeCode = in.ChargeCode
	out.WeightPayment = in.WeightPayment
	out.OtherPayment = in.OtherPayment
	out.Sci = in.Sci
	out.HandlingInformation = in.HandlingInformation
	out.Marks = in.Marks
	out.GoodsDescription = in.GoodsDescription
	out.Clauses = in.Clauses
	out.TransportTerms = in.TransportTerms
	out.OtherChargesText = in.OtherChargesText
	out.DestinationCurrency = in.DestinationCurrency
	out.CurrencyConversionRate = in.CurrencyConversionRate
	out.DestinationCollectCharge = in.DestinationCollectCharge
	out.ChargesAtDestination = in.ChargesAtDestination
	out.TotalCollect = in.TotalCollect
	out.ShipperCertification = in.ShipperCertification
	out.ShipperSignature = in.ShipperSignature
	out.CarrierExecution = in.CarrierExecution
	out.IssueDate = in.IssueDate
	out.IssuePlace = in.IssuePlace
	out.CarrierSignature = in.CarrierSignature
	out.Remarks = in.Remarks
	out.CarriageValue = airDeclaredValueToAPI(in.CarriageValue)
	out.CustomsValue = airDeclaredValueToAPI(in.CustomsValue)
	out.InsuranceValue = airDeclaredValueToAPI(in.InsuranceValue)
	for _, v := range in.RateLines {
		out.RateLines = append(out.RateLines, airRateLineToAPI(v))
	}
	for _, v := range in.OtherCharges {
		out.OtherCharges = append(out.OtherCharges, airOtherChargeToAPI(v))
	}
	out.Prepaid = airChargeSummaryToAPI(in.Prepaid)
	out.Collect = airChargeSummaryToAPI(in.Collect)
	out.AsArranged = in.AsArranged
	return out
}
