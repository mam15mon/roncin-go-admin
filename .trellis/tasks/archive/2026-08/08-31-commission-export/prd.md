# 提成导出（阶段 2）

## Goal

提供按当前筛选条件导出提成列表的后端 JSON 能力（本位币与 CNY 双口径），包含
独立权限、上限、稳定排序与成功导出审计。浏览器 CSV 属于阶段 3。

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
- 沿用 `ExportPartners` 的同步 JSON 接口形态，但本任务不实现前端 CSV。
- 审计（计划 §9）：后端成功查询全部数据后、返回响应前写入
  `finance.commission.export`，记录操作人、组织、筛选条件与最终行数，不保存完整
  导出内容；审计失败时接口失败。
- 超限、参数错误和后端查询失败由结构化请求日志记录；无权限由访问控制与请求日志
  记录，不写失败业务审计。
- 权限码登记 `internal/access/manifest.go` 后运行
  `pnpm run generate:permission-keys`，生成物同提交。

## Acceptance Criteria

- [ ] 无导出权限拒绝；超上限拒绝且不执行导出数据查询。
- [ ] 导出行数与同筛选列表一致；顺序稳定。
- [ ] 每个成功返回的导出都有审计；审计失败时不返回数据。
- [ ] 超限、参数错误、查询失败和权限拒绝不要求失败业务审计。
- [ ] `go -C server test ./...` 通过，权限生成物与源同提交。

## Out of Scope

- 前端导出按钮、CSV BOM/转义/公式注入和浏览器下载。
- 失败导出业务审计、异步导出、xlsx 和跨分页一致性快照。

## Notes

- 依赖：`08-31-commission-cny-snapshot` 已完成（导出依赖 CNY 字段与日期筛选）。
- 提交信息：`feat: 增加提成导出`。
- `design.md` 与 `implement.md` 已补齐；启动前必须通过上下文校验和最终规划批准。
