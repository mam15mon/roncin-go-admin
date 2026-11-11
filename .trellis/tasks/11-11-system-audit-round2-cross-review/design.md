# 六项修复设计

## 任务划分与边界

父审计 `11-11-system-audit-round2` 持续收集问题，本子任务直接交付 B1–B6 修复。采用一个工作分支、三个不重叠实现组，按问题分组提交；不再为本次已收敛的小修复制六套规划目录。主协调者维护计划、规范、集成验收与 Git 交付。实现者不得修改其他组文件或撤销已有工作。

## A 组：客商权限与默认抬头（B1、B2）

B1 保留现有聚合接口。`server/internal/biz/partner.go` 定义纯 Go 结算规则写能力（Create/Update，零值拒绝），经 `PartnerUsecase`、`PartnerRepo` 的 Create/Update/Import 显式传入。Service 使用现有 Principal 按目标公司解析对应独立权限，不把权限码或 Principal 泄漏为持久化字段。

`server/internal/data/partner.go` 将能力传至 `createPartnerChildren`、`updatePartnerInTx` 和 `savePartnerRoleSettlementRules`。在现有事务与规则锁内按真实数量决定动作：nil 内嵌字段跳过；零条检查 Create，一条检查 Update，多条保持原拒绝。权限不足返回已有 `ErrPermissionDenied` 并回滚整次主档/角色/规则/审计写入，不在锁外预判新增还是更新。

`server/internal/service/partner_convert.go` 的主档、角色、导出转换接收显式规则读可见性。`partner_profile.go` 按每个结果公司裁剪 Get/List/Create/Update/UpdateRoles/SetPartnerRoleBlacklist/Export，Import 只返回数量。裁剪只作用于 DTO，不修改领域对象；对应测试 stub 和直接仓储调用方一次同步，不留默认放行兼容入口。

前端 `partner-detail.tsx`、`SettlementSection.tsx`、`PartnerAnchorNav.tsx` 去掉结算权限里的通用 `canManagePartners` 旁路。现有档案根据规则查询成功后的存在性选择 Create/Update；无写权限省略内嵌规则，也不运行该分节写入校验；有读无写只读展示，无读不加载。失败/未完成不能视为不存在。`partner-secondary.tsx`、`SettlementRulesPanel` 与列表调用处同步传入明确读/建/改权限；只处理结算规则这一子资源。

B2 在 `server/internal/data/partner_invoice_profile.go` 两处清旧默认的批量更新追加 `AddVersion(1)`。沿用当前公司/主档/is_default 谓词、排除本次直接编辑项和父档案锁。直接编辑项仍由现有 `SetVersion(current.Version+1)` 递增一次，其他非默认行不变。

## B 组：提成净额与对冲候选（B3、B4）

B3 在 `finance_commission_adjustment.go` 现有 Order→父单→调整锁内，对取消 CONFIRMED 手工 INCREASE 执行取消后余额校验。复用冲减确认的订单行、父单两层计算口径，排除被取消调整，活动调整只含 CONFIRMED/PAID；任一有效金额为负返回已有超额业务错误。保持系统来源不可取消、DRAFT/CONFIRMED 状态门禁、调整 expectedVersion 和必填原因。可局部提取同文件计算助手，禁止扩成通用财务引擎。

B4 在 `finance_netting.go` LoadPreview 查询的 LIMIT 之前增加现有 `billUnsettledPredicate()`；保持公司、单位、币种、CONFIRMED 条件和 MaxFinanceNettingBills=500。余额计算、范围和有效分摊口径一致；没有新分页或自动截断。

## C 组：HAWB 保存基线与实际箱并发（B5、B6）

B5 在 `web/src/pages/orders/detail.tsx` 利用 UpdateOrder 的成功响应推进 HAWB 元数据。提交时捕获身份/刷新代际/Form 实例和提交意图，复用 `saveIdentityRef`、`explicitRefreshTokenRef` 拒绝串单或切公司迟到写回。只更新与本次已保存意图一致的 `airExport.houseStatus/houseVersion`，不整体替换共享主单/关联元数据，不 `resetTo` 覆盖当前编辑字段。服务端响应是提交后重读：若身份或版本显示已出现后续外部变化，保留输入并要求显式刷新，不给旧输入套新版本。原因仅在当前值仍等于提交时值时清理；失败不推进。原因框仍按已保存状态显示，已签发改回 DRAFT 仍需原因。

`web/src/components/ui/order-template/OrderFormTemplate.tsx` 的成功提交只在用户编辑 revision 未变化时清 dirty/草稿；`onValuesChange` 及主动 `applyValues` 记录 revision，纯元数据同步不算用户编辑。此为 B5 保存等待中新输入保护的直接邻接修复，不改变公共模板 API。

B6 `server/internal/data/order_container.go` Add/Update 使用 ForUpdate 返回的 `*ent.Order` 调用已有 `validateOrderSupportsContainers`；删除对旧快照的权威判断。Remove 不增加 FCL 门禁。`order_write_draft.go` 在目标 Order 加锁且版本检查通过后、业务写入前，目标类型非 FCL 则通过 `tx.OrderContainer` 按公司+订单检查存在性，有箱即返回原错误。清理仅服务于旧锁外检查的 `OrderRepo.HasContainers`、biz 调用、data 查询和 stub。保持已有锁序，不新增版本字段或改变父订单版本契约。

并发回归复用正式迁移 harness：专用连接变量、随机隔离 Schema、现有事务封装。先持有订单锁，让竞争请求真实等待，通过既有 pg_locks/pg_stat_activity 探针确认；在持锁事务内执行胜方真实操作再提交，分别覆盖新增箱先完成和切拼箱先完成。无需生产测试 hook、固定 sleep 或运行期自动建表。

## 依赖、风险与回退

A/B/C 三组产品文件不重叠，可并行。A 同组先统一方法签名再更新调用方；C 同组统一详情与模板输入保护。公共规范与生成协调只归主协调者；当前无生成工作。

最高风险是 B1 权限裁剪漏入口、B3 只守父单却遗漏订单行、B5 误覆盖新输入/接受后续版本、B6 只补竞态一侧。验收分别针对这些失效方式。所有数据库验证使用专用测试环境及隔离 Schema，不操作业务数据。回退以分组提交 revert 为单位，不强制重置分支或修复数据。
