# 研究：提成方案创建表单与日期提交

- 问题：完整填写融迅北京、业务人员、已实现毛利、10%、张冠楠与有效日期后，是否存在可证实的前端提交或成功反馈缺陷；同模块复制、名单路径是否有同类问题。
- 范围：仓库代码及已安装依赖，只读研究；未查询或改写业务数据库，未改业务代码或测试，未执行 Git 操作。
- 日期：2026-11-08。

## 结论

确定存在前端日期类型错配，可解释「补全日期后点击提交但无明确创建结果」。这与缺少方案导致空计提候选是两个先后问题，当前证据不能据此认定提成计算引擎有缺陷。

## 文件与调用链

| 文件 | 作用与证据 |
| --- | --- |
| `web/src/pages/finance/commissions/components/CommissionRuleFormModal.tsx:57` | 新建/编辑共用 GuardedModalForm，没有声明 dateFormatter=false |
| `web/src/pages/finance/commissions/components/CommissionRuleFormModal.tsx:154` | 生效区间使用 ProFormDateRangePicker，无必填/启用联动校验 |
| `web/src/pages/finance/commissions/components/CommissionRulesDrawer.tsx:183` | onFinish；第189–190行直接对日期元素调用 .format，rule对象在try之前构造 |
| `web/src/pages/finance/commissions/types.ts:14` | RuleValues 把 effectiveRange 声明为 Dayjs 元组，声明未反映 ProForm 默认提交转换 |
| `web/src/components/ui/GuardedModalForm.tsx:34` | 透传官方提交值并 await 业务 onFinish，不改变日期类型 |
| `web/src/services/roncin/settlementService.ts:574` | 生成客户端 POST /api/v1/finance/commission-rules，原样传递 body |
| `server/internal/service/settlement_commission.go:184` | 解析方案、组织及 configure 权限后调用 CreateRule |
| `server/internal/service/settlement_commission.go:344` | decimal比例、字符串人员身份/口径、日期指针及员工UUID转换 |
| `server/internal/biz/finance_commission_rules.go:220` | 新建启用方案不允许起始日早于业务日期 |
| `server/internal/biz/finance_commission_rules.go:232` | 校验日期格式、先后、比例0<rate<=100及启用时起始日必填 |

## 主要缺陷的依赖证据

实际安装的是 `@ant-design/pro-components` 3.1.14-6（仓库带本地补丁的版本），子组件已合并在该包的 form/utils 目录，不是独立的 @ant-design/pro-form 包。

- `web/node_modules/@ant-design/pro-components/es/form/BaseForm/BaseForm.js:234`：默认 `dateFormatter = 'string'`。
- 同文件第356行：提交前调用 conversionMomentValue；第374–375行把 getFieldsFormatValue 的结果传给 onFinish。
- `.../es/form/components/DateRangePicker/index.js:4`：该字段 valueType 为 dateRange。
- `.../es/utils/conversionMomentValue/index.js:97` 及第145–175行：浏览器中按 valueType 把 Dayjs 转为日期字符串。
- BaseForm 第395–397行的最终 catch 只复位 loading，未将业务回调外的异常显示成页面消息。

### 无业务写入的复现实验

Node 进程中设置 `global.window = {}`，加载当前包的 `lib/utils/conversionMomentValue`，按真实 dateRange 字段映射转换 `[dayjs('2026-11-08'), dayjs('2026-12-31')]`，再执行当前提交表达式。

```text
dateFormatter='string'
effectiveRange = ['2026-11-08', '2026-12-31']
typeof effectiveRange[0] = 'string'
effectiveRange?.[0]?.format('YYYY-MM-DD')
→ format is not a function

dateFormatter=false
typeof effectiveRange[0] = 'object'
effectiveRange?.[0]?.format('YYYY-MM-DD')
→ 2026-11-08
```

需设置 window 是因为工具函数在 SSR/纯 Node 无 window 时直接返回原值（工具函数第120行）；不设置会漏掉真实浏览器转换，产生错误的「无法复现」结论。实验未创建临时文件，未调用请求客户端。

由于 rule 在业务 try 之外构造，完整日期场景会在 `settlementServiceCreateCommissionRule` 调用之前拒绝 Promise；既不会进入本地 handleRuleError，也不会显示「提成方案已创建」或执行列表 reload。这与用户第二次提交后的观察一致，但未以用户当时的浏览器日志证明其历史点击一定走到此处。

## 首次泛化错误与成功反馈

- 不选生效区间时 optional chaining 令 effectiveFrom/To 为 undefined，不触发 .format 异常；当前表单没有必填校验，会请求后端。
- 启用方案缺起始日，或起始日早于 Asia/Shanghai 服务端业务日期，后端按现有规则返回 FINANCE_COMMISSION_RULE_INVALID（「提成规则字段不合法」）。因此第一次错误可能确为输入导致；前端缺少对应校验和明确引导是实际交互缺口。
- 「10% / SALES / REALIZED_PROFIT」分别经字符串比例与现有角色/口径转换，属于允许值；没有证据表明这些指定输入本身非法。
- `CommissionRulesDrawer.tsx:222–227` 已有有效写响应守卫、成功消息、关闭弹窗与 actionRef.reload；请求失败被全局消费为 undefined 时返回 false，不会宣称成功。当前核心断点发生在请求之前，不能归因于已有成功守卫缺失。

## 相邻日期路径扫描

