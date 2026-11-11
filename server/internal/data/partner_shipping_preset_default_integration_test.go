package data

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	resourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourcepartner"
	"github.com/roncin/roncin-go-admin/server/internal/service"
)

func TestPartnerShippingPresetDefaultValidationPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	f := newCreditPartnerFixture(t, d)
	user, err := d.db.User.Create().SetDisplayName("默认预设回归人员").Save(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	ctx := biz.WithPrincipal(t.Context(), &biz.Principal{UserID: user.ID, Organization: biz.Organization{ID: f.organizationID, Kind: biz.OrganizationKindCompany}})
	svc := service.NewPartnerService(nil, nil, nil, nil, nil, biz.NewPartnerShippingPresetUsecase(NewPartnerShippingPresetRepo(d)), nil)
	input := func(title string, enabled, isDefault bool) *v1.PartnerShippingPresetInput {
		return &v1.PartnerShippingPresetInput{
			PresetType: v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_SHIPPER,
			Title:      title, Enabled: enabled, IsDefault: isDefault,
			Payload: &v1.PartnerShippingPresetInput_Party{Party: &v1.PartnerShippingPartyPayload{CompanyName: "默认预设回归发货人", CountryCode: "CN"}},
		}
	}
	create := func(t *testing.T, partnerID uuid.UUID, value *v1.PartnerShippingPresetInput) *v1.PartnerShippingPreset {
		t.Helper()
		response, err := svc.CreatePartnerShippingPreset(ctx, &v1.CreatePartnerShippingPresetRequest{PartnerId: partnerID.String(), Preset: value})
		if err != nil || response == nil || !response.Success || response.Data == nil {
			t.Fatalf("合法创建失败: %+v %v", response, err)
		}
		return response.Data
	}
	update := func(t *testing.T, partnerID uuid.UUID, id string, value *v1.PartnerShippingPresetInput) *v1.PartnerShippingPreset {
		t.Helper()
		response, err := svc.UpdatePartnerShippingPreset(ctx, &v1.UpdatePartnerShippingPresetRequest{PartnerId: partnerID.String(), Id: id, Preset: value})
		if err != nil || response == nil || !response.Success || response.Data == nil {
			t.Fatalf("合法更新失败: %+v %v", response, err)
		}
		return response.Data
	}
	// 快照包含资源与关联的所有持久字段，防止拒绝请求只保住默认却改动其他资料。
	snapshot := func(t *testing.T, ctx context.Context) string {
		t.Helper()
		resources, err := d.db.EnterpriseResource.Query().Order(resourceent.ByID()).All(ctx)
		if err != nil {
			t.Fatal(err)
		}
		links, err := d.db.EnterpriseResourcePartner.Query().Order(linkent.ByID()).All(ctx)
		if err != nil {
			t.Fatal(err)
		}
		audits, err := d.db.AuditLog.Query().Count(ctx)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal([]any{resources, links, audits})
		if err != nil {
			t.Fatal(err)
		}
		return string(encoded)
	}
	for _, mode := range []string{"拒绝创建禁用默认", "拒绝更新禁用默认", "正常创建启用默认", "正常更新启用默认", "禁用非默认仍合法"} {
		t.Run(mode, func(t *testing.T) {
			partnerID, _ := f.newCreditPartner(mode)
			a := create(t, partnerID, input("原启用默认A", true, true))
			var b *v1.PartnerShippingPreset
			if mode == "拒绝创建禁用默认" || mode == "拒绝更新禁用默认" {
				if mode == "拒绝更新禁用默认" {
					b = create(t, partnerID, input("原非默认B", true, false))
				}
				before := snapshot(t, ctx)
				var err error
				if b == nil {
					_, err = svc.CreatePartnerShippingPreset(ctx, &v1.CreatePartnerShippingPresetRequest{PartnerId: partnerID.String(), Preset: input("非法新默认B", false, true)})
				} else {
					_, err = svc.UpdatePartnerShippingPreset(ctx, &v1.UpdatePartnerShippingPresetRequest{PartnerId: partnerID.String(), Id: b.Id, Preset: input("非法改名B", false, true)})
				}
				if !errors.Is(err, biz.ErrPartnerShippingPresetInvalidArgument) || errors.Code(err) != 400 {
					t.Fatalf("矛盾组合应返回已有400参数错误: %v", err)
				}
				if after := snapshot(t, ctx); after != before {
					t.Fatal("拒绝请求不得改动资源、关联或成功审计")
				}
			} else if mode == "正常创建启用默认" {
				b = create(t, partnerID, input("新启用默认B", true, true))
			} else if mode == "正常更新启用默认" {
				b = create(t, partnerID, input("原非默认B", true, false))
				b = update(t, partnerID, b.Id, input("新启用默认B", true, true))
			} else {
				b = create(t, partnerID, input("禁用非默认B", false, false))
				b = update(t, partnerID, b.Id, input("修改禁用非默认B", false, false))
				if b.Enabled || b.IsDefault {
					t.Fatalf("禁用非默认合法组合应保留: %+v", b)
				}
			}
			wantDefaultID := a.Id
			if mode == "正常创建启用默认" || mode == "正常更新启用默认" {
				wantDefaultID = b.Id
			}
			enabled := true
			list, err := svc.ListPartnerShippingPresets(ctx, &v1.ListPartnerShippingPresetsRequest{PartnerId: partnerID.String(), Enabled: &enabled})
			if err != nil || list == nil || !list.Success {
				t.Fatalf("读取启用预设失败: %+v %v", list, err)
			}
			defaults := 0
			for _, value := range list.Data {
				if value.IsDefault {
					defaults++
					if value.Id != wantDefaultID {
						t.Fatalf("实际默认项错误: %s，期望%s", value.Id, wantDefaultID)
					}
				}
			}
			if defaults != 1 {
				t.Fatalf("应恰有一个有效默认项，实际%d", defaults)
			}
		})
	}
}
