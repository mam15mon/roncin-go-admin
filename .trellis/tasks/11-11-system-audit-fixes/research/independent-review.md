# 六项系统修复独立审阅

## 范围与方法

- 基线：`3933d94f`；审阅包含规划提交 `1ff7f110`、流水提交 `b37a5792`、机场提交 `5c99058a`、资源导入提交 `d7b8116c`，以及审阅时尚未提交的 R1/R3/R5 和流水迟到失败补充。
- reviewer 未参与实施，按父任务与六个子任务验收、check.jsonl 规范、真实调用链和定向复跑检查。
- 本次未修改产品代码、未提交 Git；唯一写入为本报告。主会话统一执行最终 `check:fast`、归档、合并和 push。

## 结论

未发现需要阻断合并的新产品缺陷。六项修复的核心边界与审定设计一致；有一处机械测试断言已交实现者修复，复跑通过。

### 已发现并修复的问题

- `web/src/pages/partners/components/secondary/SettlementRulesPanel.test.tsx:105`：Ant Design 空 InputNumber 的 DOM value 为 `''`，原断言 `toHaveValue(null)` 导致回归失败；实现者已改为 `toHaveValue('')`。独立重跑该文件 3 项通过，验证异常原币额度仍被清空、必填校验阻止保存、明确重录后按本币提交。

### 需要协调者完成的文档同步

- 父任务 PRD 的“范围与限制”仍有“不自动推送或部署”，与用户后续明确要求 push 冲突；应回填最新授权，保留不部署的范围。此项已通知主会话，不属于代码缺陷。
- 完成实现后回填各任务验收/执行状态，不能把规划占位勾选作为已验收证据。

## 行为核对

1. **R1 客商身份**：新 RPC 仅 `partner_id` 与最终 `role_types`；生成访问规则沿用当前公司编辑权限，通用 GetPartnerId 资源识别可用。Biz 验证空/非法/重复角色；data 在原事务与档案行锁下只替换角色、更新时间和审计，不触碰联系人/别名/Profile/责任人/散客/启停/结算规则；原黑名单角色不能被移除。
2. **R2 覆盖导入**：服务端基础 create、覆盖额外 update，伪造覆盖参数在仓储前拒绝；预览 overwriteAllowed 和前端按钮共用真实编辑资格，纯新建无需编辑权限。原匹配歧义/事务边界保留。
3. **R3 停用状态**：编辑已成功取得的 non-optional enabled 省略映射 false，创建分支仍默认 true；客商详情未注册启停控件，提交直接保留已读取的 partner.enabled，避免 true 被误停用；预设过滤与组织树投影一致。
4. **R4 流水重试**：规范化载荷同指纹，意图 Map 支持 A→B→A；未知结果关闭重开保留键，迟到成功/失败不修改新会话，成功采纳后新登记生成新键。保持页面实例内边界，没有声称刷新后持久防重。
5. **R5 信用额度**：额度与结算币种分离，独立/内嵌写入口都验证目标公司本币；聚合先校验所有有效正额度，不能仅校验最大值或跳过异常。旧错误配置仍可读取维护，前端显示原币金额、清空输入并要求人工明确重录，不静默换币。
6. **R6 机场筛选**：AE 使用最近关联主单地点，与详情投影一致，含终止历史关联；其他品类保留订单地点口径。总数与分页使用同一过滤，不批量重写旧投影。

## 独立执行的验证

- 修改前后端文档与代码 `git diff --check`：通过。
- 36 个修改的非生成 TypeScript 文件 `pnpm --dir web exec biome lint <files>`：通过。
- `pnpm --dir web tsc`：通过（测试断言修复后再次通过）。
- 关键前端 6 文件 24 项：首次 23 项通过、1 项上述机械断言失败；修复后独立复跑该失败文件 3 项通过，其余无需重复。
  - cashflow-save、creditLimit、SettlementRulesPanel、partner-detail-enabled、ResourceImportModal.permission、resource-enabled。
- 真实 PostgreSQL 正式迁移隔离 Schema：`TestPartnerCreditCurrencyPostgres`、`TestAirOrderLocationFiltersFollowSharedMasterPostgres` 通过。
- 真实 PostgreSQL `TestPartnerUpdateRolesPreservesAggregatePostgres` 单独重跑通过（5.06 秒），覆盖完整资料/启停/散客/规则保留、跨公司拒绝和黑名单失败回滚。输出 `/tmp/roncin-audit-review-partner-pg.log`。
- Biz/Service/Server 的 PartnerRoles、RequestPartner、导入预览/提交权限定向模式通过。
- 修复后结算表单重跑输出 `/tmp/roncin-audit-review-credit-ui.log`。

## 验证边界

本次独立 review 未跑全量门禁或浏览器生产环境验收；最终全栈门禁由主会话执行。未向真实用户发送消息、未导入真实经营数据、未清空开发库。资金流水防重保障限于当前页面挂载实例；旧外币信用配置需人工维护，不自动换算或改写。

## 最后增量复核：结算规则停用状态与嵌套入口

- 二级结算规则显式将省略 isActive 归为 false，并删除字段级 initialValue=true；主档成功读取规则时保持其 isActive，新建规则仍 true。成功路径实现正确。
- 独立复跑二级表单与主档保存 2 文件 13 项通过，输出 `/tmp/roncin-audit-review-is-active.log`。
- 最新信用真实 PostgreSQL 四个嵌套写入分支（直接创建/更新、导入创建/更新）拒绝异币额度且事务回滚，独立复跑 CNY/USD 两分支通过，输出 `/tmp/roncin-audit-review-credit-final-pg.log`。
- **发现待修 P1 相邻路径**：`partner-detail.tsx:610` 以 settlementRuleQuery.data 存在性决定 isActive，新请求失败/尚未成功时 data=undefined 与成功空规则混淆。已有停用规则加载失败后主档保存仍能提交 isActive=true，误启用规则。已通知主会话和实现者：编辑且要提交嵌套规则时须要求结算规则 query 成功，失败阻止写入并提示重试、保留输入；成功 null 才可默认新建 true。此项修复复查前，增量审阅尚未通过。

## 增量缺陷修复后复查结论

上述 P1 已修复并解除：

- 结算 query 显式验证真实响应，`undefined` 与 `success:false` 不再映射成功空规则；成功空集合 `null` 仍表示新建。
- 档案和结算规则分开做首次初始化。首次规则失败仍可保留名称等已输入资料，重试成功只回填结算字段，不覆盖普通档案输入；规则尚未就绪时不暴露可被迟到回填覆盖的结算字段。
- 编辑且要提交嵌套规则时要求 query 成功、未重读且结算初始化完成。首次失败、正在加载、有旧 data 的后台失败均不能落库；失败 Alert 提供官方 `refetch` 重试。
- 已有规则 `isActive` 省略后按 false 保留；非零本币额度在失败→重试后保留。
- 独立复跑 `partner-detail-save.test.tsx` 与 `partner-detail-enabled.test.tsx`：2 文件 13 项通过，包括首次 reject→undefined→成功停用非零→后台失败→重试→保存及名称输入/dirty 保留，输出 `/tmp/roncin-audit-review-retry-final.log`。
- 最新两个修改文件 Biome、`pnpm --dir web tsc`、`git diff --check` 再次通过。

增量最终结论：无未解决阻断项。主会话仍需完成最终全栈门禁及 Git 交付。
