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


## Session 163: 反结案原因选填与审核锁校验
<!-- trellis-session: v=2 fp=20a67abba6e1db21 -->

**Date**: 2026-10-26
**Task**: 反结案原因选填与审核锁校验
**Branch**: `codex/sea-master-voyage-inheritance`

### Summary

完成并归档10-26-order-reopen-lock-guard。反结案原因选填并保留500字限制；已审核锁定不投影REOPEN，前端入口及确认提交前检查最新锁状态，已完结只读提示先解锁；服务端订单FOR UPDATE后专属检查审核锁，不套终止内容门禁。真实PostgreSQL三组零SKIP验证审核/反结案并发、锁后拒绝零写入、正式解锁后空原因成功及相邻生命周期/补审路径；4个前端suite39项定向、tsc、Biome、Go单元/vet通过。后端check:server完整通过含漏洞扫描；check:fast两项无关高并发超时，降低maxWorkers至16完整复跑307文件1681项通过，未跳过或修改测试规则。工作区无其他未提交代码。

### Git Commits

| Hash | Message |
|------|---------|
| `1e378866` | fix: 反结案原因选填并禁止审核锁定订单反结案 |
| `5d30e402` | docs: 记录反结案锁校验完整验收结果 |

### Status

[OK] **Completed**


## Session 164: 连续新增海运分单入口与号码就近填写
<!-- trellis-session: v=2 fp=bac5d9db0d43d7c0 -->

**Date**: 2026-10-26
**Task**: 连续新增海运分单入口与号码就近填写
**Branch**: `codex/sea-continuous-house-order`

### Summary

从已有HOUSE主单连续新增分单，只带入共享身份和明确航程；HBL在MBL旁唯一填写，本票资料及草稿保持独立。完整前端门禁和只读浏览器验收通过。

### Main Changes

- 新增同组织权限控制的主单建分单入口，作废主单隐藏入口，来源审核锁不影响新票。
- 规范sourceOrderId资源身份，隔离普通新票与不同来源的草稿、表单、幂等键和迟到响应。
- 保留HBL必填、批次重复、单证模式切换与错误揭示；拆分重型单证测试并同步规范。

### Git Commits

| Hash | Message |
|------|---------|
| `b4d55ad7` | feat: 支持从共享主单连续新增海运分单 |

### Testing

- [OK] pnpm run check:web通过：317文件通过、1文件跳过；1726测试通过、12测试跳过。
- [OK] 浏览器7项断言通过，无页面错误和业务写请求；定向、Biome、Ant Design lint、TypeScript及diff检查通过。

### Status

[OK] **Completed**

### Next Steps

- 代码已在codex/sea-continuous-house-order提交；原共享checkout切回main后，归档与会话记录在隔离worktree完成。


## Session 165: 复审海运分单分支并安全合并主干
<!-- trellis-session: v=2 fp=5832f3dc51780152 -->

**Date**: 2026-10-26
**Task**: 复审海运分单分支并安全合并主干
**Branch**: `main`

### Summary

独立前后端复审未发现新增业务缺陷；修复确认框卸载测试的动画事件时序，完整前后端门禁通过后快进合并main并安全删除本次功能分支。

### Main Changes

- 复审连续分单、共享航程、已完结内容编辑与审核锁/反结案守卫的完整待合并范围。
- 确认全部功能提交已被main保留后，使用git branch -d删除codex/sea-continuous-house-order，清理临时worktree；未推送远端。

### Git Commits

| Hash | Message |
|------|---------|
| `7086b28e` | test: 等待确认框退出动画就绪后验证卸载 |
| `a58f0549` | docs: 记录海运分单复审与安全合并结果 |

### Testing

- [OK] 修复后check:web通过：317文件、1726项测试通过，1文件/12项既有跳过。
- [OK] check:server通过：Go单元测试、vet、分层、Proto lint与漏洞扫描通过，无未处理可达漏洞。

### Status

[OK] **Completed**

### Next Steps

- 当前检出main，代码与审查记录均已提交。


## Session 166: 完成全球港口中文种子与离线同步
<!-- trellis-session: v=2 fp=25b6961f5c1cf167 -->

**Date**: 2026-10-27
**Task**: 完成全球港口中文种子与离线同步
**Branch**: `codex/port-chinese-seeds`

### Summary

全量17,524港口中文归档（已有7,357、编制10,167），严格配套校验、双源指纹及中文拼音同步；完整后端门禁和独立PG验证通过。

### Main Changes

- 新增完整港口中文CSV、来源与译名说明，核对中国日本汉字地名及世界长尾语义。
- sync-unlocode离线预览、完整身份校验与联合指纹，已有港口补中文并刷新检索键。

### Git Commits

