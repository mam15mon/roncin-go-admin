# 空运出口闭环验收记录

日期：2026-10-21（Asia/Shanghai）。用户授权实现、review 修复、合并 main 并推送。
起始状态：`main@5c43db86`，工作区干净，比 origin/main 领先 12 个已有提交；
本任务创建 `codex/air-export-business-loop` 实施，保留全部已有提交。

## 隔离环境

- 本任务新建专用 PostgreSQL 验收库和角色，正式 SQL 迁移初始化；集成测试继续在库内随机
  Schema 验证，未使用 Ent Schema.Create，未清空、重置或迁移既有开发库。
- 连接与临时验收账号凭据只通过仓库外私有文件及环境变量注入，不写入本记录、代码或命令输出。
- 浏览器：后端 127.0.0.1:8010、Vite 127.0.0.1:8111，钉钉关闭；不发送真实通知。
- sync-dev 在隔离库实际运行，新增同 MAWB 的两票独立 HAWB 和货物样例；重复运行无新增关系或历史。

## 已执行验证

| 验证 | 结果及断言 |
| --- | --- |
| `task.py validate` | implement/check 上下文各 11 条有效记录通过 |
| 后端 AE 核心真实 PostgreSQL 定向 | 双票共享/完整汇总/唯一性/必填/无半票/终止恢复/版本/流程/超时/锁快照/审计回滚通过，无 SKIP |
| 重量与费用来源三层定向 | 6000/5000、1/3 分泡、HALF_UP 0.001、主单400/分票500、PLANNED/ACTUAL隔离、130直接/135调整、过期来源/跨票拒绝、并发与审计回滚通过 |
| 相邻后端回归 | SE 创建/幂等/生命周期、费用数量规则/停用单位原引用/批量费用维护通过 |
| 工作台及通知定向 | AE协作订单进入近期列表；无对应读权限/跨公司/非协作人不返回；费用审核准确UUID的SE/AE受控路径通过 |
| `TestDevelopmentSeedPostgres` | 正式迁移冷启动、重复种子、两票真实主分单关系及400kg/1.8m³通过 |
| 前端定向 | 10文件51用例通过：AE适配、重量确认/旧值、费用来源调整、厘米尺寸填入、注册表、工作台、SE数量回归 |
| 类型/架构/分层 | 前端 tsc、产品架构扫描、Go分层扫描通过；后端三层 vet 通过 |

## 独立 review

由独立 `trellis-check` 对完整计划与全部变更审查，发现问题先核对事实并修复：

1. 恢复已终止票的关联会恢复生命周期：补 AE.transition 权限，除 AE.update 外重新校验，
   并补服务层反例及生命周期事件。
2. HAWB 中合法连字符不能按 MAWB 规范化删除：分开处理搜索口径，补关系搜索回归。
3. Proto 更新/关联响应需符合仓库 RPC 命名和唯一响应规范：修改契约源并统一生成。
4. 费用来源改为手工的未保存标记、已建账来源编辑禁用，以及主单/重量确认后相关缓存刷新。
5. 真实浏览器发现创建人被误当业务分工提交及 AE 费用头部海运字段/机场 ID 展示，按公共契约修复。

预计起降时间的必填沿用起始基线 `normalizeOrder` 已有契约；验收样例已补齐合法时间，
不将缺时间样例的拒绝归为产品缺陷。

## 真实浏览器验收

全部使用 Playwright、真实页面与隔离服务，不 mock 后端接口。

| 验收 | 结果及边界 |
| --- | --- |
| AE 专属页面闭环 | 最新后端最终验收通过，1 例 46.4 秒；UI 新建 A/B 两票、显式选主单、私有资料保存刷新、从主单继续建票、共享航班影响预览和执行，完整成员汇总为 400 KG |
| AE 重量 UI | 委托货量参考 200 KG、人工确认 130 KG；实际货物 120×100×100 cm×1 明确填入 1.2 CBM，实际参考 200 KG；确认后刷新重读；体积合计 1.8 无浮点尾数 |
| AE 流程 UI | 实际点击草稿→已订舱→已配舱，填写流转原因并逐步校验状态响应和重读；完整六节点矩阵由真实 PostgreSQL 验证，不宣称六节点均由浏览器点击 |
| AE 冲突与锁 UI | 模拟另一操作员修改，旧共享预览提交返回 409，弹窗保留；锁定另一成员后，共享预览显示明确阻断；测试无 pageerror |
| AE 财务 | 1 例约 50.6 秒；页面完成应收 125.00 的录费/审核/建账/收款核销与提成 12.50；同例通过真实 API 完成发票登记签发，以及另一 AE 票的应付 25.00 审核、建账、付款与核销并重读 |
| SE 财务相邻回归 | 同一财务页面用例切回 sea-export，通过，约 1.4 分钟；保留海运应收录费、审核、账单、收款核销与提成原有断言 |

用例：`web/tests/e2e/air-export.e2e.ts`、`web/tests/e2e/finance-bill-batch.e2e.ts`。
后者通过 `RONCIN_ACCEPTANCE_ORDER_KIND=air-export/sea-export` 明确选择类型，默认仍为海运。
截图与 trace 保存在仓库外验收目录，不提交可能含会话信息的浏览器原始 trace。

