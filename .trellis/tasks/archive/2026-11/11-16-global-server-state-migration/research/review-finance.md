# 财务迁移检查记录

## 已修复

- `ExchangeRateImportModal.tsx`：确认导入在途时，取消按钮、关闭叉号、遮罩、Esc 和重新上传入口原可丢弃本次提交上下文。现在统一冻结关闭与重传；失败仍保存原预检 token、载荷与幂等键，重试沿用原意图。
- `ExchangeRateSyncModal.tsx`：发布期间原可继续修改牌价、重新抓取或关闭弹窗，导致迟到发布响应作用于已变化的草稿。现在冻结内容及所有关闭入口，发布失败保留人工输入并恢复操作；抓取中不允许发布。
- `BillEditModal.tsx`、`CommissionApplicationDetailDrawer.tsx`、`CommissionRulesDrawer.tsx`、提成/发票详情页及自定义策略：查询接入 Query 的 `signal` 并设置 `skipErrorHandler`，保持原有 Query meta 负责提示，避免读请求取消产生全局错误通知。
- `CommissionCreateModal.tsx`：候选查询身份包含开合状态，关闭取消旧候选请求，重开不接收旧响应。`CommissionRulesDrawer.tsx` 的两条员工查询同样包含活动身份，关闭子表单或筛选后取消在途请求。
- `CommissionApplicationDetailDrawer.tsx`：初次加载失败原无明确失败与重试入口；现在显示就地错误并可重试。成功响应仍校验申请业务标识与明细数组类型，拒绝损坏响应冒充零笔合法明细，成功省略 repeated 明细继续合法为空。
- 三个本次修改的 Modal 关闭保护使用当前 antd `mask.closable` API，移除弃用的 `maskClosable`。

## 未修复问题

本次财务评审范围内未发现需要产品决策或跨模块调整的未决问题。已有官方 ProTable request/onLoad、提交身份保护、dirty、版本与幂等保护均继续保留。

## 验证

- 实施回归：财务 48 文件、225 例通过，见 `/tmp/finance-all-tests-final.log`。
- 评审相关相邻回归：9 文件、60 例通过，见 `/tmp/review-finance-targeted-final.log`；查询关闭相邻回归 4 文件、30 例通过，见 `/tmp/review-finance-close-queries.log`。
- 最后修复与新增回归：7 文件、29 例通过，见 `/tmp/review-finance-finish-tests.log`。覆盖导入未知结果同键重试、牌价发布冻结及失败输入保留、申请 undefined/失败信封/缺申请 ID/非法明细、申请换实体迟到隔离、提成候选关闭重开、方案员工关闭取消，以及 BillEdit 变动检查和刷新相邻路径。
- TypeScript 全仓通过，见 `/tmp/review-finance-tsc-complete.log`。
- 评审修改的 15 个源码/测试文件 Biome check 通过；三个 Modal 与申请详情的 Antd lint 无问题；`git diff --check` 通过。
- 既有状态管理规范已覆盖查询取消、失败响应、幂等与表单输入契约，本轮局部修复不新增公共能力或接口，无需另建规范。
- 未执行浏览器与后端门禁；完整 `check:web` 及统一提交由主会话完成，评审代理未提交。
