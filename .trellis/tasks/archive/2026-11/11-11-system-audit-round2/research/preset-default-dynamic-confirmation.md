# B7 单证默认项动态确认

2026-11-11，基线 main `c2c254f9`，用户明确要求实际确认。结论：B7 / P2 成立，未修复。

## 验证方式与结果

独立 PostgreSQL 18 临时集群、随机 Schema、正式 SQL 迁移；直接调用真实 Service API DTO→Biz→Data，核对成功响应、真实存储及 enabled=true 列表。未经过 HTTP/鉴权中间件，不作为浏览器正常操作复现，也未断言订单自动带入受损。

使用合法 SHIPPER 资料，同客商、同预设类型，先创建启用默认 A。三个独立场景全部 PASS、无 SKIP，共 3.422s；这里的 PASS 表示复现当前异常成功，并非修复验收。

| 操作 | 响应 B(enabled/default) | 数据库 A(enabled/default) | 数据库 B(enabled/default) | 启用候选默认数 |
| --- | --- | --- | --- | --- |
| 新建 B，enabled=false,isDefault=true | false/true | true/false | false/true | 0 |
| 原 B 为启用非默认，更新同一组合 | false/false | true/false | false/false | 0 |
| 正常新建启用默认 B（对照） | true/true | true/false | true/true | 1 |

根因锚点：`server/internal/data/partner_shipping_preset.go:58` 在创建时先清旧默认，`:75` 原样写禁用项默认；更新在 `:98` 先清旧默认，`:113` 才将本项默认规范为 isDefault&&enabled。Biz normalize 未拒绝该组合，Service直接转换传入。

新建 UI 默认 enabled=true 且无启停开关，结论限定为 API 请求边界。首轮 MARKS 更新另遇到 content 重复赋值的 SQL 错误，事务回滚；不得泛化为所有预设类型的更新都成功。正式结论基于 SHIPPER 路径；独立文本错误待单独核查，不计 B7 的下游后果。

## 精确复现

将下方临时测试放到 `server/internal/data/preset_default_audit_repro_test.go`，向专用隔离数据库注入 RONCIN_INTEGRATION_DATABASE_SOURCE 后执行：

```bash
go -C server test ./internal/data -run '^TestPresetDefaultAuditReproductionPostgres$' -count=1 -v
```

本轮通过仓库外安全包装器注入连接变量，日志为 `/tmp/roncin-preset-default-repro.log`，源码为 `/tmp/roncin-preset-default-repro.go`；实验后已删除仓库临时文件。测试只断言当前异常事实，不长期加入正常门禁。后续修复须改为期望正确行为的针对性回归。

