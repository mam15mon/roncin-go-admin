# 清理无用代码与误导性入口

## 目标

删除已证实没有生产消费者的手写代码，收敛误导维护者的孤立状态规则与测试，并保持现有路由、权限、财务、订单行为不变。以实际引用和运行验证为依据，不以 `legacy`、`mock` 等名字判断是否无用。

## 审查依据

- `web/src/components/ui/document-detail-layout/DocumentDetailLayout.tsx:9-70` 只经自身和 `web/src/components/ui/index.ts:6` 转导出，没有业务或测试消费者；面包屑、单号参数已标废弃，现行页头使用 `PageHeaderShell`。
- `web/src/pages/orders/components/fees/orderFeeColumns.tsx:1-55` 无生产导入；唯一外部引用是 `web/src/pages/orders/fees.test.tsx:237` 的完整 mock，实际页面使用 `order-fee-panel-columns.tsx` 等列定义。
- `server/internal/access/manifest.go:458-501` 的 `IsBusinessOperationPermission`、缓存 map 和分类函数只互相调用，无外部消费者；实际工作台权限判定走现有入口。
- `server/internal/biz/organization.go:39-48` 的 `RequireBaselineWrite` 无调用，现行公共主数据写入使用 `RequireGlobalMasterDataWrite`。
- `web/src/pages/orders/split.tsx:1-90` 在临时开启 `noUnusedLocals/noUnusedParameters` 后暴露约 21 项未使用导入；全仓同一诊断共 212 处，其中 189 处是未使用的 `React` 默认导入。该诊断不是当前常规门禁。
- `server/internal/biz/order_fee_supplement.go:74-87` 的 `CanTransitionTo` 只被自身单测调用；生产审批、驳回、撤回路径各自检查 `PENDING`。注释称它是“唯一流转规则”，与现实不符。
- `server/internal/biz/sea_order_change.go:1093-1097` 的 `ComputeFingerprint` 仅被一个孤立测试调用；实际拆票输入指纹由前端 canonical SHA-256 生成，服务端校验请求指纹。
- `web/tailwind.config.js` 未发现显式引用；当前使用 Tailwind v4 Vite 插件。它只是配置候选，最终以构建是否读取它为准。
- 函数重复扫描覆盖前后端 828 个手写文件、7904 个函数，没有扫描错误；仅 2 组早已记录且因跨协议／分层需要保留的映射，不是本任务合并目标。

## 范围与要求

1. **确定性死代码**：实施前再次检查静态和动态引用，删除无消费者的 `DocumentDetailLayout`、`orderFeeColumns`、过期测试 mock 及对应 barrel 导出；删除未使用的业务权限分类链和 `RequireBaselineWrite`。不得改动真实权限集合、路由、DTO、迁移或生成物。
2. **局部未使用导入**：只清理本次触碰文件与 `split.tsx` 中诊断明确指出的未使用导入，不为全仓 189 处 `React` 导入制造大范围格式提交。记录剩余数量，供后续独立批次处理。
3. **误导性状态规则**：核对费用补录审批、驳回、撤回的目标状态、锁和错误映射后，选择最小的单一规则实现：生产路径接入 `CanTransitionTo`，或删除孤立函数及自测。两种处理都必须保证 `PENDING` 竞争和终态拒绝行为不变，并保留有价值的用例级验证。
4. **孤立指纹辅助函数**：核对前端 canonical 输入、服务端校验与幂等测试后，若 `ComputeFingerprint` 只测试自身，则删除函数和自证测试；不得改变拆票请求指纹算法、输入格式或幂等拒绝行为。
5. **配置候选**：只有确认 Tailwind v4 构建未读取 `web/tailwind.config.js` 且相关构建通过时才删除；若工具链隐式读取，保留并记录用途。

## 验收标准

- [x] A1：所删模块、函数、导出、测试 mock 和配置均有最终引用核查记录；没有坏导入或僵尸测试，不删除仍被运行时、生成器或构建工具使用的代码。
- [x] A2：费用补录审批、驳回、撤回及并发终态竞争保持原有状态与业务错误；拆票预览/执行的请求指纹和幂等路径保持原行为。
- [x] A3：前端受影响测试、Biome、`web tsc`、架构检查，服务端受影响包测试与 `go vet`，`git diff --check` 全部通过。最终按根目录 `AGENTS.md` 运行 `pnpm run check:fast`，如实记录 PASS/FAIL/SKIP。
- [x] A4：改动限于无用代码和必要的测试同步，文档及提交信息使用中文，按可验证批次提交；生成物、正式迁移和开发数据库不变。
- [x] A5：报告列出已删除项、保留项及证据、剩余未使用导入数量，明确哪些候选因工具链或业务契约而未删除。

## 不在范围

- 全仓 189 处 `React` 默认导入的机械批量清理；依赖卸载与锁文件大范围变更；抽象化两组合理重复映射。
- 有实际调用或明确契约的 `LegacyReadOnly`、旧单证接口、开发免密数据、Admin 旧 query 重定向、proto/Ent/OpenAPI 生成物和历史 SQL 迁移。
- 因“兼容”字样而删除现行响应解包、权限判定或组织边界逻辑。

## 执行状态

已按 [执行清单](implement.md) 完成清理、定向验证和全栈门禁；逐项证据见 [最终验证记录](research/verification.md)。
