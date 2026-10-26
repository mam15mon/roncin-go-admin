# Journal - mam15mon (Part 3)

> Continuation from `journal-2.md` (archived at ~2000 lines)
> Started: 2026-10-21

---



## Session 152: 空运出口表单 UI 对齐海运出口五卡片架构
<!-- trellis-session: v=2 fp=d02ad5aa4fdd89d2 -->

**Date**: 2026-10-21
**Task**: 空运出口表单 UI 对齐海运出口五卡片架构
**Branch**: `main`

### Summary

把空运出口新建/详情表单从四卡片平铺重组为海运同款五卡片架构（业务与客户/订舱与运输/本票货物/分单信息/内部人员）：提取 TemplateFieldGroup 小节竖标组件供两 kind 共用，AirMasterFields 拆为航线/班期/申报子件，备注换 SeaCollapsibleNotesField 可折叠交互（name 支持嵌套路径）。字段与 payload 契约零变化，海运视觉零回归；新增 air-create-layout 结构测试锁定布局。trellis-check 8/8 验收通过，tsc/biome/architecture 全绿。

### Git Commits

| Hash | Message |
|------|---------|
| `632f5320` | refactor: 提取订单模板小节竖标组件 TemplateFieldGroup 供海运空运共用 |
| `234663d3` | refactor: 拆分 AirMasterFields 为航线/班期/申报子件，字段属性逐字保留 |
| `97cb74d7` | feat: 空运出口表单重组为五卡片信息架构对齐海运出口 |
| `201ba2ad` | test: 空运模板布局测试冲刷挂载请求并锁定小节与备注顺序 |
| `82a34fcc` | docs: 归档空运出口表单对齐海运五卡片任务与模板架构导航 |

### Status

[OK] **Completed**


## Session 153: 空运出口危险品三元组对齐海运交互
<!-- trellis-session: v=2 fp=077ed52509b45e21 -->

**Date**: 2026-10-21
**Task**: 空运出口危险品三元组对齐海运交互
**Branch**: `main`

### Summary

空运出口「本票货物」卡接入 SeaDangerousGoodsFields：勾选危险品动效展开 UN NO./CLASS NO./截申报时间，取消清值、UN 四位校验仅选中激活；air form-adapter 按海运范式补三字段序列化与 dayjs 回填。契约与服务端零改动（Order 共享字段，写路径 kind 无关）。trellis-check 6/6 验收通过，展开态断言锁 options 接线；海运路径零变化。

### Git Commits

| Hash | Message |
|------|---------|
| `ef6502c3` | feat: 空运出口补危险品 UN NO. / CLASS NO. / 截申报时间三元组 |
| `a0950af0` | test: 空运危险品布局用例补勾选危险品后展开断言锁定 options 接线 |

### Status

[OK] **Completed**


## Session 154: 空运出口危险品包装类别 PG 枚举全链路
<!-- trellis-session: v=2 fp=1f5c22fd5d04f9c0 -->

**Date**: 2026-10-21
**Task**: 空运出口危险品包装类别 PG 枚举全链路
**Branch**: `main`

### Summary

Order 共享契约新增 DangerousPackingGroup 枚举（严格照抄海运 shipment_type 五层链路：proto→biz 领域类型→service switch→ent Nillable+迁移 character varying NULL→web client）；空运危险品动效区加包装类别第四字段（showPackingGroup prop，海运零变化），proto 注释含 DGR 适用范围/UNSPECIFIED 双语义/UN↔PG 查表钩子三要素。trellis-check 7/7 通过、生成器重跑零差异、迁移幂等；固化订单共享枚举字段配方入 server spec。

### Git Commits

| Hash | Message |
|------|---------|
| `26bb579f` | feat: 订单契约新增危险品包装类别枚举字段 |
| `278a32f1` | feat: 空运出口危险品区接入包装类别字段 |
| `0184b451` | refactor: 收窄包装类别选项常量为模块内私有 |

### Status

[OK] **Completed**


## Session 155: 订单进程双标签与操作人员展示优化
<!-- trellis-session: v=2 fp=812b8566d9eaa800 -->

**Date**: 2026-10-23
**Task**: 订单进程双标签与操作人员展示优化
**Package**: web
**Branch**: `codex/order-list-lifecycle-badges`

### Summary

完成订单列表退关与完结双标签、操作进度列命名、异常独立展示及操作人员仅姓名；独立 review 无阻塞，准备合并 main 并安全删除工作分支。

### Main Changes

- 共享订单列表的进程使用强类型数组，已退关与已完结可并存；异常仅在异常列显示。
- 操作人员仅显示姓名，未分配保持占位；补充列表状态展示规范并归档任务。

### Git Commits

| Hash | Message |
|------|---------|
| `a0d73b89` | fix: 分开展示订单进程并精简操作人员 |

### Testing

