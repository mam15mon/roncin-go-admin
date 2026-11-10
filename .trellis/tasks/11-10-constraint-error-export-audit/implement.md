# 执行与验证计划

## 规划

- [x] 用户批准创建任务；已检查main状态并保留原未跟踪任务。
- [x] 检查Ent Unwrap、pgx驱动及现有审计链路，明确最小边界。
- [x] 汇总两份研究报告，收敛PRD、设计及验收；保留数据库实测边界。
- [x] 已提交最终方案；用户批准实施，并要求独立review后合并main并push。

## 实施（批准后）

1. trellis-implement修data/ent_error.go及必要调用方约束名，先完成实现再更新针对性测试，不手改生成物。
2. 另一implement负责biz提成导出审计及data/finance_commission_ledger.go的共享ctx拒绝，补biz/data/service/transport相关最小验证；共享测试文件责任由主会话协调。
3. 主会话同步error-handling/logging规范、docs核查总览与社区入口说明，保留历史台账；关键字仅记录存在性和字符数，错误Error()/HTTP转换/审计内容均验证安全。
4. 独立trellis-check按设计复核错误链、精确名称、权限/公司、敏感输入、取消/审计超时及失败不交付数据。
5. 最小验证：go -C server test -p 32 ./internal/data -run 'TestMapEnt|TestCommissionRepoSaveExportAudit|约束相关回归'；go -C server test -p 32 ./internal/biz -run 'TestCommission.*Export'；service/server相关导出权限及安全错误用例；git diff --check。
6. 任务验收运行pnpm run check:server（公共data错误映射影响多个仓储、审计跨biz/data，需服务端门禁）；不因无前端代码变更跑完整前端或构建。真实集成测试SKIP单列。
7. 规范/生成错误目录一致性及文档链接检查，分组中文Conventional Commit，保留用户原四文件；归档与会话记录；独立review通过后合并main并push（用户本轮已授权）。

## 预期文件

- server/internal/data/ent_error.go、ent_error_test.go、必要命名约束调用方及关联测试。
- server/internal/biz/finance_commission.go、finance_commission_test.go或专用导出测试。
- server/internal/data/finance_commission_ledger.go及审计测试；必要service/server回归。
- .trellis/spec/server/backend/{error-handling,logging-guidelines}.md、docs/markdown-audit.md、web文档的社区原文说明。

如果发现需更改事务封装/Schema/权限/API，返回规划说明，不自行扩大。审计不使用新事务/锁/版本字段，SaveExportAudit只接受普通上下文，5秒独立取消上下文可测试。

## 实施与复核完成

- [x] 结构化映射、全部必要运行时约束名及相关测试已同步。
- [x] 导出成功/失败/取消审计、共享上下文拒绝及安全错误链已实现。
- [x] 规范、总览及社区公约适用说明已同步，历史研究时点明确保留。
- [x] 定向与专项真实库测试通过；独立review修复三处相邻测试夹具后，全服务端门禁通过。
- [x] 错误目录及差异检查通过；验证结果与未执行边界见validation.md。
- [ ] 分组提交、合并main、归档记录及用户已授权的push。
