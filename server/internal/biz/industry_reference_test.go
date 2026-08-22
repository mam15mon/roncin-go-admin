package biz

import (
	"reflect"
	"testing"
)

func TestNormalizePortUsesUNLocodeSemantics(t *testing.T) {
	input := &Port{UNLocode: " cnshg ", NameZH: " 上海港 ", NameEN: " Shanghai ", CountryCode: " cn ", TransportModes: []string{" sea ", "RAIL", "SEA"}, SortOrder: 100}
	got, err := normalizePort(input, true)
	if err != nil {
		t.Fatalf("normalizePort() error = %v", err)
	}
	if got.UNLocode != "CNSHG" || got.CountryCode != "CN" || got.NameZH != "上海港" || got.NameEN != "Shanghai" || got.Source != "manual" || !reflect.DeepEqual(got.TransportModes, []string{"SEA", "RAIL"}) {
		t.Fatalf("normalizePort() = %#v", got)
	}
	if input.TransportModes[0] != " sea " {
		t.Fatalf("normalizePort() 修改了输入切片: %#v", input.TransportModes)
	}
}

func TestNormalizePortRejectsCountryMismatch(t *testing.T) {
	_, err := normalizePort(&Port{UNLocode: "CNSHG", NameZH: "上海港", NameEN: "Shanghai", CountryCode: "US", TransportModes: []string{"SEA"}}, true)
	if err != ErrMasterDataInvalidArgument {
		t.Fatalf("normalizePort() error = %v, want %v", err, ErrMasterDataInvalidArgument)
	}
}

func TestNormalizeAirportCodesAndNames(t *testing.T) {
	icao := " zspd "
	cityEN := " Shanghai "
	got, err := normalizeAirport(&Airport{IATACode: " pvg ", ICAOCode: &icao, NameZH: " 浦东机场 ", NameEN: " Pudong International Airport ", CityNameZH: " 上海 ", CityNameEN: &cityEN, CountryCode: " cn ", SortOrder: 100}, true)
	if err != nil {
		t.Fatalf("normalizeAirport() error = %v", err)
	}
	if got.IATACode != "PVG" || got.ICAOCode == nil || *got.ICAOCode != "ZSPD" || got.CityNameEN == nil || *got.CityNameEN != "Shanghai" {
		t.Fatalf("normalizeAirport() = %#v", got)
	}
}

func TestNormalizeAirlineRejectsInvalidAWBPrefix(t *testing.T) {
	_, err := normalizeAirline(&Airline{IATACode: "CA", AWBPrefix: "99", NameZH: "中国国际航空", NameEN: "Air China", CountryCode: "CN"}, true)
	if err != ErrMasterDataInvalidArgument {
		t.Fatalf("normalizeAirline() error = %v, want %v", err, ErrMasterDataInvalidArgument)
	}
}

func TestNormalizeShippingLineSeparatesSCACAndContainerPrefixes(t *testing.T) {
	trackingURL := " https://www.maersk.com/tracking/ "
	input := &ShippingLine{SCACCode: " maeU ", NameZH: " 马士基 ", NameEN: " Maersk ", CountryCode: " dk ", TrackingURL: &trackingURL, ContainerPrefixes: []string{" msku ", "MRSU", "MSKU"}, SortOrder: 100}
	got, err := normalizeShippingLine(input, true)
	if err != nil {
		t.Fatalf("normalizeShippingLine() error = %v", err)
	}
	if got.SCACCode != "MAEU" || !reflect.DeepEqual(got.ContainerPrefixes, []string{"MSKU", "MRSU"}) || got.TrackingURL == nil || *got.TrackingURL != "https://www.maersk.com/tracking/" {
		t.Fatalf("normalizeShippingLine() = %#v", got)
	}
}

func TestNormalizeShippingLineRejectsUnsafeTrackingURL(t *testing.T) {
	trackingURL := "javascript:alert(1)"
	_, err := normalizeShippingLine(&ShippingLine{SCACCode: "MAEU", NameZH: "马士基", NameEN: "Maersk", CountryCode: "DK", TrackingURL: &trackingURL}, true)
	if err != ErrMasterDataInvalidArgument {
		t.Fatalf("normalizeShippingLine() error = %v, want %v", err, ErrMasterDataInvalidArgument)
	}
}