- [OK] 定向回归2文件18用例、追加组件4用例、Biome、antd lint、tsc、diff检查均通过。
- [OK] 完整前端门禁非测试项通过；默认及16并发的两个提单可见性失败单独17用例通过，最终4并发全量299文件1659用例通过（既有12用例跳过）。

### Status

[OK] **Completed**

### Next Steps

- 完成本地 main 合并并用 git branch -d 删除已合并工作分支。


## Session 156: 全局表格多列冻结与铺排排序完成
<!-- trellis-session: v=2 fp=c7f86ea7d4296f45 -->

**Date**: 2026-10-23
**Task**: 全局表格多列冻结与铺排排序完成
**Branch**: `codex/table-column-layout`

### Summary

实现全局字段多列左右冻结、全量网格铺排、连续编号及拖动和精确移位；独立 review 修复生命周期、结构锚点与低高度布局。完整前端门禁303文件1653用例通过，浏览器22组及真实业务页4组通过。按用户授权合并main、删除任务分支并推送；保留其他会话的海运测试及工作台任务改动。

### Git Commits

| Hash | Message |
|------|---------|
| `384e9d1c` | feat: 支持全局表格多列冻结与铺排排序 |

### Status

[OK] **Completed**


## Session 157: 统一订单审核锁与费用补审账务回退
<!-- trellis-session: v=2 fp=4ebe75b5f1739a4b -->

**Date**: 2026-10-23
**Task**: 统一订单审核锁与费用补审账务回退
**Branch**: `codex/unified-order-review-lock`

### Summary

完成统一人工订单审核锁，移除自动锁与整票费用审核写流程；费用按逐费建账占用控制编辑/删除，锁后 ADD/REPLACE/VOID 通过费用更正补审；补充账务回退依赖提示、合并账单整单取消影响范围、提成经济重算与 Go 1.26.10 漏洞修复。真实 PostgreSQL 定向场景和 check:fast 全量门禁通过。

### Git Commits

| Hash | Message |
|------|---------|
| `2e56e7e0` | feat: 统一订单审核锁与费用补审账务回退 |

### Status

[OK] **Completed**

## 2026-10-24: 订单详情与费用录入规则调整（10-24-order-detail-fee-refinements）

### Summary

内部人员业务/客服由客户档案带入且界面/服务端双路径只读（创建与草稿换客户均按新客户档案重挂）；订单主流程允许跳转任意后续节点并同事务补齐中间节点（一键完结同样补齐）；费用大类与订单服务类型全链路下线；费用发生日期退役为不可变录入日期（created_at 口径）；订单费用汇率统一按订单 ETD 所在自然周解析（ETD 缺失报 ORDER_ETD_MISSING）；新增独立权限码 system.finance.order_fee.exchange_rate.override 控制同币种订单的手工调汇率。含 proto/Ent/权限/OpenAPI 全部生成物同步与正式迁移 20261024100000。check:fast 全量门禁通过。

### Git Commits

| Hash | Message |
|------|---------|
| `8a1bd16d` | feat: 费用大类与发生日期数据层下线及正式迁移 |
| `76f9b7dd` | feat: 费用与订单契约调整及独立汇率调整权限码 |
| `a8aedc4a` | feat: 订单状态跳转补齐、人员客户带入、汇率按 ETD 解析与费用日期退役 |
| `e9b9df89` | feat: 前端适配人员只读带入、状态跳转、大类下线与录入日期口径 |
| `80c0614a` | feat: 草稿换客户时按新客户档案重挂业务/客服人员 |
| `d1c550a4` | fix: review 修复死列死代码与集成测试适配新契约 |

### Status

[OK] **Completed**

## 2026-10-24: 业务/客服两岗恢复必填（上任务修正）

### Summary

按用户反馈恢复业务/客服必填：前端两岗新建必填（提示先维护客户档案）且保持只读；服务端三岗校验恢复，创建与草稿换客户均在客户档案覆盖后校验两岗在册，缺岗返回 ORDER_CUSTOMER_STAFF_MISSING 整体回滚。验收脚本客户补配客服岗位。check:fast 全绿。

### Git Commits

| Hash | Message |
|------|---------|
| `HEAD` | feat: 业务/客服两岗恢复必填（只读由客户带入不变） |

### Status

[OK] **Completed**


## Session 158: 订单详情与费用规则任务补齐记录并归档
<!-- trellis-session: v=2 fp=6d45a52567c666d1 -->

**Date**: 2026-10-26
**Task**: 订单详情与费用规则任务补齐记录并归档
**Branch**: `main`

### Summary

核对原任务完成记录，修正客户缺岗规则与人员必填验收描述，补录创建缺岗防绕过修复及验证结果；任务已归档。临时 PostgreSQL 16 个订单事务场景和后端完整门禁通过；本轮全量前端有 1 项失败，该文件单独复跑 3 项通过。作业待办明细任务继续保持规划状态。

### Git Commits

