# Journal - mam15mon (Part 1)

> AI development session journal
> Started: 2026-08-31

---



## Session 1: 引入 Trellis 并规划提成增强任务
<!-- trellis-session: v=2 fp=11a98ea47a4696ab -->

**Date**: 2026-08-31
**Task**: 引入 Trellis 并规划提成增强任务
**Package**: server
**Branch**: `main`

### Summary

初始化 Trellis 工作流（--zcode）；注册 server 包并把 AGENTS.md 规范沉淀进 spec/server/backend 与 spec/web/frontend；归档 bootstrap 任务；按提成管理增强计划建立父任务与三个阶段子任务（均处于 planning，待阶段0口径冻结）

### Git Commits

| Hash | Message |
|------|---------|
| `35401ee` | docs: 建立提成管理增强任务规划 |

### Status

[OK] **Completed**


## Session 2: 治理提成仓储事务客户端直连
<!-- trellis-session: v=2 fp=ed647ee548c7bad5 -->

**Date**: 2026-08-31
**Task**: 治理提成仓储事务客户端直连
**Package**: server
**Branch**: `main`

### Summary

完成 finance_commission.go 专项审计，将员工列表、候选列表、规则列表和调整幂等查询的 6 处 r.data.db 直连统一迁移到 Data.client(ctx)；保持分页、错误映射和无锁只读语义；补充已结束事务上下文回归测试，并通过全量 Go test 与 vet。

### Git Commits

| Hash | Message |
|------|---------|
| `f134d88` | fix: 修正提成生成锁顺序与汇率日期校验 |
| `119bf34` | test: 覆盖提成读取事务上下文失效 |

### Status

[OK] **Completed**


## Session 3: 验收提成导出阶段 2
<!-- trellis-session: v=2 fp=9f11733c5b37fe30 -->

**Date**: 2026-08-31
**Task**: 验收提成导出阶段 2
**Package**: server
**Branch**: `main`

### Summary

完成提成 JSON 导出、独立权限、10000 行上限、稳定分批与成功导出审计；经独立复核与全量质量门验收通过。

### Git Commits

| Hash | Message |
|------|---------|
| `9411620` | feat: 增加提成导出 |

### Status

[OK] **Completed**


## Session 4: 修复提成静态路由遮蔽
<!-- trellis-session: v=2 fp=fdc98b4cf7088a99 -->

**Date**: 2026-08-31
**Task**: 修复提成静态路由遮蔽
**Package**: server
**Branch**: `main`

### Summary

调整提成静态 RPC 在参数路由前注册，增加真实 Kratos Router 分发回归测试，并沉淀静态路由优先规范。

### Git Commits

| Hash | Message |
|------|---------|
| `38bd2d3` | fix: 修复提成静态路由遮蔽 |

### Status

[OK] **Completed**


## Session 5: 完成提成 CNY 前端展示与导出
<!-- trellis-session: v=2 fp=c746113ea27dd917 -->

**Date**: 2026-08-31
**Task**: 完成提成 CNY 前端展示与导出
**Package**: server
**Branch**: `main`

### Summary

完成归属月份筛选、列表与详情 CNY 双口径展示、预览汇率依据、权限控制的安全 CSV 导出；独立检查 PASS，54 个测试文件共 178 项通过。

### Git Commits

| Hash | Message |
|------|---------|
| `f255bfd` | feat: 增加提成 CNY 展示 |

### Status

[OK] **Completed**


## Session 6: 修复提成账单多行加锁顺序
<!-- trellis-session: v=2 fp=f5c551359864763b -->

**Date**: 2026-08-31
**Task**: 修复提成账单多行加锁顺序
**Package**: server
**Branch**: `main`

### Summary

使用 Agy gemini-3.7-flash-high 实施账单主键顺序加锁，独立 trellis-check 修复集成测试清理与并发栅栏；SQL 门禁、真实 PostgreSQL 并发测试、Go 全量测试和 vet 全部通过。

### Git Commits

| Hash | Message |
|------|---------|
| `a16e289` | fix: 固定提成账单多行加锁顺序 |

### Status

[OK] **Completed**


## Session 7: 补齐提成 PostgreSQL 事务集成验证
<!-- trellis-session: v=2 fp=109ff488b4bfb419 -->

**Date**: 2026-09-01
**Task**: 补齐提成 PostgreSQL 事务集成验证
**Package**: server
**Branch**: `main`

### Summary

