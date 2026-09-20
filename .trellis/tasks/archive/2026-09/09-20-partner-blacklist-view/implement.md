# 执行计划：往来单位列表黑名单视图切换

前置：`design.md` 已定稿；工作树有用户未提交改动，任何一步都不得
reset/checkout 覆盖无关文件。

## 执行清单（按序）

### 阶段 A：后端契约与实现

- [x] A1 `server/api/partner/v1/partner.proto`：`ListPartnersRequest` 追加
      `optional bool blacklisted = 7;`（含 design §1 注释）。
- [x] A2 `make -C server api` 重新生成；检查 diff 仅含 ListPartners 相关
      （partner.pb.go 的 ListPartnersRequest/getter、openapi.yaml 的
      blacklisted query 参数）。
- [x] A3 `server/internal/service/partner_profile.go`：ListPartners 与
      ExportPartners 两处按 design §2 透传 `Blacklisted`。
- [x] A4 `server/internal/biz/partner.go`：`PartnerListOptions` 追加
      `Blacklisted *bool`（biz 用例不加校验）。
- [x] A5 `server/internal/data/partner.go`：`List` 按 design §4 谓词分支
      重写 role/黑名单组合过滤。
- [x] A6 `server/internal/biz/partner_test.go`：新增
      `TestPartnerListFiltersByBlacklisted`（true/false/nil 三态）。
- [x] A7 验证：`go -C server test ./internal/biz/... ./internal/service/...`
      与 `go -C server vet ./...`。

### 阶段 B：前端契约与实现

- [x] B1 `pnpm run generate:web-client`；检查 typings.d.ts 中
      `PartnerServiceListPartnersParams` 新增 `blacklisted?: boolean`。
- [x] B2 `web/src/pages/partners/index.tsx` 按 design §5：
      `blacklistView` state、headerTitle Segmented、request 追加参数、
      拉黑信息列（`hideInTable: !blacklistView`）、导出透传 + 文件名前缀。
- [x] B3 `web/src/pages/partners/index.test.tsx`：新增切换用例
      （design §6）；如 Segmented 交互受动画影响，遵循全局无动画规则
      用 `fireEvent.click` / `userEvent` 即可，禁止 setTimeout 等待。
- [x] B4 验证：`pnpm --dir web exec vitest run src/pages/partners/index.test.tsx`
      + `pnpm --dir web biome:lint src/pages/partners/` + `pnpm --dir web tsc`。

### 阶段 D：追加修复（导出正统化 + 拉黑操作人姓名，design §8/§9）

- [x] D1 proto：`PartnerExportItem` 扩展（roles 改 `repeated PartnerRole`、
      新增 contacts/updated_at）+ `PartnerRole.blacklisted_by_name = 8`；
      `make -C server api` 后核对生成物 diff。
- [x] D2 biz `PartnerRole.BlacklistedByName`；data 层批量联查助手并在
      `List`/`Get` 返回前调用（范本 data/partner.go 195-209）。
- [x] D3 service：`ExportPartners` 完整填充（转换器复用 partner_convert.go，
      缺则新增）；PartnerRole 转换输出 `blacklisted_by_name`。
- [x] D4 `pnpm run generate:web-client`，核对 typings diff。
- [x] D5 前端：`handleExport` 切 `partnerServiceExportPartners`（列头/
      标注/文件名不变）；拉黑信息列姓名优先回退 UUID。
- [x] D6 测试：`index.test.tsx` 导出用例断言改走 export 接口（含黑名单
      视图传参）；拉黑列姓名断言；后端定向 `go -C server test
      ./internal/biz/... ./internal/service/...`。
- [x] D7 全栈门禁（等效 check:fast 分解，主会话执行）后提交。

### 阶段 C：联调与收尾

- [ ] C1 手动联调（用户或实现代理执行）：`pnpm run dev:server` +
      `pnpm run dev:web`，三个视图切换黑名单，验证停用+拉黑档案可见、
      搜索条件保留、导出内容与视图一致。
- [x] C2 按风险选择最终校验：本任务为契约变更，按 AGENTS.md 运行
      `pnpm run check:fast`（说明风险依据：跨层契约 + 生成物）。
- [ ] C3 提交：proto 源 + 全部生成物 + 前后端实现 + 测试同一提交，
      Conventional Commit 建议
      `feat: 往来单位列表支持黑名单视图切换`。

## 验证命令速查

```bash
make -C server api
pnpm run generate:web-client
go -C server test ./internal/biz/... ./internal/service/...
go -C server vet ./...
pnpm --dir web exec vitest run src/pages/partners/index.test.tsx
pnpm --dir web biome:lint src/pages/partners/
pnpm --dir web tsc
pnpm run check:fast        # 阶段 C2
```

## 回滚点

- 单一提交承载全部变更；任一阶段验证不过即停，不提交半成品。
- revert 该提交即完整回滚（无迁移、无权限清单变更、无状态数据）。
