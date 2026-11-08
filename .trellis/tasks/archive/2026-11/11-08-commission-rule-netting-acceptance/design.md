# 技术方案

## 修复边界与根因
方案表单使用真实 ProFormDateRangePicker，BaseForm 默认 dateFormatter=string；RuleValues及提交回调却约定 Dayjs。完整日期触发请求前异常，缺日期则可到达后端泛化校验。这是表单提交契约错配，当前没有引擎缺陷证据。

## 前端最小修复
- 在 CommissionRuleFormModal 显式 dateFormatter=false，与现有 Dayjs 类型和 .format 回调一致；不加 string/Dayjs 双路兼容，不改全站 GuardedModalForm。
- 对新建启用方案增加明确日期校验与帮助文案，校验随启用状态变化；不误阻断已生效方案按既有约定修改名称/备注。
- 复用现有合法写响应守卫、onFinish Promise 生命周期、成功提示和 actionRef.reload。若载荷转换仍可能抛异常，确保错误在当前表单可见，不制造第二套 loading 状态。
- 新增/调整真实表单回归，仅 mock 请求与必要外围；断言实际请求日期、比例、员工/公司及成功/失败反馈。复制与名单日期路径没有同类缺陷，不为本次修改。

## 后端与验收
- 现有 API 与数据库结构预计无需变化；补充通过业务 CreateRule 正式创建方案和员工分配的对冲集成回归，而非直接 Ent 塞规则。
- 金额采用现有全量订单经济投影及来源清偿贡献；不能把对冲应付分摊当作订单全部成本。独立正例 R=300、P=200、来源收入200，按8位小数逐票计算；前端只展示2位。
- 负毛利按当前合同将基数归零，仍可有候选、预览、零额草稿与确认；这不是正毛利验收的替代。
- 实际开发环境由合法界面/API完成方案、独立业务数据、候选、预览、草稿和确认，SQL只读交叉核对；不通过SQL改金额、状态、归属或方案日期。
- 新方案名 CODEX测试方案1108；启用区间按验收当天至当月末设置并核对来源日命中。执行前重查重名与重叠，发现既有验收结果先核验，不盲目重复创建。
- 当前来源日取 confirmed_at UTC日期，新建方案不可追溯以Asia/Shanghai业务日校验；本次研究时两者同日。不得为造候选静默改日期契约，执行时若出现不一致须据实分析。
- 不运行带发放动作的整套既有财务验收脚本；仅复用其请求形状和必要辅助能力。

## 预期文件与质量边界
- 主要实现：web/src/pages/finance/commissions/components/CommissionRuleFormModal.tsx，必要的CommissionRulesDrawer.tsx及同目录测试。
- 真实库回归：server/internal/data/finance_commission_netting_integration_test.go 或独立专属集成文件，复用现有夹具、正式迁移和事务约束。
- 实际验收采用最小可复用脚本或已有浏览器工具；不得记录密码、Cookie、令牌或连接串。
- 代码回滚使用本组提交回退；开发验收数据保留明示，取消/反转等后续操作须按业务动作显式办理，不自动清库或抹除历史。

## 实际验收发现的最小补修

方案和候选成功后，生成草稿 HTTP500：FinanceCommissionLine.customer_code 的 Ent NotEmpty 校验与 Partner.code 选填契约冲突。删除该字段 NotEmpty 并按生成流程更新 runtime 校验，保留非空字符串列承载空字符串、长度上限及身份快照，不伪造编码。此变更不修改 SQL 列形、默认值或数据库约束，现有正式迁移的 varchar(64) NOT NULL 接受空字符串，因此无 DDL 迁移。补无客商编码的真实草稿生成及确认回归。
