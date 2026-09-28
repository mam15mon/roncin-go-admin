# Trellis 复核记录

## 结论

- 未发现本次清理引入的代码缺陷，未修改产品代码。两批提交仅删除无消费者的手写入口及其自证测试、无用导入和未生效的配置；未改权限码、`Requires`、路由、Proto、Ent Schema、迁移或生成物。
- 全仓当前源码引用核查没有命中 `IsBusinessOperationPermission`、`RequireBaselineWrite`、`CanTransitionTo`、`ComputeFingerprint`、`DocumentDetailLayout`、`orderFeeColumns` 或 `tailwind.config`。旧 Trellis 归档任务仍会提及这些历史入口，属于原任务记录。当前 `.trellis/spec/` 未引用上述已删标识，不需要同步规范。
- 费用补录仓储仍在 `LockForDecision` 对申请行 `ForUpdate()`，`SaveDecision` 保留 `version + PENDING` 条件更新和 `ErrFeeSupplementTransition`；本批集成测试补充重复驳回、撤回及并发失败方错误断言。拆票的前端 `split-fp:` 生成与服务端真实幂等分支均未修改。
- `tailwind.config.js` 删除前后的 CSS 产物已由实现批次逐字节对比为一致；本次复核检查了 Vite 插件和 CSS 中无 `@config` 引用，未发现构建配置遗漏。

## 复核命令

| 检查 | 结果 |
| --- | --- |
| `pnpm --dir web exec biome check src/components/ui/index.ts src/pages/orders/fees.test.tsx src/pages/orders/split.tsx` | PASS；3 文件，0 修复。 |
| `pnpm --dir web tsc` | PASS。 |
| `go -C server vet ./internal/access ./internal/biz ./internal/data` | PASS。 |
| `pnpm --dir web exec vitest run src/pages/orders/fees.test.tsx src/pages/orders/split.test.tsx src/pages/orders/components/fees/feeBaseColumns.test.tsx` | PASS；3 文件、22 用例。 |
| `go -C server test -p 8 ./internal/access ./internal/biz ./internal/data -run 'TestManifest|TestOrderFeeSupplement|TestBusinessWorkspace|TestWorkspaceSeparatesSystemManagementAndCompanyBusiness|TestPublicLocationWriteRequiresSystemWorkspaceAndPermission|TestSeaOrderChange_' -count=1` | PASS；3 包。 |

前端定向测试的标准错误仍出现 `localhost:3000` 的 `ECONNREFUSED`，但 22 个用例全部通过。该噪音不由本次删除的模块触发；本批不修改测试基础设施。真实 PostgreSQL 集成结果见 [server-verification.md](server-verification.md)，最终全量门禁由主会话记录。