## 最终质量门禁与修复

- 独立 review 覆盖 `main@5c43db86` 至本任务全部已提交和未提交实现，无剩余已知阻断项。
- 补齐四个分页入口的 28 个边界子例：页码、容量非法值及 1/200 边界。
- 首轮全量检查暴露的旧测试已修复：提供真实 QueryClient 上下文、更新实际路由数量及
  AE 组织切换权限断言、等待 Form 异步渲染清空结果。8 文件 64 例定向回归通过，
  保留原有业务断言，不通过改产品绕过测试。
- 第二轮前端完整门禁通过：296 个测试文件通过、1 个跳过；1624 例通过、12 例跳过。
  后端暴露旧船期 SQL mock 与新增 AE 权威主单分支不一致，已修复并定向通过；
  保留对 UTC 完整时间下界/上界和 SE 字段分支的验证。
- 截图复核发现成员体积合计的浮点尾数显示，摘要及差值按原货量表单三位精度规范化后
  去尾零；原用例复用 1.2+0.6→1.8、申报 1.8 差值 0 的断言通过。没有改变 API 或重量计算。

常规全量测试的既有跳过项不代替数据库验收；本任务关键 PostgreSQL 定向已注入
专用隔离连接并实际通过，无 SKIP。未执行全库所有历史模块的 PostgreSQL 集成验收。

最终 `pnpm run check:fast` **退出码 0，全部通过**：WEB 75.8 秒、SERVER 136.4 秒，
并行总耗时 136.5 秒。包含权限/Proto 常量、前端架构、重复代码、前端 lint/tsc/全量单测、
Go 分层、Proto lint、Go 全量测试和 vet、可达漏洞扫描；漏洞扫描无未处理项。
新增 AC4c 精确算例在最终门禁的 Go 阶段结束后补入，单独执行 `TestFeeQuantityPreviewRules`
21 个子例全部通过。最终 `git diff --check` 与任务上下文校验通过。

没有构建配置、依赖或生产打包入口变更，本次未额外运行 build；开发代理由真实浏览器验证，
生产同域静态资源/路径处理由全量 server/webassets 测试覆盖。

## 验收标准与证据映射

| 验收标准 | 实现及验证证据 |
| --- | --- |
| AC1、AC1a | AE 适配器/注册表测试；真实 UI 双票创建保存；`TestAirExportRequiredNumbersUniquenessAndRollbackPostgres`、创建幂等与引用校验相邻测试 |
| AC2、AC2a、AC8 | AE 独立动作矩阵与锁内复验；`TestAirExportFlowVersionsAndSnapshotsPostgres`、`TestAirExportAuditRollbackAndOverduePostgres`；页面订舱/配舱真实推进；原公共异常/结案与本地时间显示回归 |
| AC3、AC3a、AC3b、AC3e | `TestAirExportSharedWaybillPostgres`、`TestAirExportTerminationRestoreAndConcurrentIdentityPostgres`、`TestAirExportMemberChangeAndLockRacePostgres`；真实 UI 共同航班变化、私有保存、409 及另一成员锁定阻断 |
| AC3c、AC3d / R10 | 主单工作台、候选选择和从主单继续建票真实 UI；共享集成测试完整活动集合汇总与组织/权限隔离；主单申报量与成员量分别呈现 |
| AC4、AC4a、AC4b、AC4d、AC4e | `TestAirWeightReferenceCalculation`、`TestAirWeightInvalidAndStale`、`TestAirWeightConfirmationAndFeeSourcePostgres`；重量面板及厘米尺寸辅助组件交互；130 确认与 200 参考真实 UI |
| AC4c | RT 现有表驱动新增并实际通过 8000 KG/25 CBM→25 RT 精确算例；空运界面明确只输入 KG，TON 录入为条件项，本期未开放；不增加整箱 RT 派生写入 |
| AC4f | 重量与来源 PostgreSQL 集成：130 DIRECT、135 MANUAL_ADJUSTED、应付独立值、过期/跨票拒绝及后续不改旧费用；费用表单未保存和冻结门禁测试 |
| AC5 | AE 应收页面闭环、应付/发票真实 API 闭环；公共审核冻结、补录/更正、账单/核销/提成测试；未建立第二套财务规则 |
| AC6 | AE 路由/权限与组织切换测试、工作台 PostgreSQL 及前端测试、受控通知路径单测、真实 UI 详情/主单/费用入口 |
| AC7 | PostgreSQL 跨公司/权限、版本、成员变化并发、审计失败回滚及无半票；恢复双权限服务/前端交互测试；真实 UI 409/锁阻断 |
| AC9 | SE 创建/幂等/生命周期/单证及数量回归、SE 财务真实 UI；Proto/权限/架构/类型与最终全栈门禁 |


## 完成判定

AC1—AC9、AC1a、AC2a、AC3a—AC3e、AC4a—AC4f 及 R10 已按批准范围完成；
独立 review 无剩余已知阻断。实现、生成物、回归测试与规范按组提交，
本任务归档并记录开发日志后合并 main、推送；实际提交关系以 Git 历史为准。
