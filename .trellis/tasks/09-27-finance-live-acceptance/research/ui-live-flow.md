# Research: 同一订单财务浏览器实弹路径

- Query: 为后续 finance-live-acceptance 子任务定位同一订单从录费用、建账单、收款核销到查看对应提成的最短真实页面路径、选择器及最少 API 夹具。
- Scope: internal
- Date: 2026-09-27

## Findings

### 1. 已读取的任务与规范

- 后续任务 `.trellis/tasks/09-27-finance-live-acceptance/{prd,design,implement}.md` 要求关键财务写入通过 UI；API 只准备基础资料/订单、辅助只读核对。
- `.trellis/workflow.md`：研究持久化、任务阶段、上下文清单与验收职责。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：费用没有独立状态，草稿账单也占用费用；订单固化提成三岗归属。
- `.trellis/spec/server/backend/operating-company-commission-attribution.md`：订单创建必须配置销售、操作、客服，创建时固化快照，后补人员不能替代创建快照。
- `.trellis/spec/server/backend/commission-scheme-assignment.md`：启用方案与员工分配交集决定来源日期资格，不能只创建裸规则。
- `.trellis/spec/server/backend/finance-commission-lock.md`：确认提成会产生财务锁；主链与取消重建辅助链应使用不同订单。
- `.trellis/spec/web/frontend/state-management.md`、`quality-guidelines.md`：真实页面消费当前请求协议、最小验证及受控测试命令。

### 2. 文件定位

| 文件 | 用途 |
| --- | --- |
| `web/tests/e2e/finance-bill-batch.e2e.ts` | 现有财务页面验收，包含可复用登录与核销工作台定位方式 |
| `web/playwright.config.ts` | 单 worker，真实开发服务 baseURL，失败 trace/screenshot |
| `web/src/pages/orders/fees.tsx` | 订单费用页、费用选项和建账工作台编排 |
| `web/src/pages/orders/components/fees/OrderFeeTableTabs.tsx` | 应收/应付行内新增、保存、勾选、建账和占用拒绝 |
| `web/src/features/finance/bill-creation/BillCreationWorkbench.tsx` | 从订单带入费用后快速进入账单资料，原子建账与批次确认 |
| `web/src/features/finance/bill-creation/BillGroupCard.tsx` | 账单日期、对账抬头、结算账户等表单 |
| `web/src/features/finance/bill-creation/BillCreationResultTable.tsx` | 新生成账单编号与确认入口 |
| `web/src/pages/finance/cashflows/index.tsx` | 流水登记表单、草稿确认 |
| `web/src/pages/finance/verifications/VerificationWorkbench.tsx` | 公司→结算单位→币种级联候选、勾选、自动分配、提交核销 |
| `web/src/pages/finance/commissions/components/CommissionCreateModal.tsx` | 来源核销→员工身份候选→预览→生成草稿 |
| `web/src/pages/finance/commissions/components/CommissionDetailDrawer.tsx` | 来源单号与逐订单计算快照展示 |
| `web/src/pages/finance/commissions/components/CommissionLineTable.tsx` | 逐订单、收入、成本、毛利、提成与展开费用明细 |
| `server/api/order/v1/order.proto` | 当前创建订单与人员分工输入契约 |
| `server/api/finance/v1/settlement.proto` | 当前方案分配、提成来源与写入契约 |
| `scripts/acceptance-finance-bill-batch.mjs` | 基础客户、账户、船公司、订单可参考结构；现存人员与规则字段漂移须修复 |
| `scripts/run-acceptance-finance-disposable.mjs` | 隔离环境编排；Stage 1 应收/应付/UI，Stage 2 外币 |

### 3. 现有 e2e 的证据缺口与断言漂移

1. `finance-bill-batch.e2e.ts:37` 从首 100 条订单里找任意 `ACC-FIN-`，然后查看已经建账的费用，未通过页面录费。
2. `finance-bill-batch.e2e.ts:114` 只断言登记按钮可用，未登记流水；核销又选第一条资金/第一条账单（:148），缺乏同一链条定位。
3. `finance-bill-batch.e2e.ts:171` 从全局提成里找带调整的历史记录，未证明当前核销对应提成，生成弹窗只检查空态（:213）。
4. 当前台账标题是 `账单列表`（`bills/index.tsx`）、`资金流水列表`（`cashflows/index.tsx:458`）、`核销记录列表`（`verifications/index.tsx:381`），现有用例仍断言 `账单管理台账`、`资金流水管理`、`核销台账管理`。应同步当前真实标题。
5. 注释仍称 max dev，实际 Playwright 配置注明 vite dev；这是文案漂移，不能据此假定 Umi 入口存在。

### 4. 最少 API 夹具

主会话已确认并负责准备合法公司工作台及成员资格，本研究不重复调查 bootstrap。浏览器登录后应按确定公司 ID 切换/验证工作台；不能任取组织候选第一项。