使用 Agy High 实施并由独立 Trellis 检查补强，新增提成创建真实 PostgreSQL 成功与三类失败事务测试，验证审计、回滚、提交后重读及来源数据完整性；同步归档 PRD 验收记账并完成任务归档。

### Main Changes

- 新增提成创建真实 PostgreSQL 事务集成测试，覆盖成功、汇率失败、保存失败和审计失败。
- 补强审计详情、未提交事务证据、提交后普通上下文重读和失败后来源数据可读性断言。
- 同步阶段 1/2/3 与静态路由归档 PRD 验收状态。

### Git Commits

| Hash | Message |
|------|---------|
| `33445813` | test: 补齐提成 PostgreSQL 事务集成验证 |
| `1cd7904c` | chore(task): archive 08-31-commission-postgres-transaction-test |

### Testing

- [OK] 隔离 PostgreSQL 临时库四个子用例全部通过，测试后数据库和角色残留均为 0。
- [OK] go -C server test ./... -count=1、go -C server vet ./...、Trellis validate、git diff --check 全部通过。

### Status

[OK] **Completed**


## Session 8: 完成财务全链路测试覆盖审计
<!-- trellis-session: v=2 fp=01041894cd962c6f -->

**Date**: 2026-09-01
**Task**: 完成财务全链路测试覆盖审计
**Package**: server
**Branch**: `main`

### Summary

审计多币种汇率、订单费用、账单开票、资金核销与提成的 E1-E5 测试证据；确认本位币应收 HTTP 长链路存在，但外币连续链路、提成支付、PostgreSQL CI 门禁和权限/双组织负向验收仍是上线前 P1 缺口。

### Main Changes

- 新增财务全链路证据矩阵、跨阶段断点、风险分级和最小补测路线图。
- 沉淀专用 PostgreSQL 测试不得以 SKIP 充当通过的质量规范。

### Git Commits

| Hash | Message |
|------|---------|
| `dfac79b1` | docs: 完成财务全链路测试覆盖审计 |

### Testing

- [OK] Go 四包测试、go vet、web tsc、54 个前端测试文件与 178 项测试通过。
- [OK] 独立 trellis-check PASS，并反证五个核心 PostgreSQL 顶层测试在无专用变量时全部 SKIP。

### Status

[OK] **Completed**


## Session 9: 完成外币财务全链路验收
<!-- trellis-session: v=2 fp=15e9488f16ddc07d -->

**Date**: 2026-09-02
**Task**: 完成外币财务全链路验收
**Package**: server
**Branch**: `main`

### Summary

在一次性 CNY/USD PostgreSQL 环境补齐从系统汇率、订单费用、账单、开票、收款、核销到提成 CNY 快照的连续验收，并以故障注入验证资源与进程组安全清理。

### Main Changes

- 新增 USD 本位币、EUR 业务币的连续 API 验收及一次性双库编排器
- 修正 CNY 应收、应付和 Playwright 验收夹具与精确状态断言
- 补齐 SIGTERM、超时、孤儿进程组和执行/清理双失败的生命周期自测

### Git Commits

| Hash | Message |
|------|---------|
| `1c13e9df` | test: 补齐外币财务全链路验收 |

### Testing

- [OK] 完整 disposable 验收通过：PostgreSQL PASS=10/SKIP=0/FAIL=0，Playwright 1 passed，外币金额与快照断言全部通过
- [OK] go test ./...、go vet ./...、web lint/tsc、Node 语法及 Trellis validate 全部通过
- [OK] 独立 trellis-check PASS：P0=0、P1=0、P2=0；最终数据库、角色和固定端口零残留

### Status

[OK] **Completed**

### Next Steps

- 后续独立任务可覆盖提成 MarkPaid、应付外币链、低权限与双组织隔离、CI 强制一次性验收


## Session 10: 修复内置海运服务类型读取
<!-- trellis-session: v=2 fp=0ff4911a2fb5b89d -->

**Date**: 2026-09-02
**Task**: 修复内置海运服务类型读取
**Package**: web
**Branch**: `main`

### Summary

确认 BOOKING 等 19 个服务类型已经由应用自动初始化，修复订单前端用字符串比较 OpenAPI 数字枚举导致的主数据误报缺失。

### Main Changes

- 订单服务类型、货物类别、箱型和地区统一消费生成的 MasterDataKind 数字常量
- 移除订单模块字符串枚举第二真相并收敛地区查询裸数字
- 补充真实数字 API 响应、19 个服务类型、后端名称和 BOOKING 缺项回归测试

### Git Commits

| Hash | Message |
|------|---------|
| `729a5a2d` | fix: 修复订单内置主数据枚举读取 |

