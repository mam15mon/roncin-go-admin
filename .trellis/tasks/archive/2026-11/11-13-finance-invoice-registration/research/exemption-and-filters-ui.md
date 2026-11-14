# 无需发票与发票筛选：前端完成记录

日期：2026-11-14。

## 本组边界

补齐账单详情中的无需发票决定及历史，发票列表接入既有统一筛选设置。接口、Schema、权限清单和正式迁移由主代理实现并生成；本组没有手改生成客户端，没有触碰部分发票金额/税额分配界面。

## 行为

- 账单详情独立显示免票记录，包含原因、标记人和时间、撤销人和时间、撤销原因；未出现 revokedAt 的记录为有效。人员显示服务端姓名，缺名显示「未命名人员」，不向业务渲染人员 UUID；时间复用 formatDate 的 datetime 精度。
- 标记/撤销入口独立消费 canExemptFinanceInvoices，不依赖费用编辑权限。父页同时检查认证公司、来源账单 organizationId 等值；无权限仍可读历史，外公司或缺归属不提供写入口。只有已确认且身份、版本完整的账单可操作；已有有效发票时禁止新增免票。
- 原因必填，保存通过 GuardedModalForm.onFinish 的官方 Promise 生命周期；保存失败保留输入。请求提交打开表单时的 expectedVersion，父级新数据与变动探测不更换基线。成功报文须含同一账单 ID 和版本，异常报文明示失败。
- 打开编辑时复用 useBillChangeNotice 的前台 15 秒与恢复焦点检查，外部变动只提醒。刷新清除原因前先确认；切换账单、移除权限、关闭详情后，旧请求不能回填新详情。卸载清理刷新确认弹层。
- 保存成功将服务端账单交父级更新详情并通过 actionRef.reload 刷新列表；标记/撤销历史从服务端投影读取，不用清空号码或零税率冒充。
- 发票列表复用 SearchFilterTemplate、useFilterSettings、useListQueryState 和 useLedgerSelectionScope，字段为关键字、方向、状态。移除字段清值并回首页刷新，同时改变选择范围；表格、完整筛选汇总和导出仍走 FinanceLedgerTemplate 的官方采纳快照。
- 字段偏好按用户、公司、页面保存在浏览器，只存字段名，不存关键字；隐藏字段的恢复值被剔除。公司工作区不作为可移除字段。

## 文件

产品与说明：

- web/src/pages/finance/bills/components/BillInvoiceExemptionSection.tsx
- web/src/pages/finance/bills/components/BillDetailDrawer.tsx
- web/src/pages/finance/bills/index.tsx
- web/src/pages/finance/invoices/index.tsx
- .trellis/spec/web/frontend/capability-navigation.md（筛选接入导航）

针对性验证：

- 新增 BillInvoiceExemptionSection.test.tsx、invoice-exemption-access.test.tsx、invoices/filter-settings.test.tsx。
- invoices/current-workspace.test.tsx 的查询用例改为先提交自定义筛选；BillDetailDrawer.test.tsx 等待新增历史表格的异步渲染收敛。

## 验证

- 定向 7 文件、25 用例通过，覆盖免票标记/撤销、历史 wire 缺省形态、权限与当前公司、失败保留、并发基线、外部变动确认、迟到响应、发票筛选清值与偏好，以及相邻账单保存/筛选。
- 后补成功报文缺少版本的回归后，免票组件 9 用例单独复验通过；合计覆盖 26 用例。
- pnpm --dir web tsc：本组实施后已通过；最终复验遇到相邻代理修改 BillFeeMaintenance.tsx 的 response.data 未收窄类型错误，已交该代理修复，最终全仓类型结果由主代理复核。
- 修改文件 Biome 检查、git diff --check：通过。
- antd lint 新免票组件与发票页的弃用检查：无问题。

没有进行真实浏览器视觉验收；发票指定金额/费用分配及税额算法仍由主任务按用户确认推进，本记录不宣称整个发票任务完成。
