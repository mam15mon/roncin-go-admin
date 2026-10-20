# 验收与审查记录

## 已交付行为

- 公共计费单位完整目录 38 项，正式迁移与开发 seed 保持一致；同码已有设置及费用引用保留。
- 数量预览接口显式区分计划/实际箱量，支持本订单 CBM、毛重 KG、吨、RT，展示原始数据和计算依据。
- 表格、普通弹窗、复制、补录/更正共用确认交互；确认后更新数量和金额，保留手工汇率。
- 实际箱量遇共享箱、箱型无精确匹配、缺数据、超精度或整数规则不符时解释原因，保留人工填写。

## 独立 review

后端审查确认组织谓词、numeric 精度、共享箱口径、迁移与 seed 一致；新增真实 Authorization 中间件测试，覆盖正确业务权限、无费用权限、其他业务线权限、跨公司与未登录。

前端审查修复并验证：

1. 成功预览后重新查询失败，React Query 仍保留旧 data：增加 isError 门禁，失败后不能确认旧数量；新增 invalidate→失败→重试回归。
2. 表格 rowKey 可能是数组：新增的已建账门禁与历史基线比较统一 String(rowKey)，补真实表格冻结用例。

无未修复的审查问题。接入订单费用抽屉时发现原金额变化回调为空，为本期数量带入即时金额要求传入真实 formRef，并复用现有精确金额算法补齐预览。

## 定向验证

- 单位目录真实 PostgreSQL：TestBillingUnitCatalogColdStartPostgres、TestBillingUnitCatalogUpgradeAndSeedPostgres、TestDevelopmentSeedPostgres 全部 PASS，无 SKIP。
- 后端预览：20 个领域规则场景、service 边界与 DTO、TestFeeQuantityPreviewPostgres、TestAuthorizationFeeQuantityPreview 全部 PASS。
- 专用 PostgreSQL 验证计划 3/实际 2，8 CBM/2500 KG/2.5 吨/8 RT，numeric 大数精确聚合、空聚合、精度拒绝、组织隔离及共享箱。
- 前端实际表单与表格验证确认、取消、手改、来源/订单/单位切换、迟到响应、冻结、重查失败、金额及手工汇率；各受影响测试文件定向通过。
- Go vet、Proto lint、修改文件 Biome、前端 tsc 与 git diff --check 通过。

## 开发环境

- 迁移前只读检查：7 项单位，已有迁移校验和无差异，本次迁移为唯一待应用文件。
- pnpm run migrate:dev 成功：单位 38/38 启用，本次 schema_migrations 记录 1 次；权限和订单主数据同步变更均为 0。
- 开发 seed 代码已更新；完整 seed 两次执行在专用库验证，未在日常开发库额外注入整套业务测试数据。
- 所有集成验证使用独立一次性数据库与随机 Schema，不重置开发数据。

## 门禁补充修复

- 首轮完整 check:fast 已启用所有 PostgreSQL 开关；前端通过，后端除 data 外的包通过，包括 migration（124.074s）和 server（71.010s）。data 在密码重置锁等待用例执行期间触发 Go 每包默认 10 分钟总时限，日志没有断言失败；不能将此轮计为完整通过。
- 按 go test -list 得到 data 全部 376 个顶层测试，均匀分成四组，分别使用独立一次性数据库与正式迁移。每组 94/94 PASS，总计 376/376，0 FAIL、0 SKIP、0 遗漏；各组耗时 171.465、182.450、206.628、217.748 秒。分组结束删除四个测试数据库。
- package.json 的 check:server 添加 -timeout 25m，与 CI 原有设置对齐；不改变用例、断言、并发或跳过策略。
- 安全扫描发现原 Excelize 依赖新增可达漏洞 GO-2026-6684、6685、6688、6689。固定升级为 v2.11.1-0.20260930021559-01a9ff32fb3c，及其上游要求的 mscfb 1.0.8、efp 0.0.2，无其他模块升级。
- 官方修复依据：https://github.com/qax-os/excelize/commit/01a9ff32fb3c1f873cf01205e1b8a3285b0e1d23 。该提交包含四项修复；正常 Excel 导入、公式拒绝、日期与负共享字符串索引内存/临时文件回归、船司解析、go mod verify 及相关 vet 均通过。
- check:server:vuln 升级后 PASS：无未处理可达漏洞，本次未使用任何审计豁免；漏洞脚本与豁免列表没有改动。独立 review 确认仅必需依赖与包总时限调整。

## 最终门禁结果

升级后的普通 pnpm run check:fast PASS，117.2s；前端 272 文件、1541 用例通过，原有 1 个海运模板文件、12 个用例保留既有 Skip。普通模式的数据库 Skip 不计为真实数据库通过，真实库证据以首轮其余包与上述 data 完整分组结果为准。

## 合并集成

合并前本地 main 新增已完成的组织切换保留页面及未保存输入保护。已同步到本次分支，保留 GuardedModalForm 与表格保护；数量程序赋值显式登记用户修改，取消预览不置脏。普通/复制、补录/更正、应收/应付表格共三文件九例集成回归通过，独立 review 无残留问题。

合并后 pnpm run check:web PASS：288 个文件、1606 个用例通过，原有海运模板 1 文件、12 用例仍跳过；类型、架构与 Biome 门禁通过。go build ./cmd/server 通过（产物位于仓库外）。

所有五个一次性验收数据库已删除，查询确认无本任务测试库残留；开发库只执行正常增量迁移，没有数据重置。

## 交付状态

任务已归档，会话已记录。完成主分支集成后 fast-forward 合并 main，并成功推送 origin/main；本次 codex/billing-unit-quantity-assist 工作分支已删除。其他既有分支与工作区保留。
