# 往来单位列表黑名单视图切换

## Goal

在客户/供应商/国外代理三类往来单位列表页增加 Segmented 分段切换，可切换到
黑名单视图只看已拉黑档案；后端 ListPartners 增加 blacklisted 过滤参数。

## 背景

- 黑名单能力已存在：`PartnerRole` 有 `blacklisted / blacklist_reason /
  blacklisted_at / blacklisted_by` 字段，设置弹窗与权限
  `business.partner.blacklist` 已就绪，列表角色列已显示 `[黑名单]` 标记。
- 缺口：`ListPartners` 接口无法按黑名单状态过滤，用户只能在普通列表里肉眼
  找 `[黑名单]` 标记，无法集中查看/管理黑名单档案。

## Requirements

1. **视图切换（前端）**：三类往来单位列表页（`/partners/customers`、
   `/partners/suppliers`、`/partners/foreign-agents`，共用
   `web/src/pages/partners/index.tsx`）的 ProTable 标题栏增加 Segmented
   分段器：「全部{客户|供应商|国外代理} | 黑名单{客户|供应商|国外代理}」。
2. **黑名单视图行为**：
   - 切到黑名单视图后，列表只显示当前角色类型已拉黑（`blacklisted=true`）
     的档案，分页重置到第 1 页；关键字等搜索条件保留。
   - 黑名单视图下表格增加「拉黑信息」列，展示当前角色类型的拉黑原因、
     操作人与时间（数据已在 `roles` 中返回，无需额外请求）。
   - 黑名单视图下「导出 Excel」导出的是黑名单档案列表。
3. **接口能力（后端）**：`ListPartners` 增加 `optional bool blacklisted`
   查询参数，按业务角色的黑名单状态过滤；`blacklisted=true` 时匹配
   「存在当前角色类型且已拉黑的角色记录」，**不要求该角色 enabled=true**
   （拉黑操作独立于停用，停用的黑名单也必须可见）。
4. **权限**：黑名单视图是已有 `business.partner.read` 列表的过滤形态，
   不新增权限码；拉黑/移出黑名单仍由既有 `business.partner.blacklist`
   控制（弹窗按现状不变）。

## 非目标（Out of Scope）

- 不做黑名单批量操作、独立黑名单管理页、URL 同步视图状态。
- 不修改 `SetPartnerRoleBlacklist` 写路径与拉黑弹窗交互。
- 不为「未拉黑」（`blacklisted=false`）提供前端入口（契约层支持，UI 不暴露）。

## 追加范围（用户于黑名单视图交付后确认修复，2026-09-20）

5. **导出正统化**：前端导出从「自行调 ListPartners 传 `pageSize: 2000`
   （会被后端 200 上限拒绝，功能实际不可用）」切换到既有的
   `ExportPartners` 接口（服务端聚合翻页，权限 `business.partner.export`）；
   `PartnerExportItem` 契约补齐现有导出列所需数据：角色带启用/黑名单
   状态、联系人列表、更新时间。导出文件列头与现状保持一致。
6. **拉黑操作人显示姓名**：`PartnerRole` 契约增加 `blacklisted_by_name`，
   data 层按审计日志既有模式批量联查 users 回填（List 与 Get 两条读路径），
   前端「拉黑信息」列优先显示姓名、缺失时回退 UUID。

## Acceptance Criteria

- [ ] proto 契约：`ListPartnersRequest` 含 `optional bool blacklisted = 7;`，
      生成物（pb.go / openapi.yaml / 前端 typings）与源同组更新，未手改生成物。
- [ ] 后端行为：`role + blacklisted=true` 只返回该角色类型已拉黑的档案；
      已拉黑但角色停用的档案仍出现在黑名单视图；`blacklisted` 缺省时行为
      与现在完全一致。
- [ ] 后端测试：biz 层仿 `TestPartnerListFiltersByIsCasual` 增加
      blacklisted 过滤用例（true/false/nil 三态）。
- [ ] 前端行为：三个视图均可切换；黑名单视图只显示已拉黑档案、分页回到
      第 1 页、搜索条件保留；「拉黑信息」列仅黑名单视图可见。
- [ ] 前端测试：`index.test.tsx` 增加切换交互用例（切换后请求携带
      `blacklisted: true`、列表刷新）。
- [ ] 校验：`go -C server test ./internal/biz/... ./internal/service/...`
      相关包通过；`pnpm --dir web exec vitest run src/pages/partners/index.test.tsx`
      通过；`pnpm --dir web tsc` 通过。
- [ ] 追加：导出走 `ExportPartners` 接口（断言不再调 ListPartners 传超限
      pageSize），黑名单视图导出携带 `blacklisted: true`，导出列头不变。
- [ ] 追加：「拉黑信息」列显示操作人姓名（`blacklisted_by_name`），
      契约与生成物同步更新。