### Testing

- [OK] 运行中开发服务返回 19 个服务类型，BOOKING kind=8 且类型为 number、source=system、enabled=true
- [OK] Vitest 54 个文件、180 项测试通过；TypeScript 与 393 文件 Biome 检查通过
- [OK] 独立 trellis-check PASS：P0=0、P1=0、P2=0

### Status

[OK] **Completed**

### Next Steps

- 用户刷新或重启前端开发服务后验证海运出口新建订单不再误报 BOOKING 缺失


## Session 11: 完成海运出口共享主单基础
<!-- trellis-session: v=2 fp=47f90d19122ac5b9 -->

**Date**: 2026-09-02
**Task**: 完成海运出口共享主单基础
**Package**: server
**Branch**: `main`

### Summary

用 agy 实施并经两轮独立 trellis-check 收敛阶段 1：建立共享 MBL、运输执行与当前/历史成员关系，海运订单首次保存强制主单号和签发主体，候选显式确认并重验版本/航程；旧单证入口收敛为可选多 HBL，补事务、下游事实门禁、迁移非空保护、真实 PostgreSQL 并发测试和前端交互。开发库中的测试残留已按授权清空，完整迁移与管理员初始化成功。

### Git Commits

| Hash | Message |
|------|---------|
| `294eb6ef` | feat: 建立海运出口共享主单基础 |

### Status

[OK] **Completed**


## Session 12: 完成海运出口主分单内容阶段
<!-- trellis-session: v=2 fp=155986cfd7485ab8 -->

**Date**: 2026-09-03
**Task**: 完成海运出口主分单内容阶段
**Package**: server
**Branch**: `main`

### Summary

完成海运出口单证三态、共享 MBL 内容、真实多 HBL、唯一签发主体、乐观锁与审计事务、默认展开页面；Agy 实施并经独立 Trellis 检查和真实 PostgreSQL 验证通过。

### Git Commits

| Hash | Message |
|------|---------|
| `8b2f9519` | feat: 增加海运出口主分单内容 |

### Status

[OK] **Completed**


## Session 13: 完成海运出口箱货定量分配
<!-- trellis-session: v=2 fp=a3935a71fcd007d4 -->

**Date**: 2026-09-03
**Task**: 完成海运出口箱货定量分配
**Package**: server
**Branch**: `main`

### Summary

完成海运出口箱货分配阶段设计、Agy Gemini 3.8 Flash 实施、独立检查修正、全量与真实 PostgreSQL 验证，并重建无历史数据的开发库后应用阶段 3 迁移。

### Git Commits

| Hash | Message |
|------|---------|
| `949b7247` | feat: 增加海运出口箱货定量分配 |
| `3cdbbdb4` | chore: 激活海运箱货分配任务 |
| `e64c646f` | docs: 升级 Agy 默认模型至 Gemini 3.8 Flash |
| `63b1e042` | docs: 完成海运箱货分配实施设计 |
| `1747245d` | docs: 明确箱货分配反馈与提单填充 |
| `d4a259bb` | docs: 确定箱货分配确认时点 |
| `b8780b70` | docs: 记录海运箱货分配现状证据 |

### Status

[OK] **Completed**


## Session 14: 完成海运出口拆票与改配
<!-- trellis-session: v=2 fp=72e02eb229fb3494 -->

**Date**: 2026-09-04
**Task**: 完成海运出口拆票与改配
**Package**: server
**Branch**: `main`

### Summary

完成阶段4部分拆票、整体改配、附件资产引用、不可变事件、严格版本与目标契约、前端精确十进制交互；Agy High 实施修正后经独立 trellis-check 复验，真实 PostgreSQL 26 子测试、全量 Go/Web、漏洞扫描与生成幂等均通过，并重建空开发库验证正式 CHECK 约束。

### Git Commits

| Hash | Message |
|------|---------|
| `c2ca2f98` | feat: 完成海运出口拆票与改配 |

### Status

[OK] **Completed**


## Session 15: 完成海运出口单证版本与换单
<!-- trellis-session: v=2 fp=1bb65d277d1df95d -->

**Date**: 2026-09-04
**Task**: 完成海运出口单证版本与换单
**Package**: server
**Branch**: `main`

### Summary

完成钉钉订单解锁审批、海运 MBL/HBL 单改作废与 Switch 不可变历史，修复并发锁序、财务门禁和终态绕过问题，通过服务端、前端、真实 PostgreSQL、迁移、生成幂等及漏洞检查，并归档任务。

### Git Commits

