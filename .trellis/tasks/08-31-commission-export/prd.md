# 提成导出（阶段 2）

## Goal

按当前筛选条件导出提成列表（本位币与 CNY 双口径），含独立导出权限与业务审计。
对应计划 §10 阶段 2。

## Requirements

- 契约：`GET /api/v1/finance/commissions/export`，权限
  `system.finance.commission.export`（依赖 read），响应返回
  `CommissionExportItem` 列表（计划 §6）。
- 请求复用列表筛选（keyword、status、commission_date_from/to）。
- 上限语义（§4.2.6）：单次同步导出上限 10000 行；先按筛选查总数，超限拒绝并
  提示缩小范围，不截断、不只导前 N 行；不超限按稳定排序分批查询。
- 排序与列表一致：`commission_date DESC, created_at DESC, id DESC`。
- 导出字段双口径：原始/调整/有效提成各两列（本位币 + CNY），另有提成编号、
  状态、核销编号、归属日期、提成员工、考核角色、规则名称、计提口径、比例、
  本位币币种、创建时间。
- CSV 由前端生成（BOM、字段转义、防公式注入），文件名含归属月份区间或导出
  日期；沿用 `ExportPartners` 先例（同步 JSON + 前端 CSV）。
- 审计（计划 §9）：`finance.commission.export`，记录操作人、组织、筛选条件、
  导出行数与成败；不保存完整导出内容。
- 权限码登记 `internal/access/manifest.go` 后运行
  `pnpm run generate:permission-keys`，生成物同提交。

## Acceptance Criteria

- [ ] 无导出权限拒绝；超上限拒绝且不加载全量数据。
- [ ] 导出行数与同筛选列表一致；顺序稳定。
- [ ] CSV 转义（逗号、双引号、换行）与公式注入防护有测试。
- [ ] 每次导出（含失败）有审计记录。
- [ ] `go -C server test ./...` 通过，权限生成物与源同提交。

## Notes

- 依赖：`08-31-commission-cny-snapshot` 已完成（导出依赖 CNY 字段与日期筛选）。
- 提交信息：`feat: 增加提成导出`。
- 实现前需补 `design.md` 与 `implement.md`。
