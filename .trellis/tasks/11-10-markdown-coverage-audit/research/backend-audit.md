# Research: 后端 Markdown 逐份覆盖补查

- Query: 核查48份后端规范与server下10份Markdown，逐份登记状态和真实证据，避免把旧报告或链接检查当作语义通过。
- Scope: internal；仅写当前任务research，不改现行文档/代码，不做git操作。
- Date: 2026-11-10

## Findings

当前逐份台账见 [backend-ledger.jsonl](./backend-ledger.jsonl)，完整登记58条。此为可持续补充的研究快照，未完成独立源码核查者明确待确认；前轮报告只作定位线索，不自动通过。

### 已发现的规范与实现差异

1. `.trellis/spec/server/backend/error-handling.md:26` 禁止字符串匹配驱动细节，而 `server/internal/data/ent_error.go:29` 的集中命名约束helper先 `ent.IsConstraintError` 后 `strings.Contains(err.Error(), constraintName)`。需确认集中命名匹配是否为有意例外，或属于实现欠账；本轮不擅自削弱规范。
2. `.trellis/spec/server/backend/logging-guidelines.md:12` 要求提成导出成功/失败审计，而 `server/internal/biz/finance_commission.go:142` 的参数、计数、超限、批读失败均在 `:165` 的保存审计前返回；`:196` 构造结果固定 `success`。疑似实现缺口，保持现行规范，列待确认。

### 已读取的入口与核对方式

- `.trellis/workflow.md`：研究只持久化research，现处规划期。
- 当前任务prd/design：本轮与前轮区分；每路径保留基线SHA；不凭文档类别批量通过。
- `.trellis/spec/domain/glossary.md`、`architecture-map.md`：领域链路与组织/费用/提成当前口径。
- `.trellis/spec/server/backend/index.md`：48规范导航与生成、分页、工作台、锁等检查要求。
- `server/AGENTS.md`、`server/README.md`、三层README、`server/migrations/README.md`：人工全文阅读；对 `server/Makefile`、根 `package.json` 的命令已读取，更多业务事实仍待逐项证据化。

## Related specs

根AGENTS、server/AGENTS、后端index、error-handling、logging-guidelines、domain glossary/architecture-map。

## External references

无。此次只核对仓库现状，不对外部平台版本或设备环境作新结论。

## Caveats / Not Found

- 台账状态如实区分已核实和待确认，不声称全部58份已完成本轮逐字段源码深审。
- 前轮48份全文阅读与关键入口核对报告位于 `.trellis/tasks/archive/2026-11/11-09-documentation-current-state/research/backend-update-result.md`；其证据可定位后续核查，但不能自动作为本轮全量通过证明。
- 未运行Go测试、生成、迁移、种子apply、写库、云端/钉钉/设备验收。历史环境通过记录不当作本轮新结果。

## 补查批次一（5份）

- migrations README：对照runner的advisory锁、校验和修复、非线性拒绝与逐文件事务，无新增文案错误。
- 三份种子编制说明：全文读取；CSV17524/360及类别统计、港口232国、船代SHA均直接计算一致，CLI参数与幂等/覆盖规则有源码依据；历史真实库结论不当成本轮执行。
- directory-structure：生成目标与buf配置、pnpm脚本一致，分层内容是开发约束，不声称已运行扫描。

## 补查批次二（前5份规范）

空运出口、登录组织切换、后台通知、整数计费单位与业务标签退役均全文读取并补直接代码/Proto/迁移证据，关键签名、当前状态及算法未发现新的文案差异；测试章节为应满足要求，不表示本轮运行。

## 本轮停止点与后续首批

完整台账58条，当前状态：{'已核实无需修改': 13, '待确认': 45}。规划收敛时停止研究，不把尚未证据化项赶成通过。后24规范由另一研究代理独立核查，主会话合并时以extra台账覆盖。

下一批优先：

1. capability-navigation、commission-scheme-assignment、company-admin-provisioning、company-english-name、database-guidelines：全文与关键调用/锁序补证。
2. dingtalk-embedded-login、dingtalk-notification-markdown、dingtalk-notification-test、dingtalk-qr-protocol、dingtalk-registration-approval：以本地协议/模板/消费者实现核查，外部历史验收不重标本轮通过。
3. exchange-rate-single-rate、finance-bill-currency、finance-commission-lock、mobile-login-password-reset、operating-company-commission-attribution：关键范围、金额来源、失败与并发入口补证。
4. server/README、server/AGENTS、server/seeds/README：已全文阅读并有部分直接依据，剩余跨域事实补完再定最终状态；两条规范实现差异仍待主会话决策，禁止直接降级审计或错误处理要求。

本轮只写backend-audit.md与backend-ledger.jsonl；未修改任何现行文档、代码、Git、数据库或运行配置。