| Hash | Message |
|------|---------|
| `4761ee45` | feat: 接入钉钉订单解锁审批 |
| `d53bef83` | feat: 增加海运提单改单作废与换单 |
| `af0f84fe` | docs: 沉淀海运提单变更历史契约 |

### Status

[OK] **Completed**


## Session 16: 完成全业务类型订单锁定与解锁
<!-- trellis-session: v=2 fp=ecab30115f7f90fd -->

**Date**: 2026-09-05
**Task**: 完成全业务类型订单锁定与解锁
**Package**: server
**Branch**: `feat/universal-order-lock`

### Summary

将订单锁扩展到 SE/SI/AE/AI/LAND/RAIL 六种业务类型，按类型隔离锁权限与解锁审批候选；锁后统一阻断订单资料和费用写入，保留费用读取与账单生成；SE 保留 MBL/HBL 不可变快照，非 SE 不创建海运快照；前端接入通用锁控件与失败关闭；补全真实 PostgreSQL 六类型并发、升级回填和外键不可变验证。

### Git Commits

| Hash | Message |
|------|---------|
| `876da3d4` | docs: 规划全业务类型订单锁 |
| `ca237447` | feat: 扩展全业务订单锁契约与权限 |
| `cb89cb30` | feat: 统一全业务订单锁与解锁审批 |
| `22f8261c` | feat: 接入通用订单锁前端交互 |
| `4aa98a74` | fix: 保持订单锁历史外键不可变 |

### Status

[OK] **Completed**


## Session 17: 海运出口业务完整度审计
<!-- trellis-session: v=2 fp=3fb181ca6042ab5c -->

**Date**: 2026-09-05
**Task**: 海运出口业务完整度审计
**Package**: server
**Branch**: `feat/universal-order-lock`

### Summary

按内部货代业务、操作、单证和财务管理平台定位完成海运出口跨层就绪度审计；确认主体可受限上线，识别结案门禁、真实单证状态、共享费用、附件、旧分单入口及拆票并发错误语义等缺口，并记录全量与 PostgreSQL 验证结果。

### Git Commits

| Hash | Message |
|------|---------|
| `ea4e30bc` | docs: 规划海运出口业务完整度审计 |
| `5c02ff6a` | docs: 完成海运出口上线就绪度审计 |

### Status

[OK] **Completed**


## Session 18: 修订海运出口审计业务口径
<!-- trellis-session: v=2 fp=d95b79f5d11aaefd -->

**Date**: 2026-09-05
**Task**: 修订海运出口审计业务口径
**Package**: server
**Branch**: `feat/universal-order-lock`

### Summary

按用户确认的人工 UI 流程重新评级海运出口上线就绪度：明确已放单为流程终点、共享费用线下计算后逐票录入、关键文件由外部受控库保存且不需要外部提交；统一修订审计 PRD、能力盘点、能力矩阵和就绪度报告，结论调整为批准边界内可以正式使用，同时保留四项真实 P1 故障入口与原始测试证据。独立复核与文档差异检查通过。

### Git Commits

| Hash | Message |
|------|---------|
| `247c9b79` | docs: 规划海运出口审计口径修订 |
| `fcaa7aa9` | docs: 按人工流程修订海运出口上线结论 |

### Status

[OK] **Completed**


## Session 19: 修复海运出口四项P1问题
<!-- trellis-session: v=2 fp=65b2e30230711c03 -->

**Date**: 2026-09-05
**Task**: 修复海运出口四项P1问题
**Branch**: `feat/universal-order-lock`

### Summary

支持放货记录关联真实海运MBL/HBL及HBL关联记录原子删除，修正SE单证入口与箱货删除版本，并稳定并发拆票409语义；真实PostgreSQL、Go/Web全量检查、构建和生成幂等均通过。

### Git Commits

| Hash | Message |
|------|---------|
| `9706cf2b` | feat: 支持放货记录关联真实海运单证 |
| `8b1e49df` | fix: 修正海运单证入口和箱货删除版本 |
| `0c64540d` | fix: 统一并发拆票版本冲突语义 |

### Status

[OK] **Completed**


## Session 20: 修复非安全上下文下 crypto.randomUUID 缺失导致的前端请求与登录报错
<!-- trellis-session: v=2 fp=040da8f73f0001c1 -->

**Date**: 2026-09-05
**Task**: 修复非安全上下文下 crypto.randomUUID 缺失导致的前端请求与登录报错
**Branch**: `main`

### Summary

