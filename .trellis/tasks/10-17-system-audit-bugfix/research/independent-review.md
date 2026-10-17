# 独立复核记录

## 范围与方法

- 按任务 PRD/design/implement 与前端状态管理、组件、质量、钉钉登录规范复核。范围为 R1 写反馈、R2 工作区缓存、R3 六类财务导出；R4 真实库/CI 由主会话单独验证，本记录不把未运行的数据库检查写作通过。
- 对照任务基线 `be38b71b` 至当前工作树，含已提交的 R2 `30bad387`，没有只看未提交差异。
- 读取 `queryClient.ts`、`requestClient.ts`、`OrganizationWorkspace.tsx`、`AppLayout.tsx`、`OrganizationSwitcher.tsx`、公开钉钉预热契约与真实 Provider/请求回归；核对首帧 keyed 重建、layout 清理、迟到成功/失败、匿名、StrictMode、公开入口与业务实例分离。
- 读取 `csv.ts`、公共 FinanceLedgerTemplate/types、六页 ledgerExport.ts 与消费者 index.tsx、提成导出。逐列对照生成 DTO、六页现有状态文案、财务服务契约；检查金额/币种、精度、日期、枚举、选中范围与资源清理。
- R1 对照 `/tmp/write-audit.json` 的 214 处直接生成客户端非 GET 调用及生成响应形状，交叉复核共享 CRUD/SubEntityDrawer、ModalForm、confirmWithReason/makeVersionActions、订单状态与拆票/改配/航程等特殊响应标识、主数据 CRUD、通讯录同步、公开登录、费用审核及提成发放的 useMutation 路径。214 项含 POST 预览及间接组件/Hook 回调，不能等同 214 个相互独立写接口。

## 发现与修正

### R3：资金流水支付方式漏映射（已修）

- 文件：`web/src/pages/finance/cashflows/ledgerExport.ts`、`index.tsx`、`ledgerExport.test.ts`。
- 证据：表单实际提交 `BANK_TRANSFER/CHECK/CASH/THIRD_PARTY`，后端原样保存；原导出直接写 `row.paymentMethod`，测试却使用中文夹具，无法发现真实代码泄漏。
- 修正：将现有四项支付方式选项供表单与 CSV 共用，导出匹配中文名称；真实后端允许的自定义字符串仍保留其业务值。测试夹具改真实代码，并逐项覆盖四个现有选项。

### R3：请求内保存快照导致当前页导出竞态（已修）

- 文件：`web/src/components/ui/finance-ledger-template/FinanceLedgerTemplate.tsx`、`FinanceLedgerTemplate.export.test.tsx`。
- 证据：原模板在 `request` 的 await 返回后立即 `setCurrentData`。已安装 ProTable 的 `useFetchData` 在 `getData` 返回后才检查 `signal.aborted` 与 `success === false`。因此 B 被采纳后 A 迟到仍可修改导出状态，失败响应的部分数据也可覆盖导出，表格与 CSV 不一致。
- 修正：请求只返回结果，以数据数组为 WeakMap 键暂存 total/summary；仅在官方 `onLoad` 采纳后同步导出与汇总快照。没有另建请求序号/状态机，没有依赖 ProTable 未在公开类型中声明的第二回调参数。WeakMap 不保留已丢弃数组。
- 回归：真实 ProTable 中 B 先完成/A 迟到后仍导出 B；`success:false` 部分结果不替换表格当前有效数据和导出。

### R1：确认层的 false 语义不等同 ModalForm（已修）

- 证据：已安装 Ant Design `ActionButton.js` 在返回 Promise resolve 时无条件 close，故 async `onConfirm` 返回 `false` 仍关闭 Popconfirm；直接 reject 虽保留弹层，却因 Popconfirm 无 isSilent 协议产生未处理拒绝。
- 已告知主会话与写反馈实现代理，并补充 `MasterDataTemplate.handleToggleActive` 将 false 吞成 undefined 的相邻传播点。共享受控 `AsyncPopconfirm` 由该代理统一实现；独立复核已读取实际 Ant Design `Popconfirm`、`ActionButton` 和 Modal hook 代码，确认仍由官方 Promise 管理 loading，失败仅拦截本次自动关闭，不产生未处理拒绝。

## 验证记录

