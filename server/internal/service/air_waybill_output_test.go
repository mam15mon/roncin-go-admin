package service

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestAirWaybillOutputCompanyWire(t *testing.T) {
	master := &biz.AirMasterWaybill{ID: uuid.New(), OrganizationID: uuid.New(), BusinessType: biz.OrderBusinessAE, OrganizationNameEn: "ACTUAL COMPANY", Version: 1, Content: biz.AirMasterWaybillInput{WaybillContent: &biz.AirWaybillContent{IssuedBy: "SAVED ISSUER"}}}
	data, err := protojson.MarshalOptions{UseEnumNumbers: true}.Marshal(airMasterToAPI(master))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"organizationNameEn":"ACTUAL COMPANY"`) || !strings.Contains(string(data), `"issuedBy":"SAVED ISSUER"`) {
		t.Fatalf("真实公司与印发正文丢失：%s", data)
	}
	master.OrganizationNameEn = ""
	data, err = protojson.Marshal(airMasterToAPI(master))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "organizationNameEn") || !strings.Contains(string(data), "SAVED ISSUER") {
		t.Fatal("非optional空名应省略，不伪造默认主体")
	}
}
