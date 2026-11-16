package service

import (
	"testing"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestAirOrderOptionalHouseAndZeroWire(t *testing.T) {
	for _, wire := range []string{`{"documentMode":"DIRECT"}`, `{"documentMode":"HOUSE","house":{"houseNo":"H-1","content":{"customsValue":{"kind":"AMOUNT","amount":"0"},"rateLines":[{"pieces":"0","grossWeightKg":"0"}],"asArranged":false},"expectedVersion":"1"}}`, `{"documentMode":"HOUSE","house":{"houseNo":"H-1","content":{}}}`, `{"documentMode":"HOUSE","house":{"houseNo":"H-1"}}`} {
		var request v1.AirOrderInput
		if err := protojson.Unmarshal([]byte(wire), &request); err != nil {
			t.Fatal(err)
		}
		input, err := airOrderInputFromAPI(&request)
		if err != nil {
			t.Fatal(err)
		}
		if request.House == nil {
			if input.House != nil {
				t.Fatal("DIRECT被生成分单")
			}
			continue
		}
		if request.House.Content == nil {
			if input.House.Content != nil {
				t.Fatal("未提交正文被变成清空")
			}
			continue
		}
		if input.House.Content == nil {
			t.Fatal("显式空正文丢失")
		}
		response := airOrderToAPI(&biz.AirOrderInfo{DocumentMode: input.DocumentMode, House: &biz.AirHouseWaybill{HouseNo: input.House.HouseNo, Content: input.House.Content}})
		encoded, err := protojson.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		var roundtrip v1.AirOrderInfo
		if err = protojson.Unmarshal(encoded, &roundtrip); err != nil {
			t.Fatal(err)
		}
		if roundtrip.House == nil || roundtrip.House.Content == nil {
			t.Fatal("响应wire正文存在性丢失")
		}
		if request.House.Content.CustomsValue != nil && roundtrip.House.Content.CustomsValue.Amount != "0" {
			t.Fatal("显式零声明丢失")
		}
	}
}

func TestAirExistingMasterIDOnlyDTO(t *testing.T) {
	id := "9c819925-b196-472c-9b18-994a3207eddf"
	in, e := airOrderInputFromAPI(&v1.AirOrderInput{MasterWaybillId: &id, ExpectedMasterVersion: 7, DocumentMode: "HOUSE", House: &v1.AirHouseWaybillInput{HouseNo: "H-ID"}})
	if e != nil || in.Master != nil || in.MasterWaybillID == nil || in.ExpectedMasterVersion != 7 {
		t.Fatalf("已有主单不应强制客户端运输副本：%+v %v", in, e)
	}
}
