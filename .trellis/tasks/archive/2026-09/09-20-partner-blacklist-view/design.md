# 技术设计：往来单位列表黑名单视图切换

## 0. 总览

一次跨层契约变更，完整照抄既有 `is_casual` optional bool 过滤范本：

```
proto (ListPartnersRequest + optional bool blacklisted)
  → make api（buf 重新生成 pb.go / http.pb.go / openapi.yaml）
  → pnpm run generate:web-client（重新生成前端 partnerService / typings）
  → service（DTO → PartnerListOptions）
  → biz（PartnerListOptions.Blacklisted *bool，透传不校验）
  → data（Ent HasRolesWith 谓词过滤）
前端：partners/index.tsx 加 Segmented 视图切换 + 拉黑信息列 + 导出透传
```

**无需数据库迁移**：`partnerroleent.BlacklistedEQ` 谓词已存在
（`internal/data/ent/partnerrole/where.go:229`），复合索引
`(partner_id, blacklisted)` 已存在（`ent/schema/partner_role.go:47`）。
**无需权限变更**：不新增权限码，不触发 `generate:permission-keys`。

## 1. 契约层（proto + 生成物）

`server/api/partner/v1/partner.proto` `ListPartnersRequest`（356-363 行）
追加：

```proto
message ListPartnersRequest {
  int32 page = 1;
  int32 page_size = 2;
  string keyword = 3;
  PartnerRoleType role = 4;
  optional bool enabled = 5;
  optional bool is_casual = 6;
  // 按业务角色的黑名单状态过滤；缺省不过滤。true 匹配存在已拉黑角色
  //（不要求角色启用）的档案，false 匹配存在未拉黑角色的档案。
  optional bool blacklisted = 7;
}
```

生成命令与产物（与源同组提交，不手改）：

```bash
make -C server api                      # partner.pb.go / partner_http.pb.go / openapi.yaml
pnpm run generate:web-client            # web/src/services/roncin/partnerService.ts + typings.d.ts
```

openapi.yaml 侧自动产出 `blacklisted` boolean query 参数（isCasual 参照
`server/openapi.yaml:7205-7210`）；kratos http 绑定自动完成，`_http.pb.go`
的手写改动量为零。

## 2. service 层

`server/internal/service/partner_profile.go` ListPartners（38-51 行）在
`IsCasual` 拷贝块后追加同构三行：

```go
if request.Blacklisted != nil {
    blacklisted := request.GetBlacklisted()
    options.Blacklisted = &blacklisted
}
```

`ExportPartners`（196-231 行）：同任务内同步透传 `Blacklisted`，使前端
黑名单视图的导出与列表一致（前端导出走的就是 ListPartners，见 §5）。

## 3. biz 层

`server/internal/biz/partner.go`：

- `PartnerListOptions`（208-215 行）追加 `Blacklisted *bool`。
- `PartnerUsecase.List`（306-316 行）**不加校验**，与 `Enabled/IsCasual`
  一致透传（bool 过滤无非法值）。

## 4. data 层（谓词语义——本设计核心决策）

`server/internal/data/partner.go` `partnerRepo.List`（55-93 行）现有：

```go
if options.Role != "" {
    query.Where(partnerent.HasRolesWith(
        partnerroleent.RoleTypeEQ(partnerroleent.RoleType(options.Role)),
        partnerroleent.EnabledEQ(true),
    ))
}
```

改为按「角色 + 黑名单」组合语义分支（**关键**：`blacklisted=true` 时
去掉 `EnabledEQ(true)` 约束——`SetPartnerRoleBlacklist`
（`partner.go:325-357`）只写 blacklisted 族字段、不动 `enabled`，拉黑与
停用正交；已停用的黑名单必须仍可见，否则黑名单视图会漏档）：

```go
rolePredicates := []predicate.PartnerRole{
    partnerroleent.RoleTypeEQ(partnerroleent.RoleType(options.Role)),
}
if options.Blacklisted != nil && *options.Blacklisted {
    // 黑名单视图：只认拉黑标记，不要求角色启用（拉黑与停用正交）。
    rolePredicates = append(rolePredicates, partnerroleent.BlacklistedEQ(true))
} else {
    rolePredicates = append(rolePredicates, partnerroleent.EnabledEQ(true))
    if options.Blacklisted != nil {
        rolePredicates = append(rolePredicates, partnerroleent.BlacklistedEQ(false))
    }
}
if options.Role != "" {
    query.Where(partnerent.HasRolesWith(rolePredicates...))
}
if options.Blacklisted != nil && options.Role == "" {
    // 无角色类型时的兜底语义：任一角色命中黑名单状态即返回。
    query.Where(partnerent.HasRolesWith(partnerroleent.BlacklistedEQ(*options.Blacklisted)))
}
```

> 备注：前端始终携带 `role`，无角色分支仅为契约完整性（openapi 消费方
> 可能不带 role 调用）。`blacklisted=false && role!=""` 语义是「存在该
> 类型、启用且未拉黑的角色」——本期前端不暴露该入口，契约先定语义。

## 5. 前端层

`web/src/pages/partners/index.tsx`：

1. **视图状态**：`const [blacklistView, setBlacklistView] = useState(false);`
   不入 URL（最小闭环；页签切换 pathname 变化自然重置）。