针对内网/公网 IP 纯 HTTP 访问时浏览器禁用 crypto.randomUUID 的问题，统一封装 generateUUID 并替换所有业务与拦截器调用。

### Main Changes

- 新增 web/src/utils/uuid.ts 跨环境安全 UUID v4 实现，带单元测试
- 在 web/src/requestErrorConfig.ts 中将 X-Request-ID 替换为 generateUUID()
- 统一 orders、finance、settings 11 处业务组件中的 randomUUID 调用

### Git Commits

| Hash | Message |
|------|---------|
| `d92fcf8f` | fix(web): 统一 UUID 生成以兼容非安全上下文环境访问 |

### Testing

- [OK] 单元测试 web/src/utils/uuid.test.ts 4/4 通过
- [OK] pnpm --dir web tsc 与 lint 校验全部通过
- [OK] 通过内网 IP http://10.180.10.50:8001 进行登录接口测试正常返回 200

### Status

[OK] **Completed**


## Session 21: 在海运出口配舱信息中增加只读关联分单号展示
<!-- trellis-session: v=2 fp=59d1bc29b68ce72b -->

**Date**: 2026-09-05
**Task**: 在海运出口配舱信息中增加只读关联分单号展示
**Branch**: `main`

### Summary

在 SeaTransportSection 中增加 SeaAssociatedHouseBillsField，实现分单直单/多分单标签/未录入的三态只读联动展示，避免就地编辑

### Git Commits

| Hash | Message |
|------|---------|
| `f8547480` | feat(web): 在海运出口配舱信息中增加只读关联分单号展示 |

### Status

[OK] **Completed**


## Session 22: 统一全站 TagsView 页面复用规则
<!-- trellis-session: v=2 fp=028d4638b24e2bf6 -->

**Date**: 2026-09-05
**Task**: 统一全站 TagsView 页面复用规则
**Branch**: `main`

### Summary

将 TagsView 页签从完整 URL 改为按稳定菜单入口复用，拆分 key 与 path 并保留 search/hash，门禁拦截迟到动态标题事件

### Git Commits

| Hash | Message |
|------|---------|
| `fc22bc91` | fix(web): 统一菜单内部页面页签复用 |

### Status

[OK] **Completed**


## Session 23: 统一订单页面面包屑导航与页签复用收紧
<!-- trellis-session: v=2 fp=b6603cd8ea57cd6a -->

**Date**: 2026-09-05
**Task**: 统一订单页面面包屑导航与页签复用收紧
**Branch**: `main`

### Summary

统一海运、空运订单新建、详情、费用录入与拆票页面的面包屑层级规范与 OrderPageHeader；修复纯重定向路由重复建签、动态订单标题防串号及逆序响应拦截、收紧页签归组正则白名单并补齐右键菜单与键盘无障碍测试。

### Git Commits

| Hash | Message |
|------|---------|
| `9f2d6bd7` | fix(web): 统一订单页面面包屑导航并收紧页签复用 |

### Status

[OK] **Completed**


## Session 24: 修复订单切换跨单状态污染
<!-- trellis-session: v=2 fp=2eed514aa0a232f3 -->

**Date**: 2026-09-05
**Task**: 修复订单切换跨单状态污染
**Branch**: `main`

### Summary

联合审查并修复 TagsView 复用页面下的订单费用、账单工作台和拆票改配动作竞态；补充同实例切单与迟到响应测试，66 项关联测试及前端类型检查通过。

### Git Commits

| Hash | Message |
|------|---------|
| `52ccb139` | fix(web): 隔离订单切换时的费用与动作状态 |

### Status

[OK] **Completed**


## Session 25: 修复移动端页面布局与顶栏显示异常
<!-- trellis-session: v=2 fp=3a1d1827ed2afac9 -->

**Date**: 2026-09-06
**Task**: 修复移动端页面布局与顶栏显示异常
**Branch**: `main`

### Summary

排查并修复手机等移动窄屏下顶栏 180px 错位留白、标题单字折行溢出遮挡及桌面下拉菜单挤压问题，完善多视口响应式体验。

### Main Changes

- 在 web/src/global.less 中重置移动窄屏（<= 768px）下顶栏定位为 left: 0 与 width: 100%，消除左侧 180px 空白
- 为 .roncin-header-title-text 添加单行截断与省略号规则，防止标题字数较多时单字竖排溢出
- 移动端隐藏顶栏冗余的 HeaderMenus 快捷下拉，并在极窄屏（<= 480px）下收起用户文字名仅保留头像徽标
- 将 PageHeaderShell 行内 height: 52 调整为 minHeight: 52，保证手机换行时高度自适应不溢出

