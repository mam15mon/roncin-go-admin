# 业务标签全站退役

## 1. 适用范围
2026-11-09 取消企业资源TAG及其与订单、订单费用、财务账单的人工标签关联。不得恢复成隐藏入口、无消费者API或历史兼容层。普通状态/进度/异常、人员客商候选、汇差视觉标记和页签不是本能力。

## 2. 退役签名与编号
以下18个RPC不再注册，OpenAPI及生成客户端不再声明：
- OrderTagService：ListOrderTagOptions、BatchAssignOrderTags、BatchRemoveOrderTags。
- OrderFeeService：ListOrderFeeTagOptions、BatchAssignOrderFeeTags、BatchRemoveOrderFeeTags。
- SettlementService：ListFinanceBillTagOptions、ListFinanceBillTagAssignmentOptions、BatchAssignFinanceBillTags、BatchRemoveFinanceBillTags、ListFinanceFeeTagOptions、ListFinanceFeeTagAssignmentOptions、BatchAssignFinanceFeeTags、BatchRemoveFinanceFeeTags。
- EnterpriseResourceService：ListEnterpriseTagGroups、CreateEnterpriseTagGroup、UpdateEnterpriseTagGroup、DeleteEnterpriseTagGroup。

仍存在的消息移除并reserved原编号及名称：Order.tags=79、ListOrdersRequest.tag_ids=35、OrderFee.tags=36、ListFeeLedgerRequest.tag_ids=15、FeeLedgerItem.tags=34、ListBillsRequest.tag_ids=10、FinanceBill.tags=36、EnterpriseResource.detail.tag=13、EnterpriseResourceInput.detail.tag=12。资源枚举TAG=4名称与编号reserved，5..10不重排。仅标签使用的概要/请求响应类型删除。

## 3. 跨层契约与数据边界
- 删除业务标签三层实现、附加读取和过滤、批量写入、DI/Provider/Wire与HTTP/gRPC接线。删除拆票中的标签复制/转移，不改变费用守恒、订单复制或锁和事务。
- Ent不再定义EnterpriseTagGroup、EnterpriseTag、OrderEnterpriseTag、OrderFeeEnterpriseTag、FinanceBillEnterpriseTag和各owner的标签edges；通用企业资源及其余明细/关联保留。
- 新正式迁移先检查TAG没有异常非标签明细/人员/地址类型引用，冗余类型关联与真实资源一致；未知依赖或异常数据显式失败，不做静默纠错。
- 正式退役迁移为 `server/migrations/20261109100000_retire_business_tags.sql`，历史迁移保留原样。
- 按依赖顺序移除三张业务标签关联表；按TAG资源ID清理客商关联；移除enterprise_tags及enterprise_tag_groups；最后精确删除enterprise_resources中TAG行。不DROP共享表，不用无界CASCADE，不回写历史SQL，不重置数据库。
- 只删除独立权限system.finance.fee.tag，保留其他订单/费用/账单/企业资源权限。复用cmd/migrate的SyncPermissionManifest删除旧权限和授权，不另造同步逻辑。
- 历史audit_logs不删不改；现有历史事件静态中文说明可保留以展示历史事实，不提供标签运行接口。历史迁移与Proto reserved也不是需要清零的功能残留。
- 生成器按当前Proto/Ent/Manifest重新生成；没有源文件的旧专属生成文件精确清除，不手改生成内容，不整目录删除含Proto源的api路径。

## 4. 前端范围与工作区
订单列表/搜索/公共模板、订单费用、费用台账、账单均不再出现业务标签列/过滤/批量操作/CSV字段/列偏好；企业资源不再有标签页或标签组及快捷新建；其他资源数字身份保留，已完成的类型隔离和原生表单校验流程保持。
账单列表去公司和标签整条topBar，列表organizationId只来自auth/me当前公司工作区，不从查询恢复状态选择其他公司；身份未就绪不发送无组织过滤的查询。组织切换复用已有workspace边界。后续财务其他列表及建账抽屉等已统一采用[当前公司工作区](../../web/frontend/finance-current-workspace.md)，不得据标签退役当时的范围恢复独立公司选择。

## 5. 验证与错误矩阵
| 场景 | 必须结果 |
| --- | --- |
| 有标签及三类关联的正式前态升级 | 五张专表与TAG元数据/客商关联清理，核心单据、非TAG资源和审计关键值保留 |
| TAG异常引用非标签明细或关系类型漂移 | 迁移失败并回滚，不CASCADE掩盖问题 |
| 冷启动重放全迁移 | 最新结构无标签实体，其他资源和权限正常 |
| 旧角色持有fee.tag | 正式迁移后权限同步去该授权，其他角色/权限不变 |
| 旧标签RPC/字段 | 当前描述符与真实注册不再有该能力，不提供兼容空接口 |
| 原TAG=4与类型5..10 | 4不能复用为其他类型；5..10既有含义不变 |
| 当前公司账单列表 | 无公司selector，恢复条件不能把请求切到其他公司；未就绪不无界读取 |
| 地址/备注/图片/单证主体维护 | 保留CRUD、导入、候选及类型切换隔离和正常表单校验 |

## 6. 必需测试
真实隔离PostgreSQL按正式前缀迁移建立含标签/非标签/审计的混合夹具；验证精确删除、异常回滚、权限同步、冷启动和结构末态，不把SKIP算PASS。费用删除与批量删除仍验证历史账单来源及金额快照；拆票仍验证费用迁移/守恒。前端验证无标签入口/载荷/候选、无标签CSV、普通状态标记保留，账单当前公司范围不越界。真实RPC注册与Proto reserved断言有效；生成重跑幂等及check:fast通过。

## 7. 错误与正确
```text
错误：隐藏标签筛选，保留业务标签API、权限、查询附加和存储模型。
正确：精确删除纵向能力和生成物，使用正式退役迁移，验证相邻业务不变。

错误：对Tag字符串全仓替换，删除状态标记、TagsView、历史迁移或审计。
正确：按明确业务标签类型和调用链删除，历史事实及其他视觉能力保留。

错误：去掉公司selector后把organizationId留空，扩大账单读取。
正确：列表只使用已就绪的当前工作区身份，组织范围仍由后端最终校验。
```