最薄闭环建议新建独立客户与空订单，以 CNY、本币 CNY、数量 1、单价 125、税率 0、无应付成本和 SALES 10% 毛利方案作为固定输入：费用/账单/流水/核销均 125，已实现收入 125、分摊成本 0、毛利 125、提成 12.50 CNY。这组期望由输入确定；若执行选择了不同税率或成本，必须在运行前重新明确公式。

基础 API 准备清单：

- 唯一客户（带本次 runId，不复用 HTTP 已核销客户）；可无业务责任人，订单三岗单独提供。
- 该客户启用的 CNY 应收结算账户，设为默认；账户是账单快照必填，参考 `acceptance-finance-bill-batch.mjs:160`。用当前契约，不带已移除的旁置组织字段。
- 可用船公司及直单主单信息，参考脚本 :178/:190；创建空海运出口订单并保存返回 ID/orderNo，不从全局列表寻找。
- `CreateOrderRequest.personnelAssignments` 在创建时配齐三岗（`order.proto:598`）；同一合法员工可以兼三岗，岗位枚举取当前 proto/生成枚举。不能照搬脚本 :801 的事后 SALES 分配。
- 启用且绑定上述员工的 SALES/REALIZED_PROFIT/10% 方案，生效日覆盖来源日期；当前 `CreateCommissionRuleRequest`/员工分配 API 以 `settlement.proto` 为准。避免与 HTTP 场景同员工同身份方案日期重叠，优先单独业务员工。
- 公司费用候选中启用 CNY 费用项目、默认计费单位、税率 0；若隔离库没有合适候选，用 API 准备费用主数据，不能用 API 写本次订单费用。
- 启用订单、账单、批次、收付款、核销、提成编号规则。日期按服务端 Asia/Shanghai 业务日期选择，避免 UTC 午夜前后方案日期漂移。

严禁 API 预写关键费用、账单、流水、核销或提成。API 只读详情以及监听 UI 请求响应均可辅助取证。

### 5. 最短页面链路与选择器

通用定位：优先 `getByRole` / `getByLabel`，每个 dialog/drawer/行内作用域先限定，再选目标。下拉项使用当前可见 `.ant-select-dropdown:not(.ant-select-dropdown-hidden)` 中明确名称；不要 `first()` 选任意业务记录。可以仅在 scoped row 中用数据字段后缀定位行内输入，但须在浏览器 DOM 确认。

#### 5.1 费用录入

入口 `/orders/sea-export/<fixture.orderId>/fees`。

- 点击 `getByRole('button', { name: '新增应收费用' })`。
- 真实实现是 EditableProTable `addEditRecord`（`OrderFeeTableTabs.tsx:632`），**不是 FeeFormModal**。新增临时行 `data-row-key` 以 `new_` 开头；可定位 `.ant-table-row[data-row-key^="new_"]`。
- 新行默认客户、CNY、数量 1、计费单位、当前发生日期（:640）。费用项目控件 placeholder `请选择费用项目`（:918），选择确定费用代码；单价 placeholder `0.00`（:1111）填 125；数量 placeholder `1`（:1138）；按需要确认 `billingUnitId`/CNY。
- 用行作用域 `getByText('保存', { exact: true })` 点 ProTable 默认 save 操作（:866），等待实际 POST `/api/v1/orders/<id>/fees` 的成功响应（:833），记录返回 feeId，页面显示 125.00。
- 刷新后按返回 feeId 的 `data-row-key` 定位确认持久化，再勾选该行 checkbox。保存前临时行 checkbox 禁用（:1423）。

#### 5.2 建账

- 点击应收卡片内 `生成账单（1）`，drawer 名 `费用批量转账单`。
- 从订单带入 feeIds + organizationId 会直接进入 `current=2` 账单资料（`BillCreationWorkbench.tsx:331`/:374），无需固定点两次“下一步”。
- `BillGroupCard.tsx:265`：标签 `对账抬头`、`账单日期`、`账期（天）`、`结算账户`；账户列表会自动填默认账户（:117）。确认日期、账户后等待正式预览形成，金额 125.00 CNY。
- 点击 `getByRole('button', { name: '原子生成 1 张账单' })`（`BillWorkbenchFooter.tsx:49`），监听 POST bill-batches 返回，记录 batchId/billId/billNo/line.feeId。
- 在“确认本批全部账单”之前取得草稿占用证据；随后点击该按钮（`BillCreationResultTable.tsx:127`），记录确认结果。
- 不需要绕行发票页面：此最短主链无需发票即可核销，开票由已有 HTTP 范围覆盖。

#### 5.3 收款登记与确认

入口 `/finance/cashflows`。