| Hash | Message |
|------|---------|
| `b5595a2f` | fix: 阻止客户缺岗时创建订单 |
| `611d685e` | feat: 业务/客服两岗恢复必填（只读由客户带入不变） |

### Status

[OK] **Completed**


## Session 159: 作业待办明细与处理直达完成验收
<!-- trellis-session: v=2 fp=39010504acddc5c8 -->

**Date**: 2026-10-26
**Task**: 作业待办明细与处理直达完成验收
**Branch**: `codex/workbench-todo-details`

### Summary

完成本人协作订单未建账费用与进行中异常分页待办，摘要明细统一权限和记账事实，复用订单费用页及异常直达；独立审查修复候选误依赖主数据管理权限及详情误依赖人员读取/创建权限，补充真实路由与工作区隔离回归。前端完整门禁1651项通过、12项既有跳过，后端完整门禁及专用PostgreSQL回归通过，临时数据库已清理。任务已归档，按用户授权合并main、安全清理已合并分支并推送。

### Git Commits

| Hash | Message |
|------|---------|
| `1bec915e` | feat: 增加本人作业待办明细与订单处理直达 |
| `8a452e88` | docs: 记录作业待办验收与审查结果 |
| `67936c82` | test: 补齐核销工作台的主题上下文桩 |
| `71fdbdfb` | docs: 明确作业待办的记账口径与读取权限 |

### Status

[OK] **Completed**


## Session 160: 修复订单直接完结入口
<!-- trellis-session: v=2 fp=4d14c6e3127ca3ec -->

**Date**: 2026-10-26
**Task**: 修复订单直接完结入口
**Branch**: `main`

### Summary

修正服务端 orderAllowedActions 对完结动作遗留的已放单或已退关前置条件，与现有结案事务直接补齐主流程的行为对齐。补充海运及空运各节点、终止状态、业务锁和活动异常资格回归，以及详情页草稿直接打开完结确认验证；同步规范。定向后端测试、页面相关22项测试、Biome、git diff --check及check:fast全栈门禁通过（前端1654项通过）。保留海运共享主单航程继承规划任务，尚未实施。

### Git Commits

| Hash | Message |
|------|---------|
| `aa86ae84` | fix: 允许订单在主流程完成前直接发起完结 |

### Status

[OK] **Completed**


## Session 161: 完结与人工审核锁分离
<!-- trellis-session: v=2 fp=4cc80b08c3fa58e4 -->

**Date**: 2026-10-26
**Task**: 完结与人工审核锁分离
**Branch**: `codex/sea-master-voyage-inheritance`

### Summary

按用户明确契约：最终状态=完结+审核锁定，完结未锁仍可修改资料及未建账费用，已建账先按权限逐步回退，锁后费用变更补录补审才生效。同步统一后端内容门禁、EDIT动作、费用能力、审核锁资格与执行、海空运共享资料及拆票改配预读执行；移除失效ORDER_CLOSED错误；同步前端写策略及规范。真实PostgreSQL覆盖闭单编辑、费用建账/回退/锁后拒改、海空运共享与单证、海运及非海运闭单审核快照/版本/幂等/审计，零SKIP。后端check:server完整通过；原工作区check:fast被另一窗口航程测试类型错误阻断，保留其改动；仅含本轮变更的临时副本check:web完整通过，1659项测试通过（一次无关验证码倒计时超时单独及完整复跑通过）。未创建或归档本轮任务，海运主单航程规划和修改保留给另一窗口。

### Git Commits

| Hash | Message |
|------|---------|
| `1eb9fa9b` | fix: 完结订单仅在审核锁定后冻结内容 |

### Status

[OK] **Completed**


## Session 162: 海运共享主单航程继承与统一修改
<!-- trellis-session: v=2 fp=79289660d7e67cf2 -->

**Date**: 2026-10-26
**Task**: 海运共享主单航程继承与统一修改
**Branch**: `codex/sea-master-voyage-inheritance`

### Summary

完成一主多分共享航程自动带入、显式差异采纳与只读来源，统一修改按真实影响成员展示订单号。完整前端门禁及只读浏览器验收通过，任务已归档。

### Main Changes

- 候选查询由 React Query 管理，主单或船公司变化使旧确认失效，保留独立分单与货物资料。
- 共享航程预览补读准确订单号，港口搜索隔离迟到结果，缺名称与资料缺失给出明确提示。
- 将列表筛选口径及存量改配候选配对校验记为后续问题。

### Git Commits

| Hash | Message |
|------|---------|
| `3660fa1f` | feat: 优化海运共享主单航程继承与统一修改 |

### Testing

- [OK] pnpm run check:web --maxWorkers=32：307文件、1677项通过，12项既有跳过。
- [OK] 真实浏览器8项交互验收通过，未发出业务写请求，无页面异常。
- [OK] git diff --check 与任务上下文校验通过。

### Status

[OK] **Completed**