### Git Commits

| Hash | Message |
|------|---------|
| `3c3f4929` | fix(web): 修复移动窄屏视口下顶栏错位留白与标题竖排溢出 |
| `b2fe7897` | chore(task): archive 09-06-fix-mobile-layout-display |

### Testing

- [OK] 运行 Playwright 多视口自动化验证脚本，覆盖 iPhone 13、iPhone SE、Pixel 7 与 Desktop 1280，断言移动与桌面端无样式回归
- [OK] 运行 web 端全量 74 个测试套件，276 个用例全部通过
- [OK] 通过 tsc --noEmit、biome lint 与 git diff --check 质量门禁

### Status

[OK] **Completed**


## Session 26: 精简订单列表头部与快捷切签
<!-- trellis-session: v=2 fp=4f274122b7d3933b -->

**Date**: 2026-09-06
**Task**: 精简订单列表头部与快捷切签
**Branch**: `main`

### Summary

移除订单列表模板的面包屑与顶部12项状态快捷筛选切签卡片，使海运出口订单列表保持纯白高密度清爽视图，保证全部单测与全量质量门禁通过并完成端到端视觉验收

### Git Commits

| Hash | Message |
|------|---------|
| `539df794` | refactor(web): 移除订单列表页面的冗余面包屑与快捷状态切签 |

### Status

[OK] **Completed**


## Session 27: 优化新建订单页面主数据加载体验
<!-- trellis-session: v=2 fp=abd3fcc9fdb2041d -->

**Date**: 2026-09-06
**Task**: 优化新建订单页面主数据加载体验
**Branch**: `main`

### Summary

实现组织隔离的会话级主数据与人员选项缓存，海空运按需模式加载，以及纯白高密度分节骨架屏占位与错误重试

### Git Commits

| Hash | Message |
|------|---------|
| `249d30d9` | feat(web): 优化新建订单主数据加载体验、分节骨架占位与组织会话缓存 |

### Status

[OK] **Completed**


## Session 28: 取消海运共享费用分摊并收尾父任务
<!-- trellis-session: v=2 fp=f4cff12426da0d76 -->

**Date**: 2026-09-06
**Task**: 取消海运共享费用分摊并收尾父任务
**Branch**: `main`

### Summary

业务确认当前没有真实共享费用案例或现行分摊规则，取消共享费用分摊阶段；父任务调整为五个已完成阶段并完成归档。

### Git Commits

| Hash | Message |
|------|---------|
| `d96c400f` | docs: 取消海运共享费用分摊阶段 |

### Status

[OK] **Completed**


## Session 29: 修复订单缓存组织隔离
<!-- trellis-session: v=2 fp=5af9b9f618caccb3 -->

**Date**: 2026-09-06
**Task**: 修复订单缓存组织隔离
**Branch**: `fix/order-cache-org-isolation`

### Summary

修复订单新建、详情与列表资源在组织切换时的异步搜索和错误状态隔离；空关键字复用当前组织首批缓存，目标浏览器验收与前端完整门禁通过，并补充组织级异步联想规范。

### Git Commits

| Hash | Message |
|------|---------|
| `a70c39e0` | fix(web): 完善订单缓存复用与组织隔离 |

### Status

[OK] **Completed**


## Session 30: 简化海运出口主单签发方录入
<!-- trellis-session: v=2 fp=3a047541b01dea31 -->

**Date**: 2026-09-06
**Task**: 简化海运出口主单签发方录入
**Branch**: `main`

### Summary

海运出口业务只维护必填的船公司与 MBL 主单号，系统统一维护 Order、运输执行和 MBL 签发主体；同步简化拆票、改配与详情界面，保留共享主单保护和 HBL 独立签发规则，并通过完整前后端门禁。

### Git Commits

| Hash | Message |
|------|---------|
| `047b3e79` | feat: 简化海运出口主单签发方录入 |

### Status

[OK] **Completed**


## Session 31: 海运出口船公司切换为 ShippingLine
<!-- trellis-session: v=2 fp=f0e62b7adc45db1b -->

**Date**: 2026-09-06
**Task**: 海运出口船公司切换为 ShippingLine
**Branch**: `main`

### Summary

将 SE 订单、运输执行与共享 MBL 的船公司身份统一为 ShippingLine，移除 Partner carrier 角色和费用结算回退；补齐严格迁移、候选锁内校验、前端搜索与历史回显，并通过完整 Web/Server、构建及真实 PostgreSQL 验证。

