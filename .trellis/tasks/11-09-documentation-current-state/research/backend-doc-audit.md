# 后端文档现状核查

- Query：核查 server 文档与后端规范中的过时描述，给出源码依据和修正建议；当前只做规划研究。
- Scope：internal，只写本任务 research；未改业务代码、规范、配置或数据库。
- Date：2026-11-09

## 已确认差异

| 文档位置 | 当前描述与问题 | 源码依据 | 建议修正 |
| --- | --- | --- | --- |
| `.trellis/spec/server/backend/finance-bill-currency.md:53`、`:58`、`:65`、`:94` | 要求先有效费用批准、无任何待审补录或更正再建账；还描述费用审批撤销 | `server/internal/data/finance_bill.go:355`–`:400` 的 `LoadBillableFeesScoped` 只对本批费用调用 `pendingCorrectionSourceFees`，无人工审核前置；`:396` 拦截待审源费；`:399` 计算是否已占用与需要记账。新契约 `order-fee-review.md:31`–`:35` 同样明确分供应商建账、无关费用不阻塞 | 写为当前有效、需建账、无有效账单占用且本条源费无待审更正；人工审核不是前置；订单锁内重验完整经济事实，不恢复独立审批 |
| `.trellis/spec/server/backend/finance-bill-currency.md:67`、`:88` | 声称 NETTING 尚未实现、应拒绝 | `server/internal/biz/finance_bill_batch.go:59` 接受 NORMAL 和 NETTING；`:133` 校验同结算单位及币种成对应收应付；`:266` 生成预览配对；`:617`–`:629` 原子生成双方原始账单和草稿对冲 | 仅未指定/未知模式拒绝；NETTING 同单位同币种必须双向配对，同事务生成原始账单和草稿对冲，账单确认后另行确认对冲 |
| `.trellis/spec/server/backend/finance-bill-currency.md:200`–`:205` | 删除示例还要求撤销独立费用批准、财务锁释放，并把更正后的替代费用全部永久保护 | `server/internal/data/order_fee.go:563`–`:598` 删除先锁订单并核对版本，仅 `superseded_by_request_id != nil` 或 DECREASE 历史行特殊保护；有效账单占用独立拒绝。当前未被替代 INCREASE 替代费无额外永久禁止；`:618` 事务内经济重算。`server/api/finance/v1/settlement.proto:256`、`:299` 已 reserved finance_locked | 删除示例改为先回退有效账单；人工锁定时按解锁或补审路径办理，未锁且 ACTIVE 的当前有效未建账费用可按权限删除；被替代原费与内部冲回历史行保护，不能连现行替代费一起永久冻结 |
| `.trellis/spec/server/backend/index.md:30` | 提成规范导航仍写“提成净额财务锁、净额归零释放” | 对应文档 `finance-commission-lock.md:5` 已明确财务锁退役，正文为经济重算与已发事实保护；`order_fee.go:618` 在普通删费事务内重算 | 导航标题/摘要改为“提成经济重算与已发事实保护”，保留已有文件路径，描述普通费用/建账/补审经济差额与实际支付事实 |
| `.trellis/spec/server/backend/order-lock-and-document-version.md:11` | “钉钉审批的外部派发、回调与本地生效另行实现”已落后实现 | `server/internal/biz/dingtalk_approval.go:188`–`:235` 已有后台派发与 Create；`:262` 起有 Inbox 消费；`server/internal/data/dingtalk_approval_inbox.go:208` 有 ApplyApproved，`:279` 清除人工锁并递增版本；`server/internal/server/dingtalk_approval_worker.go:41` 起运行消费者 | 描述现有派发、签名回调、权威查询与锁内本地生效链路，同时保留 PENDING_DISPATCH 不表示上游已发起、UNKNOWN 不自动重发的边界。不能因此宣称真实钉钉环境已完成验收 |
| `server/README.md:17` | 港口同步“需人工下载发布包”容易误导首次使用 | `server/cmd/sync-unlocode/main.go:127`–`:155` 默认查找归档的 seeds/loc251csv.zip，配套中文表自动定位；仓库已存在该发布包 | 改为默认读取仓库 seeds 归档包；更新版本才另行下载并指定 source、release、chinese-source |
| `server/README.md:68` | 把 self 范围写为仍保留配置并由运行时处理 | `server/migrations/20261016100000_retire_role_data_scope.sql:1`–`:4` 先停用 self 角色再删除整列；`server/internal/data/ent/schema/role.go:16`–`:22` 不含范围字段，`server/internal/data/bootstrap_administrator.go` 无 scope/self 分支 | 改为角色范围已退役，旧 self 角色由正式迁移停用；bootstrap 保留角色启停，不重新启用。不能声称仍保留该字段配置 |
| `server/seeds/README.md:70` 附近航司预览示例 | 输出样例仍含“组织：HQ”；同步为公共目录且当前输出无此字段；高级用法标题亦称可切换组织 | `server/cmd/sync-airlines/main.go:342` 只输出“版本：…，SHA-256：…”，parseOptions 仅 apply/source/release；无 organization 参数 | 移除过时组织输出及“指定组织/切换组织”措辞，说明公共主数据归属及实际 CLI 参数 |
| `.trellis/spec/domain/glossary.md:50` | 提成归属行将契约记为 `server/backend/operating-company-commission-attribution.md`，根目录无该路径 | 实际文件是 `.trellis/spec/server/backend/operating-company-commission-attribution.md` | 使用正确路径或相对 Markdown 链接 `../server/backend/operating-company-commission-attribution.md` |