| 路径 | 当前处理 | 本期日期错配需要同步修改吗 |
| --- | --- | --- |
| 新建/编辑方案 | RuleValues 为 Dayjs，但 ProForm 默认提交字符串；直接 .format | 必须修正共用 FormModal |
| 复制为新方案 | CopyValues 明确为字符串；Drawer第236–259行使用 dayjs(value).format；起始日有必填/禁选过去 | 无同类异常，不需要为此改写 |
| 名单加入/移除 | RosterModal第164–165行是原生 antd Form；DatePicker提交Dayjs，190行 .format 合法；未选时按当天生效 | 无 ProForm 字符串转换，不需要为此改写 |
| 提成台账日期筛选/导出 | 日期已由独立转换函数处理 | 未发现与本次完整区间提交相同断点，不扩大范围 |

### 相邻但独立的边界

1. 公司切换只清空员工候选（FormModal第99–106行），没有清除 Form 中已选 employeeIds；从A切到B后可能仍提交A员工，后端会拒绝无效成员。这不解释用户一直选择北京的案例，可作为同表单的定向边界修复，不能放宽后端成员校验。
2. 后端允许「必填起始日 + 空终止日」长期有效；FormModal第72–75行只有起止都存在才初始化区间，导致编辑长期方案时丢失起始日。RangePicker当前未声明 allowEmpty，表单类型也要求双日期。若本期要覆盖正常长期方案，应显式表达可选终点并保留既有起始日；若验收只用完整有限区间，此项需明确记录而非顺便扩大产品范围。
3. 方案配置权限与公司候选入口存在相邻不一致：页面把 canConfigureFinanceCommissions 传给 Drawer的 canManage（index第835–838行），但Drawer第79–82行公司候选 purpose仍是COMMISSION_MANAGE；服务端 settlement.go第113–114行映射经营 manage 权限，而创建接口使用 configure。仅配置权限用户可能拿不到公司选项；用户本次已能选北京，不能用此解释日期提交异常，建议单独评估权限范围。

## 日期与来源约束

- 方案匹配以「来源日期 ∩ 方案区间 ∩ 员工分配区间」及订单已固化人员身份共同判定。新建启用方案不得追溯，不应为补验收放开起始日早于服务端业务日期的规则。
- `server/internal/data/finance_commission_calculation.go:154–157` 与第260–264行：对冲必须为 CONFIRMED、有 confirmed_at 且不是更正专用对冲，当前来源日期取 confirmed_at.UTC().Format('2006-01-02')。
- 主会话已发现原对冲现为 REVERSED；本研究未查数据库，也未创建替代来源。应由主会话确认有效的新验收来源，不应修改旧对冲状态或回填人员归属以制造候选。

## 最小修复边界与建议回归

1. 优先让新建/编辑 FormModal 显式 `dateFormatter={false}`，与现有 Dayjs 类型及 .format 回调一致；这是修正真实提交契约，不需要添加 string/Dayjs兼容分支或全站转换平台。
2. 让启用方案缺起始日、过去起始日及区间倒置在表单层明确报错；终点可选语义仍以已批准的本期范围及后端合同为准，不能把后端校验删掉。
3. 业务载荷转换及 onFinish 应保持可见错误反馈和官方 Promise 生命周期；不要用手工 loading/关闭状态镜像替换。
4. 主要代码预计为 `CommissionRuleFormModal.tsx`、必要的 `CommissionRulesDrawer.tsx`/RuleValues，以及真实表单回归；复制/名单不用为本日期缺陷修改。
5. 回归必须保留真实 GuardedModalForm、ProForm 日期转换和提交事件，仅 mock 请求；真实新建/编辑提交有限区间，断言 effectiveFrom/To 为 YYYY-MM-DD、比例为'10'、组织及 employeeIds 正确、只请求一次、成功消息和列表 reload；缺起始日不请求、失败不关闭。
6. 按批准范围补公司切换清员工、长期方案编辑保留起始日；另以正常复制、名单操作的轻量回归确认未被误改。

## 现有测试为何未发现

`CommissionRulesDrawer.test.tsx:28–47` 把 ModalForm 与全部日期组件替身化；第132–151行直接调用 onFinish，而且新建用例不传 effectiveRange。它既绕过真实 ProForm 默认转换，也不触发日期 .format，无法发现当前断点。复制用例手传 Dayjs也不能证明真实新建表单日期合同正确。

## 相关规范与参考

- `.trellis/spec/web/frontend/quality-guidelines.md:157–172` 已记录同类合同表单问题：需要 Dayjs方法时设置 dateFormatter=false，回归保留真实表单转换。合同接口要求RFC3339，而提成方案接口要求YYYY-MM-DD，不能照搬合同载荷格式。
- `.trellis/spec/web/frontend/state-management.md`：写响应校验、失败保持弹窗与输入、官方表单提交生命周期。
- `.trellis/spec/server/backend/commission-scheme-assignment.md`：方案/分配有效期、不可追溯与来源日期资格。
- `.agents/skills/roncin-web-stack/SKILL.md`、`/home/vince/.agents/skills/antd/SKILL.md`：本地前端约定与组件查证方法。
- 第三方证据来自仓库实际安装并带补丁的 ProComponents 源码，无外部网页推断或未核实版本推荐。

## 限制与未证明项

未验证实际数据库中的方案、归属、来源金额；这些由主会话负责。没有创建、计算、确认或发放任何提成；目前不能给出引擎金额正确性结论。首次错误的具体输入与第二次点击当时的浏览器异常无日志，历史原因仍需区分「代码必现路径」和「该次操作的直接证据」。