```go
package data

import (
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourcepartner"
	"github.com/roncin/roncin-go-admin/server/internal/service"
)

// 此临时实验断言当前异常行为，不是修复后的回归测试；实验完成后移出仓库。
func TestPresetDefaultAuditReproductionPostgres(t *testing.T) {
	for _, mode := range []string{"create_disabled_default", "update_disabled_default", "normal_enabled_default"} {
		t.Run(mode, func(t *testing.T) {
			d, cleanup := getIntegrationData(t)
			defer cleanup()
			f := newCreditPartnerFixture(t, d)
			partnerID, _ := f.newCreditPartner(mode)
			user, err := d.db.User.Create().SetDisplayName("预设审计验证人").Save(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			ctx := biz.WithPrincipal(t.Context(), &biz.Principal{UserID: user.ID, Organization: biz.Organization{ID: f.organizationID, Kind: biz.OrganizationKindCompany}})
			uc := biz.NewPartnerShippingPresetUsecase(NewPartnerShippingPresetRepo(d))
			svc := service.NewPartnerService(nil, nil, nil, nil, nil, uc, nil)
			makeInput := func(title string, enabled, isDefault bool) *v1.PartnerShippingPresetInput {
				return &v1.PartnerShippingPresetInput{PresetType: v1.PartnerShippingPresetType_PARTNER_SHIPPING_PRESET_TYPE_SHIPPER, Title: title, Enabled: enabled, IsDefault: isDefault, Payload: &v1.PartnerShippingPresetInput_Party{Party: &v1.PartnerShippingPartyPayload{CompanyName: "审计发货人", CountryCode: "CN"}}}
			}
			create := func(title string, enabled, isDefault bool) *v1.PartnerShippingPreset {
				t.Helper()
				r, e := svc.CreatePartnerShippingPreset(ctx, &v1.CreatePartnerShippingPresetRequest{PartnerId: partnerID.String(), Preset: makeInput(title, enabled, isDefault)})
				if e != nil {
					t.Fatal(e)
				}
				if !r.Success || r.Data == nil {
					t.Fatalf("创建响应不是成功: %+v", r)
				}
				return r.Data
			}
			a := create("原启用默认A", true, true)
			var b *v1.PartnerShippingPreset
			switch mode {
			case "create_disabled_default":
				b = create("新禁用默认B", false, true)
			case "update_disabled_default":
				b = create("原启用非默认B", true, false)
				r, e := svc.UpdatePartnerShippingPreset(ctx, &v1.UpdatePartnerShippingPresetRequest{PartnerId: partnerID.String(), Id: b.Id, Preset: makeInput("更新禁用默认B", false, true)})
				if e != nil {
					t.Fatal(e)
				}
				if !r.Success || r.Data == nil {
					t.Fatalf("更新响应不是成功: %+v", r)
				}
				b = r.Data
			case "normal_enabled_default":
				b = create("新启用默认B", true, true)
			}
			aid, bid := uuid.MustParse(a.Id), uuid.MustParse(b.Id)
			resourceA, e := d.db.EnterpriseResource.Get(ctx, aid)
			if e != nil {
				t.Fatal(e)
			}
			resourceB, e := d.db.EnterpriseResource.Get(ctx, bid)
			if e != nil {
				t.Fatal(e)
			}
			linkA, e := d.db.EnterpriseResourcePartner.Query().Where(linkent.ResourceIDEQ(aid), linkent.PartnerIDEQ(partnerID)).Only(ctx)
			if e != nil {
				t.Fatal(e)
			}
			linkB, e := d.db.EnterpriseResourcePartner.Query().Where(linkent.ResourceIDEQ(bid), linkent.PartnerIDEQ(partnerID)).Only(ctx)
			if e != nil {
				t.Fatal(e)
			}
			enabled := true
			list, e := svc.ListPartnerShippingPresets(ctx, &v1.ListPartnerShippingPresetsRequest{PartnerId: partnerID.String(), Enabled: &enabled})
			if e != nil {
				t.Fatal(e)
			}
			defaults := 0
			for _, x := range list.Data {
				if x.IsDefault {
					defaults++
				}
			}
			t.Logf("成功响应B(enabled=%v,isDefault=%v)；数据库A(enabled=%v,isDefault=%v)，B(enabled=%v,isDefault=%v)；enabled=true候选=%d，默认数=%d", b.Enabled, b.IsDefault, resourceA.Enabled, linkA.IsDefault, resourceB.Enabled, linkB.IsDefault, len(list.Data), defaults)
			if !resourceA.Enabled || linkA.IsDefault {
				t.Fatal("实验前提不符：A应仍启用且被清默认")
			}
			wantEnabled := mode == "normal_enabled_default"
			wantDefault := mode != "update_disabled_default"
			if b.Enabled != wantEnabled || resourceB.Enabled != wantEnabled || b.IsDefault != wantDefault || linkB.IsDefault != wantDefault {
				t.Fatal("返回或存储与预期复现状态不符")
			}
			wantCandidates, wantDefaults := 1, 0
			if mode == "normal_enabled_default" {
				wantCandidates, wantDefaults = 2, 1
			}
			if len(list.Data) != wantCandidates || defaults != wantDefaults {
				t.Fatalf("启用候选不符：%d/%d", len(list.Data), defaults)
			}
		})
	}
}
```
