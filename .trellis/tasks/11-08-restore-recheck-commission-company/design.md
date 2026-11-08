# 技术方案

## 最小修复与边界

根因是ledger读取内嵌调整时未预载Organization，转换器只能输出空名称。应在数据读取真相处修复，不用页面当前公司名兜底。

在finance_commission_ledger.go的ListScoped、ExportBatchScoped、GetByKey、GetScoped中，给已存在WithAdjustments查询补充WithOrganization（或复用已有相同查询投影，不为4处字段增加通用抽象）。commissionAdjustmentToBiz与service既有OrganizationName传递沿用；不改proto、数据库Schema或生成代码。

只检查同模块其余WithAdjustments调用是否对外输出调整明细，若仅用于金额能力计算不新增无用预载；不得借本期做聚合重构。

## 验证

- 实现后补真实PostgreSQL回归，创建合法提成/调整，断言父单与嵌套调整、独立调整读取的公司ID/名称一致；GetByKey/GetScoped/List/Export等实际输出入口保持一致，跨组织拒绝，不使用当前工作台名称替代。
- 前端通过实际页面动作回调打开确认/取消弹窗，断言公司来自调整记录且不是当前公司或兜底文案；保留现有编号、动作说明及expectedVersion/取消原因传递；现有弹窗未直接展示金额，本次不新增该交互，金额原值不变由真实库回归验证。避免mock掉待验证的弹窗构造。
- 开发环境只读复核SE2026110800002详情与单证当前正常；修复后读取CM2026110800003的ADJ001/ADJ002公司名称。两条记录保持取消，不执行财务写入。
- 开发阶段定向Vitest、修改文件Biome和必要tsc、专库Go定向；最终风险匹配前端/后端门禁，独立review通过后按用户授权交付。

## 风险与回滚

只读投影修复，金额及动作权限不变；新增预载避免N+1逐条查公司。代码回滚不涉及数据回滚。恢复问题本轮未复现的现场限制明确保留，不声称已经重复完成退关恢复验收。
