package service

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestPartnerPaymentTermsOptionalWireRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name, wire string
		days       *int
	}{{"未约定", `{}`, nil}, {"现结", `{"paymentTermsDays":0}`, new(0)}, {"账期", `{"paymentTermsDays":30}`, new(30)}} {
		t.Run(tc.name, func(t *testing.T) {
			var input v1.PartnerSettlementRuleInput
			if err := protojson.Unmarshal([]byte(tc.wire), &input); err != nil {
				t.Fatal(err)
			}
			domain := partnerSettlementRuleFromAPI(&input)
			if (domain.PaymentTermsDays == nil) != (tc.days == nil) || (tc.days != nil && *domain.PaymentTermsDays != *tc.days) {
				t.Fatalf("请求 optional 存在性丢失: %+v", domain)
			}
			domain.ID, domain.PartnerRoleID = uuid.New(), uuid.New()
			output := partnerSettlementRuleToAPI(domain)
			raw, err := protojson.Marshal(output)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			days, exists := fields["paymentTermsDays"]
			if exists != (tc.days != nil) {
				t.Fatalf("响应 optional 存在性错误: %s", raw)
			}
			if tc.days != nil {
				var got int
				if err := json.Unmarshal(days, &got); err != nil || got != *tc.days {
					t.Fatalf("响应账期错误: %s %v", raw, err)
				}
			}
		})
	}
	api := partnerSettlementRuleToAPI(&biz.PartnerSettlementRule{ID: uuid.New(), PartnerRoleID: uuid.New(), CreditLimitMinor: new(int64(0)), CreditCurrency: new("CNY")})
	raw, err := protojson.Marshal(api)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["creditLimitMinor"]; !exists {
		t.Fatalf("显式零额度不能省略: %s", raw)
	}
}