- `登记流水` → dialog `登记资金流水`（`cashflows/index.tsx:485`）。
- `getByLabel('所属公司')` 选择确定公司；`往来结算单位` 搜索确定客户；默认 `收款（流入）`、CNY、交易日期、银行转账。
- `发生金额` 填 125；`我方账户` 填本次识别标志；`银行水单号` 填 runId。默认 ModalForm 提交按钮需 DOM 确认（未自定义 submitText），使用 dialog 内 `button[type="submit"]`。
- 监听 CreateCashflow 响应取 flowNo/id；从当前表按 flowNo 精确定位（不要第一行），点击行 `确认`，Popconfirm 确认后等待服务响应与状态 `已确认`。

#### 5.4 核销

入口 `/finance/verifications` → `新建核销` → dialog `资金与账单核销工作台`。

- scoped `getByLabel('所属公司')`、`getByLabel('结算单位')`，CNY 默认值；候选依赖公司、结算单位、币种全就绪（`VerificationWorkbench.tsx:160`）。
- `.ant-card` 筛 `待核销资金`，内部按 flowNo 定位行；另一个 card `待核销账单` 按 billNo 定位。勾选各自 checkbox。
- 点击 `按余额自动分配`；`getByLabel('第 1 行核销金额')` 应是 125；点击 `/提交核销/`。
- 监听 POST verifications 取 verificationId/no（:295）。只读核对分配恰好连接刚才 cashflowId + billId，核销额 125，两者未核销余额 0。刷新页面后记录仍存在。

#### 5.5 预览、生成并查看对应提成

入口 `/finance/commissions` → `生成提成` dialog。

- `所属公司` 选固定公司；`有效应收核销` 搜索刚才 verificationNo；保持 `核销提成` Tab。
- `计提候选（员工 / 身份 / 已解析方案）` 选夹具员工的 SALES 唯一方案（`CommissionCreateModal.tsx:491`）。
- 点 `计算并核对预览`（:515），断言仅本次订单、对应订单号、已实现收入 125、成本 0、毛利 125、提成 12.50，CNY 折算 1。
- 点击 `生成草稿`（:310），监听 POST commissions 返回 id/no。UI 当前正确载荷是 verificationId + employeeId + personnelRole + organizationId，**没有 ruleId**（:341，proto :917/:918）。
- 精确定位该 commissionNo 行，点击 `明细`；抽屉检查 `来源单号` 等于刚才 verificationNo（`CommissionDetailDrawer.tsx:249`），逐订单表唯一行 orderNo 相同、金额一致（`CommissionLineTable.tsx:126`）。
- 展开该订单费用明细，配合只读 GetCommission 断言 line.orderId + line.fees[].feeId，刷新再打开相同记录，确保持久化。
- 主链完成可停于提成草稿详情；提成确认/付款不在本次 R3 最短路径内，如现有 HTTP 场景覆盖则报告注明。不要要求历史调整记录，新增正常提成无调整是合法状态。

### 6. 占用相邻场景与证据

- 主订单建立草稿账单后，暂不确认时即可查看 hasActiveBill=true，并检查重复建账拒绝；有 UI 的普通修改拒绝场景可以点击编辑并尝试保存，留存错误。删除按钮通常直接隐藏（`OrderFeeTableTabs.tsx:1341`），截图记录禁入口证据；服务端删除拒绝交由 HTTP 断言。
- 取消重建用独立第二订单/费用/草稿账单，从账单台账精确按 billNo 点取消、填写原因，再回费用页确认释放占用并重新建账。不能取消已经用于核销与提成的主账单。
- 已建账行仍允许 checkbox 勾选（:1423），点击建账时 `rejectBilledRows` 明确拒绝（:337）；不要错误断言 checkbox 必须禁用。
- 成功节点主动截图/操作摘要（默认配置只在失败截图），失败保留 trace。摘要保存 runId、orderId/No、feeId、batchId/No、billId/No、cashflowId/No、verificationId/No、commissionId/No 和金额对照。
- 对所有 UI 写请求先注册响应等待，再点击；不能用固定 sleep。浏览器 error/requestfailed 与业务成功响应一起记录，预期负例错误应单独标记，避免和意外错误混算。

### 7. 外部参考与版本

本研究仅调查仓库已提供代码及本地 Playwright 配置，未查询外部文档；未提出新组件 API 或依赖版本变更。框架版本以仓库 package.json/锁文件为准。

## Caveats / Not Found

- 只读源码研究，未启动服务、未执行浏览器、未运行测试、未改数据库；上述 DOM 后缀、默认 submit 名称和真实行字段需实施阶段浏览器验证。
- 当前费用页保留 FeeFormModal，但可见新增应收入口走行内表格，不能照 modal label 编写录入步骤。
- 脚本 fixture 仍有人员旁置组织、事后配岗、ruleId 等旧契约，不能原样复用；相关修复属后续实施范围。
- 实际提成 12.50 的前提是专用订单只包含上述单条应收、0 税率、0 应付成本、CNY 本币及 10% REALIZED_PROFIT 方案；实施如增加成本必须先写固定输入预期。
- 方案启用与分配、公司成员授权以及日期资格遵循现行契约，不能通过扩大权限或任取候选消除缺失。