### Git Commits

| Hash | Message |
|------|---------|
| `834ff883` | feat: 海运出口船公司改用航运公司主数据 |

### Status

[OK] **Completed**


## Session 32: 实施海运主分单简化阶段2：Proto、单值HBL与查询收敛
<!-- trellis-session: v=2 fp=a213e5f45f8f0ce9 -->

**Date**: 2026-09-07
**Task**: 实施海运主分单简化阶段2：Proto、单值HBL与查询收敛
**Branch**: `main`

### Summary

完成阶段2所有内容：收敛Proto契约与生成物；重构Order/SeaMasterBill/SeaDocument的service、biz、data实现；增加Booking No.与客户业务号/Booking/MBL三维度同批订单查询；移除废弃的Add/Remove HBL与UNDETERMINED调用链；通过全量服务端测试与代码检查。

### Git Commits

| Hash | Message |
|------|---------|
| `66389349` | refactor: 收敛海运订单单值分单契约 |

### Status

[OK] **Completed**


## Session 33: 完成海运主单航次解耦与单证共享箱全链路实现与门禁通过
<!-- trellis-session: v=2 fp=5b94e3ba5afd8a06 -->

**Date**: 2026-09-07
**Task**: 完成海运主单航次解耦与单证共享箱全链路实现与门禁通过
**Branch**: `main`

### Summary

完成海运主分单模型简化任务：修复增量迁移脚本与单测，对齐并生成 Wire/PB/Ent/Web-Client 契约，实现 MBL 与实际航次解耦、单证变更模式切换与外部确认、SharedContainer/Allocation 跨订单共享箱模型，通过全量后端 check:server 与前端 tsc/biome/vitest 测试并归档任务。

### Git Commits

| Hash | Message |
|------|---------|
| `372966b0` | refactor: 解耦海运主单航次并收敛单证变更与共享箱模型 |
| `020dfe43` | feat: 增加共享航次统一调整入口 |
| `09c01592` | feat: 为海运单证变更记录外部确认 |
| `871650c8` | refactor: 删除旧海运箱货分配页面 |
| `0fc306ea` | feat: 为海运改配记录外部确认 |
| `5ae8e62b` | refactor: 删除海运换单前端流程 |
| `c7c63a10` | feat: 增加海运外部确认表单 |
| `092fdef8` | refactor: 移除旧海运箱货分配入口 |
| `0eda90d9` | feat: 展示海运同批关联订单 |
| `dc5cfa27` | feat: 增加海运订单聚合号码筛选 |
| `c8409caa` | refactor: 收敛海运订单单值分单表单 |

### Status

[OK] **Completed**


## Session 34: 海运分单模型简化：重构拆票链路并实现共享箱工作台
<!-- trellis-session: v=2 fp=a0d616136996746a -->

**Date**: 2026-09-07
**Task**: 海运分单模型简化：重构拆票链路并实现共享箱工作台
**Branch**: `main`

### Summary

重构拆票 Proto 与领域契约，彻底移除旧箱货分配入参并支持货物件重尺切分与新 HBL 录入；完成拆票 Preview 与 Execute 逻辑及守恒校验；实现跨订单共享箱工作台并补齐前后端定向测试与数据库集成验证。

### Main Changes

- 契约重构：更新 sea_order_change.proto，移除旧 cargo_allocation_version 与 house_bill_ids，引入货物件重尺分配与新 HBL
- 业务与仓储：重写 ExecuteSplit/PreviewSplit，支持独占箱移动、件重尺切分、共享箱分配跨票迁移与残余删除
- 前端拆票：重构 orders/split.tsx，支持逐行货物件重尺输入、独占箱归属与实时守恒校验
- 前端工作台：新建 SeaSharedContainerDrawer.tsx 支持跨订单共享箱货物分配、快捷填满、保存草稿、确认生效与撤回
- 测试与迁移：修复 20260907140000 迁移脚本；修复 sea_document_test 集成测试；补齐前端单测

### Git Commits

| Hash | Message |
|------|---------|
| `3e0dabf6` | feat: 重构海运拆票闭环并实现跨订单共享箱工作台 |

### Testing

- [OK] go -C server test ./internal/biz/... ./internal/service/...
- [OK] RONCIN_INTEGRATION_DATABASE_SOURCE=... go -C server test -v -count=1 -run TestSeaDocumentPostgresIntegration ./internal/data/...
- [OK] pnpm --dir web tsc --noEmit
- [OK] pnpm --dir web exec vitest run src/pages/orders/components/drawers/SeaSharedContainerDrawer.test.tsx src/pages/orders/split.test.tsx

