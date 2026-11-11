# Research: 六类审计修复的关键设计复核

- Query: 复核专用业务身份写入、本币信用额度约束、资金登记重试的最小设计与遗漏入口。
- Scope: internal
- Date: 2026-11-11

## Findings

### 一、业务身份独立写入：方案成立

新增 `UpdatePartnerRoles` RPC、用例与仓储方法比回传整份旧档案更直接，能在根因处隔离写集。新请求只接收身份类型/启用值（或明确的最终角色集合），不接收档案、联系人、散客、结算规则等字段；可复用现有 `business.partner.update`，无需新增权限码。仓储在 `WithTx` 内按本公司 + partner ID 锁定档案，读取当前角色后执行 `replacePartnerRoles`，审计与角色更新同事务，提交后重读。

关键代码：
- `server/internal/biz/partner.go:239` 定义 `PartnerRepo`；增加方法要同步真实实现和 `partner_test.go` 的接口断言 stub。`normalizePartnerRoles` 在同文件 `:581`，其中“启用档案至少一个启用角色”依赖当前档案 Enabled，不能在用例里硬编码为 true；建议结构/去重检查与锁内当前状态检查分清。
- `server/internal/data/partner.go:694` 的 `replacePartnerRoles` 已保留角色记录，只停用被移除身份，且阻止移除黑名单角色；复用它可保留结算规则和黑名单事实。新路径不应调用 `savePartnerRoleSettlementRules`，也不能接收嵌套 SettlementRule 借此越权修改财务规则。
- `server/internal/server/auth.go:91-112` 的 `requestPartner` 按请求类型识别资源：新请求若字段叫 `partner_id` 会被已有 `GetPartnerId` 接口识别；若叫 `id` 必须新增显式类型分支，否则丢失直接资源鉴权识别。仓储本公司谓词仍独立保留。
- `server/api/partner/v1/partner.proto:40` 为现有 UpdatePartner 声明；新 RPC 需生成 pb/grpc/http/OpenAPI、服务端访问规则，以及前端 `partnerService.ts`、typings、OpenAPI 输入；依根流程 `make -C server api` → `pnpm run generate:web-client`。不手改生成文件。
- `web/src/pages/partners/components/RoleSwitchModal.tsx:67` 改用专用客户端，`index.tsx:445` 是列表入口；保持写响应验证、失败留输入和成功 reload。

实施后回归：同一档案联系人/别名/Profile/各责任人/创建人/isCasual/enabled 完整保留；角色变更生效；已有结算规则、黑名单保留；非法/重复/空角色，黑名单移除、外公司 ID、无权限均拒绝；失败整体回滚。

### 二、信用额度按公司本币：方案符合既有产品决定，但要覆盖全部写入口

既有 `.trellis/tasks/archive/2026-09/09-11-partner-terms-credit/prd.md:40-44` 明确额度为“主档录入的本币信用额度”、消费已有折本币余额；同文档非目标明确不做查询时实时换汇。因此保持该决定，不为本次修复加入周汇率查询或重标旧金额。

容易遗漏：
- `web/src/pages/partners/components/secondary/SettlementRulesPanel.tsx:389-399` 将 creditCurrency 绑定 settlementCurrency；必须分离为当前公司 baseCurrency，展示明确本币单位。
- `web/src/pages/partners/partner-detail.tsx:579-604` 的完整建档/编辑内嵌结算规则也有同样绑定，且未填额度时发送 0；同步修正，不能只修二级面板。`components/SettlementSection.tsx:232` 标签本来已写“信用额度(本币)”。
- `server/internal/data/partner_settlement_rule.go:61-117` 独立 Create/Update；以及 `server/internal/data/partner.go:726-760` 的 `savePartnerRoleSettlementRules`（完整建档、编辑、导入共同消费）都需验证额度币种等于目标公司本币。现有 `validateSettlementCurrencies(:200)` 只检查币种存在启用，不能替代公司一致性校验；尽量共用一次业务校验，避免只堵一条写入路径。
- `server/cmd/sync-dev/main.go:670-679` 直接建 Ent 结算规则：当前写入正额度却没有 SetCreditCurrency。若聚合新增严格校验，该开发种子会成为非法数据源；生成开发数据必须同步填写已确认的公司本币。不要为修复旧数据静默改写已有规则，既有外币或缺币种规则应明确报错并由用户重录。
- `server/internal/data/finance_bill_credit.go:80-109` 读取激活客户规则后直接聚合最大额度；应在比较最大值前校验每条有效额度的币种，不可仅校验最终最大那一条。币种缺失/不同不得解释为本币，也不能跳过规则导致放行。公司本币读取复用已有组织/汇率上下文入口，避免 CNY 硬编码；通过 Data.client(ctx) 参与事务。
- `server/internal/biz/partner_settlement_rule_test.go:75-85` 明确把 USD 额度+CNY 结算视作合法，纯 normalize 本身可继续允许两个合法独立币种，但仓储集成必须基于目标公司本币验证，不能机械地要求 creditCurrency==settlementCurrency。