- 首轮缓存/导出：12 个文件、45 个测试全部 PASS，无 SKIP。
- 独立补充 R2 集成交叉窗口：真实 Provider + request/Axios adapter 在 StrictMode 模拟卸载后，新请求完成且旧传输迟到401不导航、不提示、不覆盖。`requestClient.workspace.test.tsx` 更新后12例全部 PASS。
- 支付方式局部修正：`cashflows/ledgerExport.test.ts` 与 `cashflow-save.test.tsx`，2 文件、6 测试 PASS。
- 公共模板竞态修正：`FinanceLedgerTemplate.export.test.tsx` 与 `FinanceLedgerTemplate.test.tsx`，2 文件、7 测试 PASS。
- `pnpm --dir web tsc`：最新 PASS；测试开发途中 Page 夹具遗漏 success 类型已修，未跳过类型检查。
- 两个缓存实现、CSV工具、六个映射、模板等16文件 Biome lint PASS；修正后5个相关文件 lint PASS。
- `git diff --check`：PASS。
- 修正后联合定向：17 个文件、66 个测试全部 PASS，无 SKIP。包含缓存/导出全部回归与 R1 真实请求代表：重置密码、角色、客商角色、账单取消、对冲取消。
- 最终全量门禁/构建、浏览器与完整 PostgreSQL 验收由主会话运行；确认组件收敛后的专项验证继续补录。

## 剩余边界

- 工作区 token 约束响应展示生命周期，不回滚已发出的服务端写入；服务端组织授权与事务仍是写入正确性的边界。
- 财务导出仅已加载选中行或当前页，没有全量接口或循环翻页；字段从真实 API DTO 映射，不伪造界面历史投影字段。
- 不提交代码，不接管主会话的规范编辑、数据库资源、提交与最终合并。

## R4 增补静态复核

主会话追加分工后，对照 `be38b71b` 独立读取 CI、正式迁移前态辅助函数及全部本次服务端测试差异；没有重跑 PostgreSQL 整包，也没有修改服务端或其夹具。

- `.github/workflows/ci.yml` 同时配置集成库、`DATABASE_SOURCE` 与迁移开关，完整 `./...` 带 count=1 运行，并以 JSON 记录 PASS/FAIL/SKIP；任何 SKIP 失败。独立 Schema 的正式迁移 helper 已被两处原 runtime 自动建表用例复用。
- `cmd/sync-dev` 通过真实 `BootstrapAdministrator` 构造系统前态；“员工无系统身份”排除合法 bootstrap 管理员，仍禁止普通员工有系统成员身份。其余组织与经营数据边界断言保留。
- 企业微信退役用例先只应用至被测迁移时点并完整比较历史快照，再升级最新迁移验证账号/会话，避免让后续合法删除角色范围字段误报为当前迁移破坏数据。
- 索引用例先物化本 Schema 的索引 OID，避免优化器在并发 Schema DDL 时解引用其他测试关系；原无条件索引=0、特定部分索引=1及迁移记录=1断言均保留。
- 提成夹具改为公司经营组织；每个核销来源有独立订单及完整应收/应付账单，不重复核销同一1000账单。合并应付800补齐第三票无员工归属的500成本，原计提票利润200/300与提成20/30保持。
- 对冲反转断言从内部字符串前缀改为真实来源对冲ID、逐订单20/30追回与独立非空幂等键；父单PAID、总冲减50、净额归零等业务断言保留。未发现通过弱化断言、增添SKIP、删除约束或修改产品逻辑规避真实库失败。

R4 完整真实库 PASS/FAIL/SKIP 以主会话最终 JSON 审计记录为准，本静态复核不替代数据库实际执行。

## R1 最终异步确认专项复核

### 实际代码路径

- 逐一核对 `AsyncPopconfirm` 的19个消费者文件、23处异步写确认，及其间接调用的 `makeVersionActions.run`、共享主数据模板、子实体模板、账单/流水动作、共享箱删除与拆票执行。
- 真实请求层消费失败后返回的 `undefined` 由各写调用的 `isWriteResponse` 转成 `false`；实体缺 ID 与 `success:false` 同样阻断。组件回调中成功落库后的 `undefined` 是既有 `void` 成功契约，不能把组件全部 `undefined` 统一改成失败。
- 原生 Popconfirm 保留处分别为本地联系人删除、本地建账明细移除及未接入写接口的批量操作，没有遗漏本次服务端异步写确认。
- `confirmWithReason` 的 async 返回会吸收 Ant Design Modal hook 的 thenable，开启 `isSilent`；其 false→reject 语义可留窗且不会像 Popconfirm 那样产生未处理拒绝。账单与对冲的真实请求回归验证了原因输入保留和成功重试。

### 发现与直接修正

