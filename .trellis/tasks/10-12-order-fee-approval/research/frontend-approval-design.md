# 费用审核前端衔接研究

## 已核实入口

| 入口 | 当前结构 | 本次衔接 |
| --- | --- | --- |
| `web/src/pages/orders/list.tsx:155` | OrderListTemplate，已有批量标签回调、费用抽屉、提成摘要下钻 | 注入末列审核组件与批量审核动作，消费服务端订单审核投影 |
| `web/src/components/ui/order-list-template/OrderListTemplate.tsx:272` | extraColumns 位于表格中部 | 增加最小 trailingColumns 插槽，业务组件由页面传入，公共 UI 不导入 features/pages |
| 同文件 `:435`、`:646` | 末尾操作列、现有 rowSelection | 审核列默认在操作列后；复用多选，不能把整张列表勾选能力限制为可审批，以免破坏批量标签操作 |
| `web/src/components/ui/order-list-template/OrderListSearchFilter.tsx:266` | 提交查询、恢复筛选值 | 审核状态、待审补录筛选加入已提交查询与 chips、重置、页签恢复 |
| `web/src/pages/orders/list-query.ts:23` | 请求映射与结果投影 | 传递新筛选字段；不在当前页本地过滤，不从旧订单业务状态猜审核状态 |
| `web/src/pages/orders/fees.tsx` | 完整费用页、费用汇总、建账入口 | 增加整票状态区、提交／撤回／审核／撤销批准及审计入口；普通编辑禁用叠加业务锁／财务锁 |
| `web/src/pages/orders/order-fee-panel.tsx:113` | 列表费用抽屉已有双锁写门禁 | 同样消费审核冻结能力，不能成为普通新增修改旁路 |
| `web/src/pages/orders/components/fees/FeeSupplementSection.tsx:788` | 已有补录审核窗口及利润前后对比 | 扩展统一费用审核资格、整票快照版本、亏损意见、提成冲减前后确认文案 |
| `web/src/features/finance/bill-creation/` | 订单费用、费用台账、账单页共用建账工作台 | 候选返回 canBill / blockedReasons / feeReviewStatus，预览与创建失败时准确刷新候选及提示 |
| `web/src/pages/finance/commissions/components/PendingDecreasePanel.tsx:86` | DRAFT 确认、忽略，CONFIRMED 标记已扣回 | 补录来源改为已确认待执行；移除该来源的确认／忽略入口，保留合法实际执行登记 |

## 界面决定与实现边界

1. 费用审核列承载文字状态与动作，不同时再增加重复的普通审核状态列；待审且有资格显示高亮按钮，已批准显示状态及待审补录徽标。服务端决定行能力，页面 access 只作粗入口判断。
2. 列默认置于最末，接入已有 column-settings 的稳定 key、结构列与首尾冻结机制。必须验证已有用户列偏好、列重排、390px 窄屏和宽屏恢复，不能靠硬编码多列 fixed 推翻全局末列规则。
3. Popover 支持 hover/focus，触屏点击打开审核详情；概要含本位币、应收／应付笔数与合计、利润、利润率、提交人及本地时区提交时间。不从前端浮点相减生成权威利润。
4. 完整审核用 Drawer 展示完整应收／应付列表及历史，批准／退回表单走 ModalForm.onFinish；负利润意见逐票必填，退回原因必填，普通批准意见选填。加载失败时不能留下可提交的旧快照。
5. 审核详情和批量预览通过生成客户端及 React Query 获取，queryKey 含组织、订单、申请／版本。hover 不为每行预取全部费用；当前页摘要由列表后端批量投影，真正打开再加载明细。工作区切换清除查看上下文，不能闪现其他组织缓存。
6. 批量只处理用户明确选择的订单，不提供“把搜索命中的所有页都批准”。确认窗口逐票显示能力与亏损意见，支持逐票展开完整明细；不能只显示合计利润。提交带每票预览身份和版本，不用页面布尔值代替后端复核。
7. 批量建议逐票独立结果。成功行移除选择，失败行保留原因供刷新重审，不能在自动刷新后自动批准新版本；网络结果不明用相同动作键查询／重试，不重复产生审批与通知。
8. 列表继续 ProTable request，操作成功通过拥有列表的 actionRef.reload 刷新；其他摘要 Query 按领域失效。通用组件只提供插槽与动作参数，不内部请求审批接口，也不层层传递手工 fetchList。
9. 共用展示数据类型使用生成契约。订单模块的审核 Drawer 留 pages/orders/components/fees；订单与财务均消费的状态标签／候选原因如需共享，放 features/orders/fee-review，经 index.ts 暴露最小能力并更新能力导航。通用 constants/statusMeta 可登记枚举映射，不反向导入 features。
10. 金额 formatAmount，利润率 trimDecimal，日期 formatDate；不能把后端 decimal 尾零或 UTC 原字符串直接渲染。

## 针对性验证

- 状态筛选正确传后端、总数及翻页、筛选组合、页签恢复／重置／组织切换。
- 有权／无权／自审／跨组织行；没有授权的响应不包含利润或其他员工提成信息。
- hover/focus 显示与错误加载、点击明细、负利润意见、批量混合成功失败、重复点击与超时重试。
- 审核期间和获批后，完整费用页、费用抽屉、批量编辑均禁用普通变更；接口失败保留表单输入。
- 三个建账入口筛选一致、补录待审原因可见、创建前版本变化明确拒绝。
- 补录显示完整前后利润及逐原提成单冲减；批准后展示已确认待执行，不残留“忽略建议”。
- 公共列插槽不能改变其他列表行为，真实浏览器验证末列可达、键盘焦点、浮层遮挡和不同宽度。
- 相关 Vitest 文件按交互拆分，单文件不超过 10 个重型用例；最小集通过后任务验收执行 check:fast，不以文档检查代替代码验收。
