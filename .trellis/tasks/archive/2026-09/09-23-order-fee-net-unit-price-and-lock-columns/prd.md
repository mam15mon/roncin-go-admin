# 订单费用列：新增「不含税单价」可选列

## Goal

参考竞品费用列配置，在订单费用录入页新增「不含税单价」独立可选列，让操作员扫表时直接看到单价的不含税口径，替代目前藏在税率列里的「未税单价」Tag。

## Background（代码��据）

- 列清单与默认显隐：`web/src/pages/orders/components/fees/feeColumnPreference.ts`（`FEE_COLUMN_KEYS` / `DEFAULT_FEE_COLUMN_DEFS`）；可选只读列：`orderFeeOptionalColumns.tsx` `buildOptionalFeeColumns`；唯一消费入口 `OrderFeeTableTabs.tsx:1093`。
- 现状：无独立不含税单价列，仅在税率列对 `taxInclusive === false` 的行附「未税单价」Tag（`orderFeeOptionalColumns.tsx:149-153`），既有测试 `OrderFeeTableTabs.feeColumns.test.tsx:218` 断言该 Tag 存在。
- 单价口径：前端行内编辑与弹窗新增/编辑均固定提交 `taxInclusive: true`（`OrderFeeTableTabs.tsx:638`、`fees.tsx:415`），`taxInclusive=false` 仅存在于历史行。服务端只返回总额口径的 `netAmount`/`taxAmount`，不返回不含税单价。
- 行内编辑预览 `RowEditPreview`（`OrderFeeTableTabs.tsx:91`）当前含 `amount.total`、`rate`、`feeCode`、`taxRate`，不含单价；税金/不含税总额列在编辑中按 `amount.total` 与费用项目默认税率实时折算。
- 旧偏好兼容：`resolveFeeColumnPreference` 对存储中未提及的新列按 `defaultVisible` 处理，新增默认可见列会自动出现在老用户表格中。
- 竞品「是否锁定」不需要新增：费用状态列已有「已进账单」（`web/src/constants/statusMeta.ts:66`），账单创建置 BILLED（`server/internal/data/finance_bill_write.go:87`）、取消回退 CONFIRMED（同文件 :270），对所有用户可见。

## Requirements

- R1 新增列 key `netUnitPrice`、标题「不含税单价」，加入 `FEE_COLUMN_KEYS` 与 `DEFAULT_FEE_COLUMN_DEFS`，默认可见、可隐藏、可排序，默认顺序紧跟「单价」之后；偏好存取沿用现有机制。
- R2 已保存行的取值：`taxInclusive === false` 时直接显示单价；否则有税率时显示 `单价 ÷ (1 + 税率/100)`，两位小数 ROUND_HALF_UP；税率缺失显示 `-`；右对齐等宽字体，与税金列样式一致。
- R3 行内编辑中：预览含单价与费用项目税率时按 R2 含税口径实时折算并显示；缺少输入时与税金列一致显示「待保存」。`RowEditPreview` 需补充单价字段。
- R4 移除税率列上的「未税单价」Tag 及其提示；历史不含税行由本列（数值等于单价）表达。
- R5 更新对应单测：列 key 清单、默认顺序、含税/不含税/缺税率三种取值、编辑中实时预览、Tag 已移除。

## Acceptance Criteria

- [ ] A1 列设置弹窗出现「不含税单价」，默认勾选且位于「单价」之后；隐藏后刷新页面仍隐藏（R1）。
- [ ] A2 `taxInclusive=true`、单价 106、税率 6 的已保存行显示 100.00；`taxInclusive=false`、单价 100 显示 100.00；税率为空显示 `-`（R2）。
- [ ] A3 行内新增行输入单价并选择带默认税率的费用项目后，本列实时显示折算值；未输入单价时显示「待保存」（R3）。
- [ ] A4 税率列不再渲染「未税单价」文案（R4）。
- [ ] A5 `pnpm --dir web lint`、`pnpm --dir web tsc`、`pnpm --dir web test -- fees` 通过（R5）。

## Out of Scope

- 「是否锁定」及锁定人/时间列：状态列「已进账单」已覆盖，账单审计可追溯。
- 「创建时间」「创建人」列：用户明确本次不做。
- 结算币种/汇率/金额、发票号/开票时间下沉、佣金比例、VAT、所属公司、标记入账月份、关联信息、费用编号：域模型或产品决策，不在本任务。
- 后端接口不改动。

## Risks

- 反算的不含税单价与服务端 `netAmount ÷ 数量` 可能存在分位舍入差；本列为展示口径，不参与提交。
- 历史不含税行去掉 Tag 后，仅能通过「不含税单价 = 单价」间接识别；前端已不再产生此类行。

## 实施后发现（2026-09-23）

- protojson 默认省略零值字段：`tax_inclusive=false` 不出现在 HTTP 响应 JSON 中，
  wire 上「字段缺失（undefined）」即历史不含税行。据此按既有规范
  （`.trellis/spec/web/frontend/type-safety.md`「服务端布尔响应的零值省略陷阱」：
  判断后端 false 一律 `!== true`，禁止 `=== false`）实现本列的不含税分支，
  历史不含税行可正确直取单价展示；测试同时覆盖 wire 真实形态（字段缺失）
  与类型层显式 `false` 两种载荷。
