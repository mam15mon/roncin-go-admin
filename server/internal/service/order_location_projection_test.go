package service

import (
	"encoding/json"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestOrderLocationEnglishNamesWire(t *testing.T) {
	for _, tc := range []struct {
		name  string
		order *biz.Order
		want  map[string]string
	}{
		{"英文名称独立投影", &biz.Order{
			OriginLocationName: "上海港 (CNSHA)", OriginLocationNameEn: "Shanghai",
			DestinationLocationName: "洛杉矶港 (USLAX)", DestinationLocationNameEn: "Los Angeles",
			DischargeLocationName: "新加坡港 (SGSIN)", DischargeLocationNameEn: "Singapore",
		}, map[string]string{"originLocationNameEn": "Shanghai", "destinationLocationNameEn": "Los Angeles", "dischargeLocationNameEn": "Singapore"}},
		{"未设置时不回退中文", &biz.Order{OriginLocationName: "上海港 (CNSHA)"}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := protojson.Marshal(orderToAPI(tc.order))
			if err != nil {
				t.Fatal(err)
			}
			var wire map[string]any
			if err := json.Unmarshal(encoded, &wire); err != nil {
				t.Fatal(err)
			}
			for _, field := range []string{"originLocationNameEn", "destinationLocationNameEn", "dischargeLocationNameEn"} {
				want, present := tc.want[field]
				got, exists := wire[field]
				if exists != present || (exists && got != want) {
					t.Fatalf("%s = %#v（存在=%v），期望 %q（存在=%v）", field, got, exists, want, present)
				}
			}
		})
	}
}
