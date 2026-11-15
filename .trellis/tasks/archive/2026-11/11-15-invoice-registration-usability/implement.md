# 实施计划

## 步骤

1. `features/finance/invoice-registration/autofill.ts` 纯函数 + `autofill.test.ts`
   （建议税额 6%/0/未设置/非法输入、autoAllocate 顺序/余额不足/跨发票占用、followTotal 手改保护）。
2. `types.ts`：草稿类型增加 `*Source` 可选字段；`invoiceRegistrationPayload` 不输出这些字段（现有测试回归）。
3. `InvoiceRegistrationEditor.tsx`：来源标记、联动、差额展示、余额校验、“按可登记余额分配”、建议提示文案；文件过大时把 `AllocationFields`/费用行拆成同目录组件。
4. `InvoiceRegistrationModal.tsx`：登记模式打开后自动建一张发票（单账单单费用时带好分配）；修订预填标 manual；接入 `InvoiceRevisionDiff`。
5. `InvoiceRevisionDiff.tsx` + 测试。
6. 前端组件测试：简单场景只填四项可提交、手改后改总额不覆盖并阻止提交、多费用分配按钮、修订对照、依赖阻断回归。重型用例按需拆 `*.autofill.test.tsx`，单文件 ≤10 个重渲染用例。
7. 更新 `.trellis/spec/server/backend/finance-invoice-registration.md`（建议税额口径）与必要的能力导航。
8. 浏览器验证（`pnpm dev`，开发库测试数据，可按需注入/修改样本）：简单登记、超额阻断、手改保护、修订对照、有核销时阻断。

## 验证命令

```bash
pnpm --dir web exec vitest run src/features/finance/invoice-registration/ src/pages/finance/bills/components/BillInvoiceRegistrationSection.test.tsx src/features/finance/bill-creation/
pnpm --dir web exec biome check src/features/finance/invoice-registration
pnpm --dir web tsc
go -C server test -p 32 ./internal/biz ./internal/service -run 'TestInvoiceRegistration|TestNormalizeInvoice|TestInvoiceRevision|TestInvoiceAllocation'
go -C server test -p 32 ./internal/data -run '^TestFinanceInvoice(Registration|RevisionReadSnapshot)' -count=1 -timeout=5m   # 需 RONCIN_INTEGRATION_DATABASE_SOURCE
git diff --check
pnpm run check:fast   # 最终验收
```

## 提交分组

- `feat(finance): 发票登记自动带入分配与建议税额`（步骤1–4、6）
- `feat(finance): 发票修订展示原值与修改后对照`（步骤5）
- `docs(spec): 发票登记建议税额口径`（步骤7）

## 风险点

- 共享编辑器也用于建账附带登记：回归 `bill-creation` 测试，确认不会自动生成发票记录。
- Form.List 内 `onChange` 与 `setFieldValue` 混用：只在事件中写值，禁用 effect 同步。

## R8 执行清单

1. biz 票种常量/判定 + 校验改造 + 单测。
2. Ent Schema 枚举、`go generate`、新增 SQL 迁移（映射旧值）、`pnpm run migrate:dev`。
3. proto 注释、`make -C server api`、`pnpm run generate:web-client`。
4. 前端共享票种模块 + 各展示/选择处替换；抬头默认票种带入登记。
5. 验证：`go -C server test ./internal/biz/... ./internal/service/...`、vet、定向 vitest、tsc、check:architecture、浏览器登记数电专票一次；提交。
