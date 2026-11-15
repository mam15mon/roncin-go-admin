package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// 验证可空字段赋值和清空各只生成一种更新意图，避免 PostgreSQL 重复列赋值。
func TestPartnerShippingPresetOptionalUpdatePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	f := newCreditPartnerFixture(t, d)
	partnerID, _ := f.newCreditPartner("预设可空字段更新")
	user, err := d.db.User.Create().SetDisplayName("预设更新回归人员").Save(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	uc := biz.NewPartnerShippingPresetUsecase(NewPartnerShippingPresetRepo(d))
	for _, kind := range []biz.PartnerShippingPresetType{biz.PartnerShippingPresetShipper, biz.PartnerShippingPresetMarks, biz.PartnerShippingPresetHSCode} {
		t.Run(string(kind), func(t *testing.T) {
			input := &biz.PartnerShippingPreset{PresetType: kind, Title: "原预设", Enabled: true, Remark: "原备注"}
			if kind.Party() {
				input.Party = &biz.PartnerShippingPartyPayload{CompanyName: "原发货人", CountryCode: "CN", Address: "原地址", ContactName: "原联系人", Phone: "123", Email: "old@example.com", TaxIdentifier: "OLD"}
			} else {
				input.Text = &biz.PartnerShippingTextPayload{Content: "原内容", Code: "123456"}
			}
			created, err := uc.Create(t.Context(), f.organizationID, user.ID, partnerID, input)
			if err != nil {
				t.Fatal(err)
			}
			input.Title, input.Remark = "新预设", "新备注"
			if kind.Party() {
				input.Party = &biz.PartnerShippingPartyPayload{CompanyName: "新发货人", CountryCode: "US", Address: "新地址", ContactName: "新联系人", Phone: "456", Email: "new@example.com", TaxIdentifier: "NEW"}
			} else {
				input.Text = &biz.PartnerShippingTextPayload{Content: "新内容", Code: "654321"}
			}
			updated, err := uc.Update(t.Context(), f.organizationID, user.ID, partnerID, created.ID, input)
			if err != nil {
				t.Fatalf("非空更新失败: %v", err)
			}
			if updated.ID == uuid.Nil || updated.Title != input.Title || updated.Remark != input.Remark {
				t.Fatalf("更新投影错误: %+v", updated)
			}
			if kind.Party() && *updated.Party != *input.Party {
				t.Fatalf("发货人持久化错误: %+v", updated.Party)
			}
			if !kind.Party() && *updated.Text != *input.Text {
				t.Fatalf("文本持久化错误: %+v", updated.Text)
			}
			input.Remark = ""
			if kind.Party() {
				input.Party = &biz.PartnerShippingPartyPayload{CompanyName: "新发货人", CountryCode: "US"}
			} else if kind == biz.PartnerShippingPresetHSCode {
				input.Text.Content = ""
			} else {
				input.Text.Code = ""
			}
			updated, err = uc.Update(t.Context(), f.organizationID, user.ID, partnerID, created.ID, input)
			if err != nil {
				t.Fatalf("清空可选字段失败: %v", err)
			}
			if updated.Remark != "" {
				t.Fatal("备注未清空")
			}
			if kind.Party() && *updated.Party != *input.Party {
				t.Fatalf("发货人可选字段未清空: %+v", updated.Party)
			}
			if !kind.Party() && *updated.Text != *input.Text {
				t.Fatalf("文本可选字段未清空: %+v", updated.Text)
			}
		})
	}
}