实施后回归：公司本币 CNY / USD，结算币种与本币可不同；同币额度边界 0/NULL/相等/超额；外币/缺币种额度明确拒绝且不部分写；非法旧配置读取消费明确失败；只有提醒与直接干预都保留既定行为。修复资料入口应可读取原有非法规则以人工重录，不能被聚合错误堵死。聚合错误会使同批候选查询失败，需错误文本指明需要修正信用额度配置，不能吞为空候选。

### 三、流水稳定意图键：方案成立，注意 A→B→A 与异步回调

- `web/src/pages/finance/cashflows/index.tsx:530-543` 每次 onFinish 新建 key 是根因，后端 `server/internal/biz/finance_cashflow.go:141-150` 已按公司+key 回放，并在解析实时汇率前优先重放。无需增加银行号唯一索引或猜测业务去重。
- 后端 `finance_cashflow.go:123-131` 规范化币种大写、日期/账户/付款方式去空白、可选字符串空白折叠；`:199` 的 `sameFinanceCashflowIntent` 用 decimal 数值等价比较金额和显式汇率。前端 fingerprint 应以同一次规范化请求快照构造，排除 key；金额 100 与100.00、空可选值与未设置、账户首尾空白不能人为制造不同意图。日期使用提交的 YYYY-MM-DD。
- 仅保存“最后一次 fingerprint/key”会遗漏 A 请求结果未知→改成 B→又改回 A：A 会得到新 key，仍可能重复创建。可在当前登记会话内缓存已发送的 fingerprint→key（或在结果未知时冻结原提交并要求先确认）；无需上升为全局通用幂等平台。
- 成功确认后清除本次会话记录；新的明确登记使用新 key。不要只按 `open=false` 无条件重置未知请求键：用户关闭再打开同一未确认输入会失去防重。最小修复若只保障当前打开会话，应把范围/限制写明；更稳妥是在未知结果期间保留意图直到确认或明确新登记。
- 已有 `web/src/pages/orders/use-sea-order-create.ts:39-90` 采用先固化载荷再请求、未知结果同键确认；`use-sea-order-save.ts:53-68` 也是冻结同一载荷。可借鉴行为，不必抽出跨订单/财务的大型 Hook。
- 写响应 `undefined`/缺 ID/失败必须保留 key；仅确认成功才清除。在途旧响应成功不能清掉新会话的 key 或关闭新表单；工作区已有生命周期机制应继续复用。

实施后回归：第一请求实际已提交但响应丢失→同意图重试保持同 key；格式等价输入同 key；实质修改换 key；A→B→A；成功后新登记新 key；失败输入保留；关闭/重开与工作区切换边界；并发点击由既有提交 loading/防重保护约束。

## Related Specs

- `.trellis/spec/server/backend/database-guidelines.md`：统一事务、ForUpdate、Data.client。
- `.trellis/spec/server/backend/partner-casual-contract.md`：散客标识只由明确维护改变。
- `.trellis/spec/server/backend/exchange-rate-single-rate.md`：公司本币来源与手工汇率权限。
- `.trellis/spec/web/frontend/type-safety.md`：ProtoJSON 零值及 optional 存在性。
- `.trellis/spec/web/frontend/state-management.md`：写响应验证和工作区隔离。

## External References

未使用外部资料；本次结论来自本仓库现有契约与代码，不涉及外部版本事实。

## Caveats / Not Found

本次为规划设计复核，未修改产品代码、未执行数据库或复现测试。行号以 2026-11-11 主仓库当前版本为准。信用额度非法现存数据不能未经授权重标或重置；应明确报错、允许人工维护。父任务不应把此研究结果写成修复已完成证据。
