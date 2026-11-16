# 最终验收定位

这是AI实施期间的只读入口盘点，不代表本组已开始或四类财务链已通过。

## 可复用与需复核

- `scripts/run-acceptance-finance-disposable.mjs` 已有随机库/角色、owner确认、追踪进程组、
  信号回收及独立cleanup；SI浏览器和AI准备均在本机私有harness复用其最小阶段。
  最终验收继续这一受控方式，不用真实开发业务数据或真实员工通知。
- `scripts/acceptance-finance-{bill-batch,payable,foreign-currency}.mjs` 有现行财务API、
  发票登记、收付核销、提成及批次结果断言；现有订单夹具固定SE，不是四类完整证据。
  `acceptance-finance-fixtures.mjs` 的辅助建票也写死seaMasterBill，不可原样用于航空。
- `scripts/acceptance-se-order.mjs` 仍有旧shippingDocuments载荷；这是源码盘点，尚未运行。
  最终四类实际验收须消费当前真实海空实体协议，不能用该入口存在作为闭环已过。
- 各子组真实PG记录覆盖事务、正式迁移、权限/版本及相邻路径，不替代真实HTTP财务链；
  HTTP stub鉴权测试亦不能替代实际数据库写入。最终报告逐类绑定真实结果。

## 四类最薄财务样例

SE/SI/AE/AI各需真实订单及适用单证关系，至少各有应收与应付。按现行合同完成必要
业务/进口服务、费用录入、完结、可选审核锁与账单、发票登记或免票、收付款核销和提成；
不把“先开票”强加为核销前置，不以主单运价印字代替经营费用。AE/AI双模式的审核、
终止恢复与打印由对应专项证明，财务样例不重复穷举所有模式组合。

验收公司、用户、客商、账户和数据均随机伪造且仅在一次性库中。业务日期验证出口ETD、
进口ETA及中国自然日，实际收付按确认流水交易日，账龄按当前未清余额；三者不能互相代替。

## 列表/导出与横切

- 订单列表 `ListOrdersRequest`/过滤排序和读取权限是专用完整导出内核；独立导出不带分页，
  只读快照及完整行DTO沿已批准design，不用前端循环翻页。需真实超过200行文件核对，
  采纳筛选、排序、授权列、行数、本地时间/金额/小数/中文转义和迟到公司切换隔离。
- 工作台/待办/异常、通知深链与受控预览、台账号查询、经营报表四类实际权限矩阵需重新
  复核。SI/AI创建接通不证明这些出口硬编码已清除；不得借AE/SE权限或恢复业务标签。
- 前序阶段通过的check:fast/build只属于各自提交。最终横切/CSV完成后跑风险匹配最终门禁；
  航空PDF组的生产入口、字体和同Blob输出要有实际PDF及同域证据。

## PDF组期间追加的精确入口（仍为只读盘点）

- 当前`ListOrdersRequest`在`server/api/order/v1/order.proto`，包含page/pageSize、全部筛选、sortBy/sortOrder；新专用导出不复制分页参数，不改变常规列表200上限。
- `web/src/pages/orders/list.tsx`与`list-query.ts`承接kind和真实列表请求；公共`components/ui/order-list-template/OrderListTemplate.tsx`当前只有request返回，没有用于导出的onLoad采纳快照，Toolbar亦尚未接onExportTable。必须在官方采纳后保存本次筛选/排序，再给完整导出使用；不能仅取currentFilterRef或request完成时直接更新快照。
- `Data.withReadSnapshot`位于`server/internal/data/transaction.go`，现有`finance_report_{profit,cashflow,aging}.go`可作为真实调用范本；该只读快照和共享写事务ctx边界保持原合同。
- `workbench-todos.md`当前仍写SE/AE与两类出口直达，属于本组需要按真实实现同步的旧规范，不表示进口已接通。

已确认的产品硬编码（尚未修改）：

- `service/workbench.go`第31行只遍历SE/AE，`data/workbench_todo.go`第20行也仅接受SE/AE；需同时扩展作用范围与权限交集，不能只改前端链接。`web/pages/workbench/MyTodosDrawer.tsx`第61行及`WorkbenchSideCards.tsx`第258行附近亦仅处理两出口。
- `biz/notification.go`第232、262行及`notification_test_usecase.go`第49行只处理两出口；本组验证通知受控预览和深链即可，禁止向真实员工发送测试。
- `scripts/acceptance-finance-fixtures.mjs`已公开`prepareAcceptanceCompany`、`verifyCompositeFinanceWorkflow`等函数，可减少新验收编排重复；这些现有函数不是四类新建票协议的替代。