| Hash | Message |
|------|---------|
| `9e4af6df` | feat: 补齐全球港口中文种子并接入离线同步 |

### Testing

- [OK] 完整check:server通过；命令配套17524条硬验收、真实PG专项无Skip通过；独立库全量首次及重复导入、最终中英文与指纹逐项一致。
- [OK] check:fast前端两轮分别有既有主单异步用例失败；对应文件定向8/8、6/6通过，未将全栈门禁记为绿色。

### Status

[OK] **Completed**

### Next Steps

- 用户需要初始化时运行pnpm run sync:all或sync:unlocode；本次未对业务库apply，长尾译名可按业务反馈修订。


## Session 167: 船代公共主数据与订单选择交付
<!-- trellis-session: v=2 fp=bd2cebb1f8f2413d -->

**Date**: 2026-10-27
**Task**: 船代公共主数据与订单选择交付
**Branch**: `codex/shipping-agent-masterdata`

### Summary

船代29条离线同步接入sync:all；系统管理维护，订单引用公共字典；隔离PostgreSQL和check:fast通过，独立review完成。

### Git Commits

| Hash | Message |
|------|---------|
| `e68326f0` | feat: 接入船代公共主数据与订单引用 |
| `1106f9bf` | feat: 增加船代维护页与海运订单选择 |
| `66d47948` | docs: 记录船代同步与引用契约 |

### Status

[OK] **Completed**


## Session 168: 修复海运主分单独立保存及只读状态
<!-- trellis-session: v=2 fp=c8636035707104b7 -->

**Date**: 2026-10-28
**Task**: 修复海运主分单独立保存及只读状态
**Branch**: `main`

### Summary

主单和分单保存直接应用各自写入响应，保留另一单证草稿及原版本；切换或清空客户清理旧岗位人员；补齐订舱号、合约号、工厂与模式切换的只读状态；空单证响应提示中文错误。相关回归、类型检查、修改文件 Biome 与 diff 检查通过。最终 check:web 为 1764 通过、1 失败、12 跳过，唯一失败为未改动的密码重置冷却用例，该文件单独复跑 3 项全部通过；此前航程选择用例全量超时后单独及最终全量均通过。未修改无关测试或业务代码。

### Git Commits

| Hash | Message |
|------|---------|
| `d642f11a` | fix: 保留海运单证独立草稿并补齐详情只读状态 |

### Status

[OK] **Completed**


## Session 169: 海运订单整页统一保存并完成审查合并
<!-- trellis-session: v=2 fp=1f974c5669e41faa -->

**Date**: 2026-10-28
**Task**: 海运订单整页统一保存并完成审查合并
**Branch**: `main`

### Summary

订单资料、修改过的MBL和本票HBL通过现有UpdateOrder同事务保存，移除局部保存入口并提供吸底保存。独立review已修复隐藏字段、草稿原版本、专用动作并发刷新、异步身份隔离、默认值与中文错误等问题，无剩余P1/P2。最终check:web 329文件1783项通过，check:server全部通过（含vet和漏洞扫描）；专用PostgreSQL事务/冲突/幂等/共享门禁回归实际通过，Chromium验证一次写入、连续版本、409保留输入与0页面异常。已合并main并用git branch -d安全删除任务分支；随后按用户授权推送并核对远端。

### Git Commits

| Hash | Message |
|------|---------|
| `f02d73d9` | feat: 海运订单整页统一原子保存主分单 |
| `1728ecd4` | merge: 合并海运订单整页统一保存 |

### Status

[OK] **Completed**


## Session 170: 海运出口录单可靠性修复与验收
<!-- trellis-session: v=2 fp=d14a533b2f1f87bf -->

**Date**: 2026-10-29
**Task**: 海运出口录单可靠性修复与验收
**Branch**: `fix/sea-export-entry-experience`

### Summary

修复客户人员异步覆盖与草稿遗漏，完善新建提交冻结和未知结果刷新确认，增加吸底创建栏与ETA冲突保留；独立review闭环，最终前端1803项与浏览器5项通过。

### Main Changes

- 记录复合控件禁用继承及锁定来源跨记录新建的例外契约。

### Git Commits

| Hash | Message |
|------|---------|
| `8248f20b` | fix: 保留海运录单手动输入并完整恢复人员草稿 |
| `0a979ace` | fix: 海运新建冻结提交并恢复待确认创建结果 |
| `79a0ef48` | test: 验收海运录单恢复流程并记录审查契约 |

### Testing

- [OK] pnpm run check:web：333文件通过、1跳过；1803项通过、12项跳过。
- [OK] Playwright海运录入体验5项整轮通过；API均stub，无真实业务写入。

### Status

[OK] **Completed**

### Next Steps

- 按用户授权合并main、安全删除本地工作分支并push；真实操作系统中文输入法仍保留人工验收限制。
