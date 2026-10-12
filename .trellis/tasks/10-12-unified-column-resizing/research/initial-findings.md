# 初步证据
费用专属表头、列宽状态、双击装配均位于 FinanceLedgerTemplate.tsx；唯一真实 components.header.cell 配置也在该模板。目前业务没有其他自定义 header.cell，但公共层仍需保留官方 components 组合。
公共三类组件在 ResponsiveTable.tsx，布局递归规则见 layout.ts/useResponsiveTableLayout.ts。列设置输出按 stable key 管理，消费真实认证身份；此次宽度无需存入该偏好。首尾固定通过 start/end 适配 Pro 内部排序，不应破坏。
既有参考线测试在 FinanceLedgerTemplate.test.tsx；固定/编辑节点行为测试在 responsive-table/*.freeze.test.tsx。浏览器可使用本地 8001 Vite 与 8000 服务端，Playwright 已安装。已有列冻结验收文档在 archive/2026-10/10-10-column-fixed-preferences/research/browser-notes.md。
antd CLI 已查询 Table 官方 API，支持 columns/onHeaderCell/components；没有 resizable-column 内置 demo，需要沿用仓库自有实现。
