# 修复与验证结果

日期：2026-11-10。用户已批准实施、独立 review 后合并 main 并 push；使用独立工作树与 `codex/constraint-export-audit` 分支隔离并行客商工作。

## 完成行为

- 命名约束通过 Ent 类型门禁、pgx SQLSTATE 和完整 `ConstraintName` 映射；未知错误原样返回。64 处调用配置、36 种约束名已对照正式迁移，修正账单幂等子串及两种超长名称。
- 提成导出在可靠身份和组织范围内，为参数、Count、超限、批读、取消及成功各记录一次审计；失败不返回部分数据，成功审计失败也不交付数据。
- 审计保留请求元数据，用独立 5 秒超时脱离请求取消；明确拒绝活动/关闭的共享事务上下文。关键字只记录存在性与字符数，失败不记录底层报文。
- 错误链保留业务与审计两侧原因，HTTP 语义和错误文本分别验证安全；身份、权限、公司及真实 ProtoJSON 枚举边界保持原契约。
- 同步错误/日志规范、核查总览和社区联系人适用说明；未修改 API、权限、Schema、依赖或生成物，历史核查台账保留原时点。

## 验证证据

| 验证 | 结果 |
| --- | --- |
| `go -C server test ./internal/biz ./internal/service -count=1` | 两个完整包通过 |
| data 导出审计、取消、元数据、关闭上下文定向测试 | 通过；使用 sqlmock，不声称真实数据库持久化验收 |
| 专用一次性 PostgreSQL 中 `go -C server test -p 32 -timeout 25m -count=1 ./internal/data -run '^TestMapEnt' -v` | 通过，2.340 秒，无 FAIL/SKIP；使用正式迁移及随机 Schema |
| 独立 review 相邻 data 回归 | 通过，0.050 秒；更新三处旧 pq 文案夹具为真实 pgx 结构，不添加兼容分支 |
| 最终 `pnpm run check:server` | 退出 0；重复检测、分层、Proto lint、全服务端默认测试、vet、可达漏洞扫描通过 |
| `git diff --check`、错误目录生成器 `--check` | 通过，403 个错误码一致 |
| 本轮十一份变更/新增 Markdown 的本地链接 | 9 个目标均存在；归档后再核对任务引用 |

真实数据库分别触发账单幂等、月度提成申请和主单身份的唯一冲突，核对驱动错误名与 catalog：

- `financebill_organization_id_idempotency_key`
- `financecommissionapplication_organization_id_employee_id_applic`
- `seamasterbill_organization_id_shipping_line_id_normalized_maste`

独立复核由未参与主体实现的 `trellis-check` 完成，发现并修复三个相邻测试夹具问题，最终无遗留阻塞；详见[独立复核报告](./research/independent-review.md)。首轮失败与最终门禁日志分别位于本机 `/tmp/roncin-constraint-first-server.log`、`/tmp/roncin-constraint-final-server.log`，不提交临时日志。

## 验证限制

- 默认服务端门禁未注入专用数据库环境，历史数据库/实时验收用例按原配置跳过；上述专项 PostgreSQL 通过不代表完整真实库验收。
- 审计取消、事务拒绝与元数据以 sqlmock、领域和传输回归验证，未额外执行审计真实数据库持久化；共享事务内独立保存不在支持范围。
- 未执行外部宿主、设备、第三方服务、业务端到端或正式部署验收；没有前端业务代码变更，未重复前端门禁和生产构建。
- `settlement_commission.go` 一处原有未改字段行的 gofmt 对齐差异保留，未做无关格式化。

一次性测试库由本任务单独创建，收尾时清理；日常开发数据库与原未提交任务不删除、不重置。
