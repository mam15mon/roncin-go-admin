# 清理设计与边界

## 清理门槛

以“无运行时消费者且删除不改变行为”为门槛。前端从使用端向出口文件追踪，再检查路由、动态导入和 Vite 构建；后端从 service、data、server、cmd 入口反向追踪。生成代码、历史迁移和 API 契约不参与无引用删除。

## 前端

- 删除 `web/src/components/ui/document-detail-layout/`，同步移除 `web/src/components/ui/index.ts` 重导出；现行 `PageHeaderShell` 不改。
- 删除孤立的 `orderFeeColumns.tsx` 与 `fees.test.tsx` 对应完整 mock；保留实际页面使用的 `order-fee-panel-columns.tsx` 和 `feeBaseColumns`，验证费用页加载、表格列与操作入口。
- `split.tsx` 仅删 TypeScript 标明未使用的 import/别名，不移动函数、不改 JSX 与请求逻辑。
- `tailwind.config.js` 属条件项：先查 Tailwind v4 的显式／隐式加载，再以当前前端构建验证。若删除改变样式生成，保留并记录，不在本任务迁移样式配置。

## 服务端

- 删除 `IsBusinessOperationPermission` 分类 map/函数和 `RequireBaselineWrite` 时不触碰 Manifest 权限定义、`Requires`、`RequireGlobalMasterDataWrite` 或工作台判定。若发现实际权限码变化，视为超出范围。
- 比对费用补录 `Approve`、`Reject`、`Withdraw` 的状态检查、锁内时点和错误码。若能无行为差异地在三路径调用 `CanTransitionTo(target)`，让它成为实际单一规则；若会改变错误映射或要求跨层重构，则删除孤立函数，保留生产路径并验证终态拒绝。状态检查不得移出事务或行锁。
- 删除 `ComputeFingerprint` 只涉及未调用辅助函数和自证测试。服务端真实请求指纹比对、前端 canonical SHA-256 与幂等键不变；不以 Go helper 替换浏览器算法。

## 取舍与回滚

删除优先于创建通用抽象。按前端孤立模块、服务端纯死入口、状态/指纹条件项分批提交，任一批次可单独回退。静态无引用有动态加载盲区，以定向构建、类型检查和路径测试补证；若发现消费者，记录保留原因并移出删除批次。

本任务不调整数据库、API、权限码和用户可见流程，不需要数据迁移或兼容窗口。
