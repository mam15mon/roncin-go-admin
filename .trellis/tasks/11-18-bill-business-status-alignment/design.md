# 技术设计

## 最小差异与归属

缺口是账单当前待办无法多选查询，以及完全未开票可以新增核销。状态真相属于biz读取投影与data筛选；准入真相属于biz规则和data现有锁内事实。前端只消费服务端投影及候选，不在当前页过滤或重算状态。

## 契约与状态

在finance/v1/settlement.proto新增FinanceBillBusinessStatus枚举：UNSPECIFIED、UNINVOICED、PARTIALLY_INVOICED、UNVERIFIED、PARTIALLY_VERIFIED、COMPLETED；ListBillsRequest追加repeated business_statuses，FinanceBill追加读取投影business_status。不落库，不修改现有实体status、开票／结算进度字段。

biz新增纯Go状态类型与解析函数，列表／详情投影以已有进度、completed、有效verifiedAmount为输入：

| 条件（按优先级） | 当前待办 |
| --- | --- |
| 有效账单且completed | COMPLETED |
| 开票NONE | UNINVOICED |
| 开票PARTIAL | PARTIALLY_INVOICED |
| 票齐或NOT_REQUIRED，未结清且无实际核销 | UNVERIFIED |
| 票齐或NOT_REQUIRED，结算PARTIAL且有效核销额非0 | PARTIALLY_VERIFIED |
| 非有效账单或未经完整投影 | UNSPECIFIED |

零额有效账单沿用completed=true；有正负更正核销时用非零判断，不用大于零。data基于现有开票、结算、完成谓词组合SQL，并复用有效核销分摊合计判断，多个业务状态用OR后与既有条件AND。将条件加入统一predicates，使分页、完整匹配汇总、顶部指标共享。以真实PG矩阵对照biz投影集合，防止SQL与投影漂移。

service只负责枚举边界转换与返回；biz拒绝UNSPECIFIED／未知枚举。重复状态按集合语义，不造成重复记录。页面继续固定CONFIRMED；请求的业务状态不会扩大公司或实体状态范围。

## 核销准入

biz定义“开票进度为PARTIAL／FULL／NOT_REQUIRED才具备核销前置”的纯规则及领域错误。完全未开票返回明确中文冲突错误；不存在或跨公司账单仍先遵守已有授权与查无记录规则。

- 核销候选在Count和LIMIT之前，按有效发票分配或免票事实过滤；与剩余余额、当前公司等条件共同生效。可以通过FinanceBillFilter内部专用条件复用现有列表与聚合，不新增公共通用查询平台。
- verificationRepo.Create已按订单→资金→账单固定顺序加锁；在既有账单锁内读取当前有效发票额／免票决定，用共享规则复核全部目标账单，再写核销。沿用WithTx和现有错误外传，不新增锁层级、版本字段或事务抽象。
- 差额流水与核销已处同一共享事务，准入失败必须回滚差额流水；幂等回放继续沿用现有同键语义。
- executeFinanceBillBatchOperations已有先登记发票、再核销顺序，统一走最终仓储准入即可防止绕过；可以在组输入边界补充清晰预校验，不代替锁内复核。
- 建账表单根据本组有效登记行提示核销前置。用户删除发票行后保留核销输入，明确阻止提交并提示；不静默删除核销草稿或自动提交发票。无开票权限的用户仍可单独建账，但不能为无发票的新账单同次核销。
- 核销工作台消费服务端候选，保留过期候选提交后的输入与错误展示，不用客户端过滤改变候选总数。

## 前端与生成

经make -C server api及pnpm run generate:web-client更新Go绑定、OpenAPI、TS客户端与枚举／错误原因生成物。features/finance/bill-status提供最小业务状态元数据，账单页状态列与CSV只映射businessStatus。原billProgressStatus接口可保留为薄展示入口，删除前端金额／双进度重算。

账单SearchFilterTemplate新增businessStatuses字段，通过fieldProps.mode=multiple等现有透传实现多选；不改通用模板架构。沿用commitFilters、sanitizeRestored、onLoad采纳快照和actionRef刷新，保留其他独立进度筛选。

## 影响文件与必要性

- server/api/finance/v1/settlement.proto及生成物：新增类型、查询数组、读取投影；唯一契约来源。
- server/internal/biz/finance_bill_progress.go、finance_bill_types.go、finance_bill.go、finance_verification.go：状态解析、筛选校验及核销前置规则／领域错误。
- server/internal/data/finance_bill.go、finance_bill_credit.go或就近新增状态谓词文件：SQL条件、读取投影与统一聚合；finance_verification.go：候选与锁内准入。
- server/internal/service/settlement_bill.go及对应转换文件：请求／响应转换。
- web/src/features/finance/bill-status、web/src/pages/finance/bills：共享元数据、多选查询、页面／导出显示；web/src/features/finance/bill-creation：同次核销前置提示与提交校验；核销工作台按实际需要补候选说明／失效提示。
- 对应biz/service/真实PG集成、页面筛选／导出／建账／核销测试及确受影响的验收脚本：验证新规则而非仅镜像实现；修正正向夹具前置。
- .trellis/spec/web/frontend/finance-ledger-snapshots.md、server/backend/finance-bill-currency.md及涉及旧先核销规则的现行规范：记录用户新决定。

## 回滚与约束

无Schema或历史数据迁移；应用代码、Proto及生成物同组交付。回滚以整体Git提交为单位，不只回滚前端。既有实际账务保持，不增加自动回退／双读。先核销后开票的历史状态仍按当前事实显示，但不能新增违反新规则的核销。未知HGJ规则不作为本系统隐藏分支。
