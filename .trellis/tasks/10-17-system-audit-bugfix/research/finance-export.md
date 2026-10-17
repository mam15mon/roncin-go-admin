# R3 财务台账导出实施与验收记录

## 行为与边界

- `FinanceLedgerTemplate` 移除从可见 ProColumns/dataIndex 推导 CSV 的逻辑，必传类型化 `exportColumns`；六个产品消费者全部同步。CSV 使用固定业务字段，不因界面列偏好丢掉币种。
- 范围保持「有选中行时导出选中行，否则当前页」，按钮显示「导出选中 (数量)」或「导出当前页」，提示和成功文案同步范围，不新增全量请求或循环翻页。
- 费用、账单、发票、流水、核销、对冲各有页面就近的 `ledgerExport.ts`。原币/本位币独立列，金额无千分位固定两位；数量、单价、税率、汇率用去尾零精度，日期时间统一 `formatDate`，纯日期保持原日期。
- 状态使用现有账单/费用进度/费用审核/发票映射；流水状态元数据和核销、对冲文案提取至各页导出模块供界面复用。发票销项已开具、进项已收票分别输出。
- 标签、关联账单、核销分配摘要显式展开为名称/单号/金额文本，不会输出对象字符串。费用 API 没有旧界面的主单号、人员、已核销金额等投影字段，导出只声明真实 API 字段，不伪造空字段或以 ID 冒充名称。
- 新增 `utils/csv.ts` 的最薄类型化序列化与下载：UTF-8 BOM、CRLF、双引号/逗号/换行转义，文本公式防护覆盖前导 Unicode 空白；负数仅在声明为数值且匹配十进制时保留为数值。空列表返回 null。下载成功与点击异常均在 finally 清理链接及 URL。
- 提成原有重复 CSV 实现同步消费公共工具，保持字段/比例精度，并将生成时间改为本地可读时间。
- 六页 index.tsx 只编辑导出 import/prop 与上述状态文案提取，写动作由 write_feedback 代理负责，未覆盖其编辑。

## 验证结果

- 定向 Vitest 10 文件、27 用例全部 PASS，耗时 5.95 秒，单个文件均低于 20 秒：
  - `utils/csv.test.ts`：BOM/CRLF/公式与空白/特殊文本/负数/小数/空值/空列表，以及成功和异常下载资源回收。
  - `FinanceLedgerTemplate.test.tsx` 与 `FinanceLedgerTemplate.export.test.tsx`：真实模板和真实表格请求协议；当前页/选中/空页点击下载、Blob 内容、币种列未在界面显示仍导出、总数100只导出当前页2行、下载不产生额外请求、URL与链接回收。
  - 六页 `ledgerExport.test.ts`：逐字段完整对象相等与列数验证，多币种第二行、负数、汇率/数量/单价/比例精度、中文状态、对象展开、特殊文本、UTC 时刻在上海本地输出。费用31列、账单25列、发票22列、流水19列、核销18列、对冲18列。
  - `commissions/commissionExport.test.ts`：既有序列化回归及新增本地生成时间回归。
- `pnpm --dir web tsc`：PASS（其他代理修复途中类型问题后重跑确认）。
- 本代理专属修改文件的 `biome lint`：PASS，零警告。六页共享 index.tsx 检查曾产生写代理新增 Promise<void | false> 的15条警告，已反馈 write_feedback 统一修复，不属于导出代码。
- `pnpm run check:architecture`：18个扫描器测试 PASS，产品文件444个/依赖1715条，零边界违规。
- `antd lint web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx --format json`：零问题。
- `git diff --check`：PASS。

## 交接与限制

- 未执行全量门禁、未提交，按分工由主会话集成、检查、更新规范和提交。
- 浏览器 CSV 样本复核留主会话；已提供真实组件下载与六类逐字段回归。导出始终是已加载的选中行或当前页，产品范围如此，没有承诺全量导出。