1. 七条调用链本地 catch 提示错误后仍返回 `undefined`，会绕过确认组件的 reject/false 保护。已在后台任务重执行、合同终止、预设停用、账单确认、流水确认、共享物理箱删除、拆票执行的 catch 中明确返回 `false`。
2. 已迁移确认链的缺 ID、缺当前上下文、权限或表单前置校验失败补成 `false`，避免尚未写入就触发成功关闭；`makeVersionActions.run` 缺必要版本时也返回 `false`，并同步测试断言。没有更改版本判定规则、请求客户端契约或服务端写入行为。
3. 在现有账单真实请求回归文件增加实际 `AsyncPopconfirm` + 页面动作代表用例：依次 reject、undefined、`success:false`、缺 ID 后仍保持 open，重试实际成功才刷新一次并关闭。失败判断使用真实 `onOpenChange` 未发送 false 与触发器 open 类，成功同时核查刷新及关闭事件/leave 状态，避免将 happy-dom 的动画可见性当作业务状态。

### 最终专项验证

- 定向11文件、52用例全部 PASS，无 SKIP：包含 AsyncPopconfirm、账单/对冲真实请求、版本动作、共享箱、拆票、主数据模板/Hook、资金流水登记以及角色/用户表单相邻路径。
- Vitest 未报告 unhandled rejection。联合用例进程有一条 localhost:3000 的 AggregateError 控制台输出；未变更的相邻用例仍需按主会话完整门禁结果核对，不能将输出隐藏后宣称无噪声。
- 最终新增账单文件独立执行2例全部 PASS，输出无 stderr、无 AggregateError（`/tmp/roncin-independent-bill-confirm.log`），确认联合日志中的连接输出不来自该文件。
- `pnpm --dir web tsc`：PASS。
- 相关22文件 Biome lint 与最终新增代表测试独立 lint：PASS。
- `git diff --check`：PASS。
- 产品/测试最终变更时点：2026-10-17 15:52:38 UTC，之后仅补审查文档。全量门禁、完整真实库、生产构建和最终提交合并以主会话记录为准。

本轮已发现的局部机械问题均已修复；没有待产品决策、公共接口或模块边界争议的未修复问题。未提交代码，也未操作主会话数据库资源。

## 全量门禁暴露的原生 Modal 未处理拒绝增补

前述专项通过不代表最终全量门禁通过。主会话后续完整运行虽然250个测试文件、1352用例断言全部通过，Vitest仍捕获1个 `Unhandled Rejection { message: undefined }`，因此该轮门禁失败。

### 根因与修正

实际 Ant Design `ActionButton` 在 onOk Promise 拒绝后会重新抛出拒绝，除非 `isSilent()` 为 true；`App.useApp().modal.confirm` 只有在调用其返回 thenable 的 `.then` 时才启用 silent。新增 `return Promise.reject()` 正确保留了失败弹层，但直接丢弃确认返回值导致拒绝溢出。

- 修正10个产品文件、13处直接确认：费用更正撤回；工作台申请与重新提交；提成及调整确认；申请批准；调整补付/扣回；订单锁定；费用删除；整票改配；订单主流程、退关、结案。
- 各处沿用仓库 `FeeSupplementSection` 已有 `.then(() => {}, () => {})` 消费确认结果模式。onOk 本身仍拒绝，由 Ant Design 留窗并解除 loading；未全局忽略 unhandled，未替换成 resolve/关闭，也未隐藏业务错误。
- 同步这些确认内部 catch 的失败传递；3处原因确认 catch 返回false，订单费用的前置写权限失败向确认层传播失败。唯一返回普通 `{}` 的订单状态测试mock改为Promise，产品侧没有新增兼容mock的optional then分支。

### 增补验证

- 9个相关测试文件、69个用例PASS，覆盖工作台、提成列表/调整/申请、订单锁、改配、状态流转、费用更正及费用工作台。
- 新增 `order-detail-transitions.modal.test.tsx` 使用真实 `App` Modal，主流程、退关、结案3例PASS：直接点击按钮（不手工消费onOk Promise），验证必填原因留窗、reject、undefined、success:false、缺ID均保留输入且不完成；第五次实际成功才调用完成回调并进入关闭状态。
- 两组定向输出干净，无未处理拒绝：`/tmp/roncin-modal-confirm-regression.log`、`/tmp/roncin-modal-real-regression.log`。
- 最终类型检查、13个相关文件Biome和 `git diff --check` PASS。
- 主会话还运行了诊断全量轮（250文件1352例断言PASS且无unhandled），但该轮是否完整采集所有最终变更不确定，不作为最终门禁；最后交付仍以冻结后新一轮 `check:fast` 为准。
