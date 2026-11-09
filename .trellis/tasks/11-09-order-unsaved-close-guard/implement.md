# 执行计划

## 规划与上下文

- [x] 创建任务，记录先前讨论与最新产品选择。
- [x] 完成订单草稿链路、页签/离开守卫两个独立研究。
- [x] 完成 PRD、设计与本执行计划；本期只有一个相互依赖的用户闭环，不拆无法独立验收的子任务。
- [ ] 完成上下文清单校验与最终方案审阅后，`task.py start` 进入实现。

## 实现顺序

1. 停止已有海运/空运详情读取及持久化草稿；保留原表单版本、单证基线和显式刷新隔离。
2. 复用实时 dirty 与已有路由离开 guard；同步修正外部只读后已有修改仍受保护、编辑同 tick 关闭的时序。
3. 明确仍有效的新建与独立 HBL 编辑持久草稿判定（包含来源 query 的新建路径），排除遗留详情草稿误报；不建立迁移或扫描清理链路。
4. 统一关闭/离开确认动作文案，补关闭全部；复用临时确认放行，避免批量关闭后路由重复弹框。
5. 工作区补原生 `beforeunload`；主动退出在请求前确认，失败保留编辑。
6. 功能完成后调整并补充针对性回归；同步规范。

## 代理分工与边界

- 实现由 `trellis-implement` 执行，按设计与研究直接实现，不再派发下级 implement/check。
- 检查由独立 `trellis-check` 执行，可修正范围内机械问题，设计或范围变化必须报告。
- 主会话维护任务、用户取舍、规范和最终提交/归档；保留其他窗口修改。

## 定向验证

根据最终文件拆分，优先运行对应集合：

```bash
pnpm --dir web exec vitest run src/components/layout/tabCloseGuard.test.ts src/components/layout/useRouteLeaveGuard.test.tsx
pnpm --dir web exec vitest run src/pages/orders/detail-draft-lifecycle.test.tsx src/pages/orders/detail-port-save.test.tsx src/pages/orders/detail-port-save-master.test.tsx
pnpm --dir web exec vitest run src/pages/orders/detail-change-notice.test.tsx src/pages/orders/detail-change-notice-air.test.tsx src/pages/orders/detail-termination-refresh.test.tsx
pnpm --dir web exec vitest run src/components/ui/order-template/OrderFormTemplate.test.tsx src/components/ui/order-template/OrderFormTemplate.draft-snapshot.test.tsx src/pages/orders/new-create-idempotency.test.tsx src/pages/orders/use-sea-order-create.test.tsx
pnpm --dir web tsc
git diff --check
```

新增关闭全部、原生离开及详情真实路由用例按实际新文件补入；不把超过 10 个重型渲染用例堆进已有长文件，不用无意义定时等待。

## 最终检查与验收

- [ ] 定向测试覆盖 PRD 全部验收项；修改文件 Biome 检查通过。
- [ ] 独立检查代理核对并发基线、保存失败/只读生命周期、批量关闭范围与一次确认。
- [ ] `pnpm run check:web` 通过（含类型、架构、生成物一致性及完整前端测试）。
- [ ] 没有修改生成物、依赖、服务端或其他窗口的既有改动。
- [ ] 规范已同步，Conventional Commit 中文提交说明准确描述最终行为。
- [ ] `trellis-finish-work` 归档当前任务并记录会话。

## 风险与回退点

- 新路由 blocker 需要真实 data router 测试，禁止为了测试环境跳过运行时保护。
- 删除旧恢复断言时须以活表单原版本/失败保护断言替代，不能连同并发测试一起删去。
- `component-guidelines.md` 已有他人修改；优先更新其他归属清晰的规范，确需合并时保留其原差异。
- 本任务不接管新建页排版，绝不使用 reset/checkout 覆盖该范围。
- 出现超出设计的跨模块状态平台需求时停止扩展，先回报范围差异。