## 服务端模板说明需要重写

`server/AGENTS.md` 仍以英文 Kratos 模板描述仓库，约 150 行，包含当前项目未普遍采用的 AIP ListOption/fieldmask 范式（约第 63–79 行）和 Todo 范例，与根 AGENTS 的中文要求及当前分页契约不一致。

可按根规范浓缩为中文服务端约束，并链接专项规范；不能把模板的新资源配方当作仓库当前全部接口的事实。示例实现入口：

- `server/internal/service/masterdata.go:124` 使用 `biz.MasterDataListOptions{Page, PageSize, Kind, Keyword}`；`:553`–`:565` 归一分页，不是强制 AIP filtering/fieldmask。
- `server/internal/data/ent/schema/role.go:11` 起明示工作台功能权限实体。
- `server/Makefile:15` 起提供 init/config/api/build/generate 等实际生成目标；`api` 执行权限规则生成，`all` 执行 api/config/generate。

`server/internal/{biz,data,service}/README.md` 当前分别只有 `# Biz`、`# Data`、`# Service` 一行，可补充最小中文职责与规范入口，避免新成员把模板空文档当作完整说明。此项是文档补全，不是已确认行为错误。

## 可保持的描述及避免误报

- `server/README.md:88` 的旧 Excelize 精确豁免版本仍与 `scripts/check-server-vuln.mjs:12`–`:19` 完全一致；不能直接把文档版本替换成 go.mod 当前依赖版本，否则会扩大豁免。当前 `server/go.mod:3` 为 Go 1.26.10、`:19` 为 Excelize `v2.11.1-0.20260930021559-01a9ff32fb3c`；该新版不会继承旧版精确豁免。可补充“豁免仅在报告 affected 且版本精确匹配时适用”，不能说版本变化本身必然失败，也不声称本次执行过漏洞扫描。
- 钉钉 `FEE_REVIEW_PENDING/RESULT` 模板并非纯历史：`server/internal/data/order_fee_correction.go:315`–`:317` 仍复用该模板处理费用更正待审提醒；`notification_test_usecase.go:74`–`:82` 保留八类测试目录。不能因独立整票审核退役删除这些通知能力；相关文档若调整应说明“当前更正复用提醒渠道”，而非把模板都判为下线。
- 退役规范中列出的 milestone 生成文件是明确的已删除清单，不属于错误路径；业务标签、企业微信退役说明亦不能机械修正为文件必须存在。
- `quality-guidelines.md:17` 的 web/types 在全局生成物禁令列表中出现，但当前目录不存在，不足以判定规则错误，应避免伪造该目录或为通过路径检查机械删规范。
- `server/migrations/README.md` 已区分开发校验和重录、生产严格校验，不应简单把“禁止修改历史迁移”当成所有环境绝对规则再扩大。

## 核查覆盖

已读或重点核对：

- `server/README.md`、`server/AGENTS.md`、三层 README、`server/migrations/README.md`、`server/seeds/README.md`、`server/seeds/ports.zh-CN.md`；其余 seed 说明通过库存/路径扫描核对。
- 后端专项详细阅读：finance-commission-lock、finance-bill-currency、order-fee-review、order-fee-correction、order-lock-and-document-version、retire-role-data-scope、auth-session-org-switch、database-guidelines、capability-navigation、index、dingtalk-notification-markdown、dingtalk-notification-test。
- 所有 `.trellis/spec/server/backend/*.md`（48 份）进行了关键词冲突和内联仓库路径/Go 文件名存在性扫描；对退役清单、范例路径、通配符作人工区分。其余文档尚未逐条契约深审，不能将本初步报告描述为所有规范的逐字段完整验证。
- `.trellis/spec/domain/glossary.md` 检查后端事实和路径，发现上列错误契约定位。

## Caveats / Not Found

- 当前仅规划研究，未运行数据库、同步 CLI 的 apply、迁移、集成测试、全量门禁或外部钉钉验证；没有可证明线上配置状态的证据。
- 用户要求全仓文档最新；上述结果是供规划收敛的初步证据，实施阶段仍需补齐其余规范的逐份语义核对及链接检查。
- 未修改任何现行文档，也未做任何 Git 写操作。源码行号均以此次工作树为准。
