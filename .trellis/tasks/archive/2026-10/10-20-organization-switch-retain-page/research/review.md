# 独立审查记录

工作区：`/tmp/roncin-org-switch-20261020`。审查直接读取本工作区 PRD、design、implement、check.jsonl 所列规范与 forms-audit / guards-implementation / domain-forms-coverage，未使用其他会话的任务。全程不修改原 `/home/vince/roncin-go-admin`，不提交或合并 Git。

## 已修问题

1. **订单费用行内编辑未受保护**。`OrderFeeTableTabs.tsx` 原接入只覆盖三个批量编辑弹窗；真实 EditableProTable 数量、单价、备注等用户修改没有 guard。审查补应收/应付各自独立 guard；本地框架源码确认 editable.onValuesChange 内部防抖 64ms，因此在公开 editable.formProps.onFieldsChange 中仅对 touched 用户字段同步登记，避免用户紧接着切组织漏保护。单纯进入编辑与默认值不脏，取消或成功保存只清对应一侧；失败保持。新增真实双侧行内编辑回归覆盖输入、切换取消、交叉取消、保存失败与成功。
2. **四个台账原因确认缺少所属组件 signal**。`finance/{bills,cashflows,nettings,verifications}/index.tsx` 的 makeVersionActions.confirm 会打开可编辑原因框，原来只有提成页传 getter。嵌套 App 卸载虽可销毁 UI，但本地 antd destroy 不结算 thenable，未传 signal 时 confirmWithReason 手工注册的 guard 可残留。审查补 useModalOwnerSignal + getter 参数；全生产六个 makeVersionActions 实例均已核对。新增真实 StrictMode + makeVersionActions 事件打开、输入、所属组件卸载的回归，守卫清理且旧实体不提交。
3. **完整门禁暴露三处旧测试替身未同步**。`partners/index.test.tsx`、`partners/partner-detail.test.tsx`、`parameter-setting-template/ParameterSettingTemplate.test.tsx` 只提供 history.push/replace，缺少真实公共契约 history.location，新 guard 在渲染时读取 pathname 导致 18 项失败。仅补测试 location，产品无测试环境回退、不跳过用例。修复后三文件 21 项全部通过。
4. **补浏览器 POP 的实际导航回归**。原真实路由测试覆盖另一条导航取消，审查另补 router.navigate(-1)：目标 lazy 未结束时后退，转安全工作台并结束切换，lazy 迟到不覆盖；请求轨迹始终没有新身份读取旧订单。
5. **主会话协同补齐普通 App.modal 所有权与漏扫入口**。审查确认根 AntdApp 位于 Router 之上，普通业务确认不会因页面卸载自动关闭；主会话在 OrganizationWorkspace 内嵌 App component=false，使普通业务框随 workspace 销毁，根反馈桥/头部切换确认仍保留根 App；新增真实边界测试已读代码与断言。主会话并行审计追加 BusinessTagModal、企业资源批量编辑与导入/汇率同步入口。审查读取实际源代码和真实输入断言；有效导入载荷才标脏、无效文件不误提示、删除文件同步恢复默认导入模式，均已补齐。

6. **追加非 Form 输入保护**。团队共同补齐企业资源三个批量编辑 state、汇率三类手工报价/精度/套用/还原、汇率及客商 Excel 和资源 CSV 导入、共享箱件重尺分配。独立审查要求把导入标脏从选择文件延后到有效 payload，且删除客商文件时恢复导入模式，避免清 guard 却残留模式输入。共享箱保存/确认失败原强制 reload 会覆盖用户分配，已改为失败保留，成功才清 guard 并重查。实际输入的 5 文件 24 项补漏定向已由领域代理交接，源代码和断言均复核。
7. **订单表单程序赋值旁路**。主会话把 OrderContainerRequestFields 箱型/数量/增删、SeaTransportSection 人工选择既有共享 MBL 航次、主动清空散杂货箱量，接回标准 Form.Item value/onChange；直接程序 setFieldValue 不再绕过原 OrderFormTemplate 的 dirty/隔离草稿。审查核对实际事件链及新增两个真实模板组合回归（初始化不脏、编辑落草稿、失败保留、resetTo 清理）；默认自动匹配仍是程序初始化，不标脏。ShippingDocumentFields 无生产消费者，明确排除。

## 未修问题与存量限制

