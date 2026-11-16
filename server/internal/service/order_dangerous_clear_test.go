package service

import (
	"testing"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestAirDangerousOptionalWireClearAndOmission(t *testing.T) {
	for _, business := range []biz.OrderBusinessType{biz.OrderBusinessAE, biz.OrderBusinessAI} {
		pg := biz.OrderDangerousPackingII
		old := &biz.Order{BusinessType: business, UNNumber: "1234", HazardClass: "3", DeclarationCutoffAt: "2026-11-16T00:00:00Z", DangerousPackingGroup: &pg}
		for _, tc := range []struct {
			wire  string
			clear bool
		}{{`{"notes":"仅改备注"}`, false}, {`{"unNumber":"","hazardClass":"","declarationCutoffAt":"","dangerousPackingGroup":0}`, true}} {
			req := &v1.UpdateOrderRequest{}
			if err := protojson.Unmarshal([]byte(tc.wire), req); err != nil {
				t.Fatal(err)
			}
			merged, err := mergeOrderUpdateRequest(old, req)
			if err != nil {
				t.Fatal(err)
			}
			if tc.clear {
				if merged.UNNumber != "" || merged.HazardClass != "" || merged.DeclarationCutoffAt != "" || merged.DangerousPackingGroup != nil {
					t.Fatal("显式空/UNSPECIFIED未清空")
				}
			} else if merged.UNNumber != old.UNNumber || merged.HazardClass != old.HazardClass || merged.DeclarationCutoffAt != old.DeclarationCutoffAt || merged.DangerousPackingGroup == nil || *merged.DangerousPackingGroup != pg {
				t.Fatal("optional省略错误清空原危险信息")
			}
		}
	}
}
