# 结案门禁实施与验证计划

## 进入条件

业务结案与财务处理分离已确认；本设计的第一阶段规划总览确认后进入实现。依次使用 trellis-implement / trellis-check，主会话负责规范与提交；禁止先写测试再实现。

## 顺序

- [ ] 扩展 biz readiness 事实；只读和现有订单锁内复核都检查 PENDING 补录，错误原因一致。
- [ ] 复核补录各入口在 CLOSED 下仍可运行，保持现有授权、费用批准、版本和提成保护。
- [ ] 在订单详情消费既有 pending 投影展示结案阻断原因；保留服务端失败不误成功和反结案行为。
- [ ] 实现后补领域、真实 PostgreSQL 与前端回归，逐项对应 AC1–AC6；不要向钉钉实际发送通知。
- [ ] 并发覆盖补录先/结案先两种实际顺序，批准后未建账仍挡结案；记录 PASS 而非 SKIP。
- [ ] 更新费用审核及订单生命周期相关规范，定向检查通过后提交；第一阶段最后跑完整门禁。

## 验证命令

新后端测试统一用 TestOrderClosurePendingSupplement 前缀；配置隔离集成数据库后执行：

```bash
go -C server test -p 32 ./internal/biz -run 'TestOrderClosure' -count=1 -v
go -C server test -p 32 ./internal/data -run 'TestOrderClosurePendingSupplement|TestOrderLifecycleGatePostgres|TestFeeSupplementApprovalAdmissionPostgres' -count=1 -v
pnpm --dir web exec vitest run src/pages/orders/detail-status-transition.test.tsx src/pages/orders/order-detail-transitions.test.tsx
pnpm --dir web tsc
git diff --check
```

根据实际改动对修改文件执行 Biome 检查，增加的 CLOSED 后补录回归也必须在命令匹配范围内。重型组件 Suite 超过 10 个用例或超过 20 秒时按功能拆分。完整第一阶段执行 `pnpm run check:fast`，无需固定 build。

## 交付检查

- [ ] 专用集成数据库使用 `getIntegrationData` 正式迁移隔离 Schema；不自动重置开发库、不把凭据写入日志或文档。
- [ ] 费用及利润可以在受控更正后变化，但订单结案状态不被自动修改；财务收付不中断。
- [ ] acceptance.md 记录命令、结果、并发顺序和未覆盖项；总任务仍保留更正子任务，不以第一阶段完成结案归档总任务。