- `components/ui/master-data-template/types.ts:107` 存量 Biome warning：Promise<unknown> | void 的 noConfusingVoidType；7 项 info 来自存量 userColumns.test.tsx 的 fragment 和 CommissionRulesDrawer.test.tsx 的 literal key。已以 git show HEAD 核实原来即存在，非本任务逻辑，不做范围外清理，门禁退出码不受影响。
- 既有 `orders/templates/sea-template.test.tsx:35` describe.skip 覆盖 12 项，HEAD 已有注释说明海运模板仍快速迭代；本轮未新增 skip、only、类型绕过或规则关闭。
- 前两轮完整测试输出存在 localhost:3000 ECONNREFUSED（第二轮 35 个 AggregateError），没有归属到具体失败用例，未将其确认为本次新增或既有问题；不为追日志加入全局拦截、修改配置或静音。不能把退出码通过描述为测试日志零噪声；最终轮实际为 30 个 ECONNREFUSED AggregateError 与 5 个 socket hang up / ECONNRESET。没有失败用例、未捕获异常或 act/弃用/antd Warning；这些连接日志仍是明确的验证限制，尚未定位来源。
- 本次无服务端、API、Schema、迁移或生成物改动，未运行后端/数据库测试，也未运行生产构建；按任务前端风险与仓库门禁阶段约定执行。

## 实际验证与代码时序

- 实施代理定向结果作为背景，审查没有将其当作完整门禁。审查亲自重跑真实 RR 导航补 POP 后单文件 7 项通过（4.07 秒），真实行内费用单文件 1 项通过（7.87 秒，其中测试执行 3.07 秒）。相邻定向 4 文件 22 项通过（27.86 秒）；本轮修改文件 Biome 与 git diff --check 通过。
- **第一轮 check:web**：日志 `/tmp/roncin-org-switch-check-web.first.log`，2026-10-20 04:12:16–04:13:30 UTC，总 73.60 秒；权限/枚举、架构、重复函数、Biome 与 tsc 通过，Vitest 3 文件失败 / 277 通过 / 1 跳过，18 项失败 / 1556 通过 / 12 跳过，测试 62.67 秒。错误均为上述三处缺少 history.location 的旧替身。
- **修旧替身定向**：3 文件 21 项通过，24.26 秒，日志 `/tmp/roncin-org-switch-review-test-contract.log`。
- **第二轮 check:web**：日志 `/tmp/roncin-org-switch-check-web.second.log`，2026-10-20 04:14:22–04:15:34 UTC，总 71.94 秒；全脚本退出 0，Vitest 280 文件通过 / 1 跳过，1574 项通过 / 12 跳过，测试 60.33 秒。该轮早于后续补漏，不能当作最终代码验收。
- **原因 owner 补漏定向**：真实 helper 生命周期与账单/对冲写入相邻 5 文件 11 项通过（6.23 秒），日志 `/tmp/roncin-org-switch-review-owner.log`。全部六个生产 makeVersionActions 实例现均传 getter。之后领域/共享代理补齐批量编辑与导入、主会话补标准订单 Form 事件；全部停止编辑后，独立审查再次读取最终新增差异并运行第三轮完整 check:web。

## 最终验收

**审查结论：本任务已发现的行为缺口均已修复，无剩余已知功能阻断；允许主会话按既有授权提交并合并本地 main。** 未修的检查提示、原有跳过与未定位连接噪声如上如实保留，未用关规则或产品回退绕过。

- Lint：通过。最终脚本的 `web lint` 包含 `biome:lint` 与 `tsc --noEmit`；Biome 扫描 775 文件，0 error，保留上述 1 warning / 7 info。
- TypeCheck：通过（最终完整脚本覆盖，无额外重复运行）。
- Tests：通过。**285 个测试文件通过、1 个存量文件跳过；1587 项通过、12 项存量跳过，共 1599 项**。最终 Vitest 61.94 秒。
- 权限生成物：305 个权限键通过；72 个枚举、7 个错误原因域通过；没有手改生成物。
- 架构：18 项检查器测试通过；扫描 459 个产品文件、1795 条本地依赖、4 条 features 能力边，没有违规。
- 重复函数扫描相关测试：20 项通过。
- `git diff --check` 通过；另外对 33 个新增实际文件执行 `git diff --no-index --check /dev/null <file>`，全部通过。本地 node_modules 符号链接明确排除，不应提交。

**最后一轮门禁时序**：2026-10-20 **04:22:07.152895–04:23:25.377494 UTC**，总 **78.22 秒**，`pnpm run check:web` 退出 **0**。日志 `/tmp/roncin-org-switch-check-web.log`。开始前所有实现代理和主会话停止产品修改，独立审查又读取最后的订单 Form 事件、导入及共享箱分配差异；本轮通过后只完善本审查文档，不再修改产品或测试。第一轮失败、第二轮通过但早于追加补漏的事实均保留，不混同最终代码的验收证据。

最终已复核范围覆盖：集中 route metadata/共享功能 tabs/新 access 落点，真实 Data Router 懒加载与同路径提交/POP/取消/失败，工作区先卸载再换身份及请求/缓存隔离，多个 guard 与临时确认撤回，受控/trigger ModalForm，角色矩阵与真实共享联系人，财务 Form/state/行内费用/原因框，企业资源字段/图片/批量关联与有效导入，订单独立模态编辑/拆票/共享箱及原模板草稿边界。逐项接入及排除依据见 `research/domain-forms-coverage.md` 和 `research/guards-implementation.md`；本轮未新建未来表单治理平台。