### Status

[OK] **Completed**

### Next Steps

- 重写并恢复 3 份被 //go:build ignore 排除的 Data 集成测试 (sea_cargo_allocation_test, sea_document_change_integration_test, sea_order_change_integration_test)
- 运行全套代码门禁 (check:server, check:web) 并归档任务


## Session 35: 海运分单模型简化阶段 5/6 收尾：拆票资格、共享箱守恒与授权锚点五轮复核修复
<!-- trellis-session: v=2 fp=56cb3d80cf4ba6e5 -->

**Date**: 2026-09-08
**Task**: 海运分单模型简化阶段 5/6 收尾：拆票资格、共享箱守恒与授权锚点五轮复核修复
**Branch**: `main`

### Summary

完成任务 09-07-simplify-sea-order-house-bill-model 阶段 5/6 剩余工作并经五轮独立 Review 修复后归档。核心变更：拆票资格改为 HOUSE 订单唯一当前 HBL 即可拆票且历史 VOIDED 不阻断；Preview/Execute 增加逐结果逐货物共享箱交叉守恒与独占箱/共享箱版本强校验；三个 //go:build ignore 集成测试按新模型重写并在真实 PostgreSQL 全绿；拆票内嵌改配补外部确认契约（ExecuteSeaOrderSplit 新增 confirmation）；共享箱全部 RPC 增加 order_id 授权锚点并绑定锚点订单与运输执行/共享箱归属，权限收敛为 container.*；工作台完成异步上下文隔离（上下文键重挂载+请求序号防迟到覆盖）、服务端订单分页与货物行解耦、跨页草稿保全、确认单事务化并携带四实体乐观锁；统一 SharedContainer→Allocation 锁序并消除 Update 反向锁序与 TE 可变语义（不可变+锁内锚点校验+提交后私有 reload）；补真实 Authorization 中间件测试（含跨组织读写正向切换）。验证：真实 PostgreSQL 集成测试、go vet、govulncheck、tsc、biome、vitest 350 项、生产构建、生成幂等全部通过。教训：纯内存单测无法发现 NOT NULL 契约缺失与锁序死锁——集成测试必须接真实库；授权锚点与业务资源上下文必须双向绑定，仅中间件鉴权不够。

### Git Commits

| Hash | Message |
|------|---------|
| `ef9ae789` | fix: 修复拆票资格与共享箱守恒闭环并恢复集成测试 |
| `e273867f` | feat: 对齐共享箱细粒度权限并实现候选订单服务端搜索分页 |
| `f53b4237` | fix: 补齐共享箱授权锚点与工作台异步上下文隔离 |
| `0b876598` | fix: 补齐共享箱确认乐观锁与锚点业务上下文绑定 |
| `8b54f7e6` | fix: 共享箱运输执行不可变并修正提交后响应重读 |
| `2210d559` | fix: 消除共享箱 Update 反向锁序并收敛 Confirm 预读鉴权 |
| `3cc7fca7` | test: 收紧共享箱 Update 并发测试为恰好一个成功 |

### Status

[OK] **Completed**


## Session 36: 统一全站面包屑UI规范与组件
<!-- trellis-session: v=2 fp=df438608e4d45253 -->

**Date**: 2026-09-08
**Task**: 统一全站面包屑UI规范与组件
**Branch**: `main`

### Summary

排查并收口全站面包屑规范，移除订单模块中无实体的'订单管理'虚拟层级；废弃DocumentDetailLayout自研面包屑；统一页面跳转使用标准Link(href)；同步更新组件规范文档及单元测试。

### Git Commits

| Hash | Message |
|------|---------|
| `6e302495` | refactor(web): 统一全站面包屑规范并移除订单管理等虚拟层级 |

### Status

[OK] **Completed**


## Session 37: 美化页头面包屑为现代紧凑微交互流
<!-- trellis-session: v=2 fp=4308b24d129c1dc4 -->

**Date**: 2026-09-08
**Task**: 美化页头面包屑为现代紧凑微交互流
**Branch**: `main`

### Summary

精简返回按钮为带Tooltip的轻量圆形图标按钮，移除生硬垂直分割线；面包屑增加Hover高亮蓝与浅底微交互动效；升级斜杠分隔符为柔和样式；保持纯白高密度吸顶布局。

### Git Commits

| Hash | Message |
|------|---------|
| `37bc9414` | style(ui): 美化页头吸顶栏与面包屑交互样式 |

### Status

[OK] **Completed**