2. **Segmented**：放进 ProTable `headerTitle`（现 528-533 行的 Space 内，
   标题文字右侧），选项 `全部${currentView.title}` / `黑名单${currentView.title}`：
   ```tsx
   <Segmented
     value={blacklistView ? 'blacklist' : 'all'}
     options={[
       { label: `全部${currentView.title}`, value: 'all' },
       { label: `黑名单${currentView.title}`, value: 'blacklist' },
     ]}
     onChange={(v) => {
       setBlacklistView(v === 'blacklist');
       actionRef.current?.reset();   // 分页回第 1 页；外部 searchParams 不受 reset 影响，搜索条件保留
     }}
   />
   ```
   交互验证点：`actionRef.current?.reset()` 只重置 ProTable 内部分页
   （`search={false}`，表单不参与），页面级 `searchParams` 是外部 state
   不会被清除——正是「换视图、留搜索」的语义。若实现时发现 reset 语义
   与预期不符（例如清了不该清的），降级方案是 `reload()` 并接受页码
   越界回退到空页的边缘情况，二选一在实现时以实测为准。
3. **请求组装**（542-552 行）追加 `blacklisted: blacklistView || undefined`。
4. **拉黑信息列**：`columns` 追加一列，`hideInTable: !blacklistView`，
   渲染 `partner.roles` 中 `type === currentView.roleType` 的角色的
   `blacklistReason / blacklistedBy / blacklistedAt`（时间用现有格式化
   工具）；三者皆空显示 `-`。
5. **导出**（`handleExport` 172-250 行）：`partnerServiceListPartners`
   调用追加 `blacklisted: blacklistView || undefined`，文件名在黑名单
   视图下带「黑名单」前缀。既有的 `pageSize: 2000` 问题不在本任务修。

## 6. 测试设计

**后端**（`server/internal/biz/partner_test.go`，仿 532-575 行
`TestPartnerListFiltersByIsCasual`）：

- 新增 `TestPartnerListFiltersByBlacklisted`：true / false / nil 三态断言
  `repo.listOptions.Blacklisted` 透传。
- data 层谓词逻辑以代码评审 + 现有集成测试模式为准（repo 无 List 集成
  测试基建，不为本任务新建）。

**前端**（`web/src/pages/partners/index.test.tsx`，沿用现有 mock 结构：
`@/router/history` / `@/app/access` / `react-router.useLocation` /
`@/services/roncin/partnerService`）：

- 新增用例：点击 Segmented「黑名单客户」→ 断言
  `partnerServiceListPartners` 最新调用参数含 `blacklisted: true` 且
  `role` 保持当前视图类型；切回「全部客户」→ 参数中 `blacklisted`
  回到 `undefined`。

## 7. 风险与回滚

- 风险集中在 §4 谓词语义（分支组合）与 §5 reset 交互语义，两者都有
  降级方案，实现时实测确认。
- 回滚单元：整组变更一次提交，revert 即回滚；无数据迁移、无权限变更、
  生成物残留风险（生成物随提交一起回滚）。

## 8. 追加设计：导出正统化（阶段 D）

**问题**：前端 `handleExport` 调 `partnerServiceListPartners` 传
`pageSize: 2000`，被后端 `ValidListPagination` 上限 200 拒绝，导出实际
不可用；且前端循环翻页拼接全量为 AGENTS.md 明令禁止。

**现状**：`ExportPartners` RPC（权限 `business.partner.export`）已存在，
service 层服务端聚合翻页（`options.Page++` 直至 `len(items) >= Total`，
`PageSize: biz.MaxListPageSize`）——这正是合规聚合点，但前端未使用，且
`PartnerExportItem` 缺导出所需列。

**方案**：

1. proto `PartnerExportItem`（无历史兼容包袱，直接改）：
   - `repeated PartnerRoleType roles = 6;` → `repeated PartnerRole roles = 6;`
     （携带 enabled/blacklisted，供「业务角色 (停用)/(黑名单)」标注）；
   - 新增 `repeated PartnerContact contacts = 7;`（proto 已有该消息，
     含 name/phone/is_primary，供「主要联系人」列）；
   - 新增 `string updated_at = 8;`（供「更新时间」列）。
2. service `ExportPartners`（partner_profile.go 202-237 行）：填充改为
   从 biz Partner 完整转换（角色、联系人、更新时间）；转换器复用
   `partner_convert.go` 既有函数，缺则按现有风格新增。
3. 前端 `handleExport`：改调 `partnerServiceExportPartners({ keyword,
   role, enabled, blacklisted })`（沿用 `searchParams` 与 `blacklistView`
   口径），`unwrapList` 后本地构建 XLSX；列头、(停用)/(黑名单) 后缀、
   联系人 `name(phone)` join、时间格式与现状一致；文件名黑名单前缀保持。

## 9. 追加设计：拉黑操作人姓名（阶段 D）

**问题**：`blacklisted_by` 契约只有用户 UUID，列表「拉黑信息」列显示
一串 ID。

**方案**（复用 `ListPartnerAuditLogs` 的批量联查范本，data/partner.go
195-209 行）：

1. proto `PartnerRole` 追加 `string blacklisted_by_name = 8;`。
2. biz `PartnerRole` 追加 `BlacklistedByName string`（biz/partner.go 75-83 行）。
3. data 层新增私有助手（如 `enrichBlacklistedOperatorNames(ctx,
   []*biz.Partner)`）：收集去重非 nil `BlacklistedBy` →
   `client.User.Query().Where(userent.IDIn(...))` 批量查 → 回填
   `BlacklistedByName`；在 `List` 与 `Get` 返回前调用（两条读路径契约
   一致；`SetPartnerRoleBlacklist` 结果经 `Get` 重读自动继承）。
4. service `partner_convert.go` 的 PartnerRole → API 转换输出
   `BlacklistedByName`。
5. 前端 `index.tsx` 拉黑信息列：`role?.blacklistedByName || role?.blacklistedBy`。

生成物流水线同阶段 A/B：`make -C server api` + `pnpm run
generate:web-client`，与源同组提交。
