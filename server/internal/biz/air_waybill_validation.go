package biz

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

var airDecimalPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$`)
var airCurrencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

func airDecimal(value string) (decimal.Decimal, error) {
	if !airDecimalPattern.MatchString(value) {
		return decimal.Zero, ErrAirWaybillInvalid
	}
	v, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.Zero, ErrAirWaybillInvalid
	}
	return v, nil
}

// NormalizeAirWaybillContent 不修剪正文；换行及显式空串必须原样保存。
func NormalizeAirWaybillContent(input *AirWaybillContent) (*AirWaybillContent, error) {
	if input == nil {
		return nil, nil
	}
	b, err := json.Marshal(input)
	if err != nil {
		return nil, ErrAirWaybillInvalid
	}
	v := new(AirWaybillContent)
	if json.Unmarshal(b, v) != nil {
		return nil, ErrAirWaybillInvalid
	}
	rv := reflect.ValueOf(v).Elem()
	rt := rv.Type()
	for i := 0; i < rv.NumField(); i++ {
		if rv.Field(i).Kind() != reflect.String {
			continue
		}
		limit := 200
		name := rt.Field(i).Name
		switch name {
		case "ShipperText", "ConsigneeText", "NotifyPartyText", "SecondNotifyPartyText", "AgentText", "IssuedBy", "AccountingInformation", "HandlingInformation", "Clauses", "Remarks", "OtherChargesText", "ShipperCertification", "CarrierExecution":
			limit = 5000
		case "Marks", "GoodsDescription":
			limit = 10000
		}
		if utf8.RuneCountInString(rv.Field(i).String()) > limit {
			return nil, ErrAirWaybillInvalid
		}
	}
	for _, d := range []struct {
		v     *AirDeclaredValue
		token string
	}{{v.CarriageValue, "NVD"}, {v.CustomsValue, "NCV"}, {v.InsuranceValue, "XXX"}} {
		if d.v == nil {
			continue
		}
		if d.v.Kind == "AMOUNT" {
			if _, e := airDecimal(d.v.Amount); e != nil {
				return nil, e
			}
		} else if d.v.Kind != d.token || d.v.Amount != "" {
			return nil, ErrAirWaybillInvalid
		}
	}
	for _, s := range []string{v.Currency, v.DestinationCurrency} {
		if s != "" && !airCurrencyPattern.MatchString(s) {
			return nil, ErrAirWaybillInvalid
		}
	}
	for _, s := range []string{v.WeightPayment, v.OtherPayment} {
		if s != "" && s != "PPD" && s != "COLL" {
			return nil, ErrAirWaybillInvalid
		}
	}
	for _, s := range []string{v.IssueDate, v.FlightDate} {
		if s != "" {
			d, e := time.Parse("2006-01-02", s)
			if e != nil || d.Format("2006-01-02") != s {
				return nil, ErrAirWaybillInvalid
			}
		}
	}
	for _, s := range []string{v.CurrencyConversionRate, v.DestinationCollectCharge, v.ChargesAtDestination, v.TotalCollect} {
		if s != "" {
			if _, e := airDecimal(s); e != nil {
				return nil, e
			}
		}
	}
	if len(v.RateLines) > 200 || len(v.OtherCharges) > 200 {
		return nil, ErrAirWaybillInvalid
	}
	weight := decimal.Zero
	unknownWeight := v.AsArranged
	for _, l := range v.RateLines {
		if l == nil {
			return nil, ErrAirWaybillInvalid
		}
		if utf8.RuneCountInString(l.GoodsDescription) > 10000 || utf8.RuneCountInString(l.RateClass) > 20 || utf8.RuneCountInString(l.CommodityItemNo) > 100 {
			return nil, ErrAirWaybillInvalid
		}
		for _, s := range []string{l.Pieces, l.GrossWeightKg, l.ChargeableWeightKg, l.Rate, l.Amount} {
			if s != "" {
				if _, e := airDecimal(s); e != nil {
					return nil, e
				}
			}
		}
		if l.Pieces != "" && strings.Contains(l.Pieces, ".") {
			return nil, ErrAirWaybillInvalid
		}
		if l.Amount == "" {
			unknownWeight = true
		} else {
			d, _ := airDecimal(l.Amount)
			weight = weight.Add(d)
		}
	}
	// 有运价/费用时币种与付款栏必须明确，禁止猜测报价规则。
	if (len(v.RateLines) > 0 || v.AsArranged) && (v.Currency == "" || v.WeightPayment == "") {
		return nil, ErrAirWaybillInvalid
	}
	other := map[string]decimal.Decimal{"PPD_AGENT": decimal.Zero, "PPD_CARRIER": decimal.Zero, "COLL_AGENT": decimal.Zero, "COLL_CARRIER": decimal.Zero}
	for _, l := range v.OtherCharges {
		if l == nil || utf8.RuneCountInString(l.Name) > 200 || utf8.RuneCountInString(l.BillingUnit) > 100 {
			return nil, ErrAirWaybillInvalid
		}
		if l.Currency == "" || l.Currency != v.Currency || (l.Payment != "PPD" && l.Payment != "COLL") || (l.DueTo != "AGENT" && l.DueTo != "CARRIER") {
			return nil, ErrAirWaybillInvalid
		}
		for _, s := range []string{l.Quantity, l.UnitPrice} {
			if s != "" {
				if _, e := airDecimal(s); e != nil {
					return nil, e
				}
			}
		}
		amount, e := airDecimal(l.Amount)
		if e != nil {
			return nil, e
		}
		k := l.Payment + "_" + l.DueTo
		other[k] = other[k].Add(amount)
	}
	hasCharges := len(v.RateLines) > 0 || len(v.OtherCharges) > 0 || v.Prepaid != nil || v.Collect != nil || v.AsArranged
	if hasCharges {
		if v.Currency == "" {
			return nil, ErrAirWaybillInvalid
		}
		if v.Prepaid == nil {
			v.Prepaid = &AirChargeSummary{}
		}
		if v.Collect == nil {
			v.Collect = &AirChargeSummary{}
		}
		for _, x := range []struct {
			s       *AirChargeSummary
			payment string
		}{{v.Prepaid, "PPD"}, {v.Collect, "COLL"}} {
			total := decimal.Zero
			for _, s := range []string{x.s.ValuationCharge, x.s.Tax} {
				if s != "" {
					d, e := airDecimal(s)
					if e != nil {
						return nil, e
					}
					total = total.Add(d)
				}
			}
			x.s.WeightCharge = "0"
			if v.WeightPayment == x.payment {
				if unknownWeight {
					x.s.WeightCharge = ""
				} else {
					x.s.WeightCharge = weight.String()
					total = total.Add(weight)
				}
			}
			a, c := other[x.payment+"_AGENT"], other[x.payment+"_CARRIER"]
			x.s.OtherDueAgent = a.String()
			x.s.OtherDueCarrier = c.String()
			x.s.Total = total.Add(a).Add(c).String()
			if unknownWeight && v.WeightPayment == x.payment {
				x.s.Total = ""
			}
		}
	}
	return v, nil
}

func AirWaybillContentHash(v *AirWaybillContent) string {
	if v == nil {
		v = &AirWaybillContent{}
	}
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
