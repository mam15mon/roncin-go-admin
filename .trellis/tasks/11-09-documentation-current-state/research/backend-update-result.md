# 后端文档更新与核查结果

- 日期：2026-11-09。
- 写入责任范围：server 下 Markdown、`.trellis/spec/server/backend/`；未修改代码、Proto、生成物、配置、数据库或用户任务。
- 本轮在初步研究基础上逐份读取 48 份后端规范，核对关键身份/权限/状态/生成/迁移入口；不声称逐字段或真实环境运行验收。

## 更新摘要

- 24 份现行文档修正当前规则或补足导航，未改动所有文件的日期以制造“已更新”。
- server/AGENTS 用中文项目化约束替代通用 Kratos/AIP 模板；biz/data/service README 补职责和根规范导航。
- 初始化、全局主数据、工作台范围、费用建账/更正、经济重算、海运两态与模式切换采用实际实现。
- 历史 Switch 的表/RPC/枚举已退役，但当前 Manifest 仍有 switch 权限名称：文档明确区分，不扩本任务去改代码。
- 漏洞精确豁免版本和历史真实验证记录保留，没有以当前依赖版本扩大豁免，也未补造本轮通过记录。

## 关键新增证据

| 修正点 | 源码依据 |
| --- | --- |
| 未确定模式已退役、创建显式两态 | `server/api/order/v1/sea_document.proto:117`、`server/internal/biz/sea_document.go:260`、`server/internal/biz/order_usecase.go:489` |
| Switch 与 REPLACED/SWITCH 退役 | `server/migrations/20260907140000_simplify_sea_export_house_bill_model.sql:99`、`server/api/order/v1/sea_document.proto:140`、`:167`；当前 service 仅改单/作废/模式切换 |
| 公司方案配置不向系统工作台开放 | `server/internal/access/manifest.go:459`、`server/internal/biz/auth.go:852` |
| 正式客户账期带入已实现 | `server/internal/biz/finance_bill_batch.go:308`、`:324` |
| 船代引用公共目录而非 supplier | `server/internal/data/order_sync.go:85`；`:165` 的 Partner 引用仅客户/订舱代理/国外代理 |
| 共享船公司无组织列 | `server/internal/data/ent/schema/shipping_line.go:13`、`server/internal/data/sea_order_change_shared.go:169` |
| 退关恢复新 Link | `server/internal/data/sea_order_restore.go:128` |
| 空运通知合法费用动作 | `server/internal/biz/notification.go:255`、`server/internal/data/dingtalk.go:320` |

## 后端规范逐份覆盖

| 文档 | 已核对的实现入口 | 结论 |
| --- | --- | --- |
| `air-export-contract.md` | `server/internal/biz/air_export.go` | 已读，复核共享主分单、重量与公共财务入口，本轮未确认需改动。 |
| `auth-session-org-switch.md` | `server/internal/service/auth.go` | 已读，复核 400 天 Cookie、30 天滑动会话与工作区，未确认需改动。 |
| `background-task-notify.md` | `server/internal/data/background_task_scheduler.go` | 已读，四类消费者、NextRun 与固定通知通道存在，保留当前规则。 |
| `billing-unit-quantity-rule.md` | `server/internal/data/ent/schema/billing_unit.go` | 已读，数量整数字段与当前订单费用规则一致，保留。 |
| `business-tag-retirement.md` | `server/migrations/20261109100000_retire_business_tags.sql` | 已读，现行退役迁移与 Proto reserved 一致，保留。 |
| `capability-navigation.md` | `server/internal/biz/finance_commission_application.go` | 修正 Switch 入口与 Clawback 锁摘要，采用模式切换和现行提成经济保护。 |
| `commission-scheme-assignment.md` | `server/internal/access/manifest.go` | 公司财务配置权限不能在系统工作台取得，修正总部配置旧口径；方案区间与分配规则保留。 |
| `company-admin-provisioning.md` | `server/internal/data/bootstrap_administrator.go` | 已读，唯一初始管理员补齐真实公司成员及停用保护为现行权威，保留。 |
| `company-english-name.md` | `server/internal/data/ent/schema/organization.go` | 已读，公司英文名及真实 HBL 签发投影与当前模型一致，保留。 |
| `database-guidelines.md` | `server/internal/data/transaction.go` | 已读，事务/快照/锁/迁移规则保留，仅将已退役总部术语改系统工作台。 |
| `dingtalk-embedded-login.md` | `server/internal/data/dingtalk.go` | 已读，独立临时码身份交换、Cookie 与业务身份准入保持；未做真实钉钉验收。 |
| `dingtalk-notification-markdown.md` | `server/internal/biz/notification.go` | 动作白名单补当前已支持的空运费用入口；8 类消息模板不因独立费用预审退役删除。 |
| `dingtalk-notification-test.md` | `server/internal/biz/notification_test_usecase.go` | 已读，系统工作台有效管理员仅本人显式发送与示例动作保持。 |
| `dingtalk-qr-protocol.md` | `server/internal/data/dingtalk_qr.go` | 已读，本站二维码、独立尝试和长轮询入口存在；历史真机事实保留。 |
| `dingtalk-registration-approval.md` | `server/internal/data/dingtalk_directory.go` | 已读，当前为通讯录同步及逐人开通；旧注册/邀请只保留历史语义。 |
| `directory-structure.md` | `server/buf.gen.yaml` | 修正 Go/API/OpenAPI 与前端生成命令的职责顺序。 |
| `error-handling.md` | `server/internal/data/ent_error.go` | 已读，领域错误与驱动映射规则保留；映射入口按现有代码继续核对。 |
| `exchange-rate-single-rate.md` | `server/internal/service/order_fee.go` | 明确费用独立覆盖权限与账单/流水维护 update 权限的分界，避免矩阵覆盖专项例外。 |
| `finance-bill-currency.md` | `server/internal/data/finance_bill.go` | 修正人工预审前置、源费待审粒度、NETTING 已支持、删费与替代费保护口径。 |
| `finance-commission-lock.md` | `server/internal/data/fee_correction_commission.go` | 已读，经济重算与已发事实保护为当前权威，未恢复财务锁。 |
| `index.md` | `server/internal/access/manifest.go` | 修正财务锁/审核/Switch 摘要与检查清单，链接路径保留。 |
| `logging-guidelines.md` | `server/internal/biz/password_reset.go` | 已读，日志与审计分离及验证码敏感信息规则保留。 |
| `mobile-login-password-reset.md` | `server/internal/data/password_reset.go` | 已读，PENDING/READY、一次消费与失败计数门禁存在，保留。 |
| `operating-company-commission-attribution.md` | `server/internal/biz/auth.go` | 移除已删 AssignPersonnel 写消息指引、系统跨公司读旁路与角色范围措辞。 |
| `order-attachment-oss.md` | `server/internal/biz/order_attachment.go` | 已读，上传授权、最后引用删除任务及存储边界保持，未声称真实云端验证。 |
| `order-fee-correction.md` | `server/internal/data/order_fee_correction.go` | 已读，锁后统一 ADD/REPLACE/VOID、先回退源账务及当前有效替代费门禁与现行代码一致，保留。 |
| `order-fee-quantity-preview.md` | `server/internal/biz/order_fee_quantity_preview.go` | 已读，显式来源、不可用不回退及费用保存最终校验保持。 |
| `order-fee-review.md` | `server/internal/data/order_fee_review.go` | 已读，人工审核锁和费用摘要/逐费能力为当前权威，保留。 |
| `order-lock-and-document-version.md` | `server/internal/data/order_lock.go` | 修正钉钉派发已实现、退关恢复新 Link 与工作台推导锁资格矩阵。 |
| `order-milestone-retirement.md` | `server/migrations/20261030140000_remove_order_milestones.sql` | 已读，独立里程碑退役与操作进度保留，历史删文件清单不按失效路径误修。 |
| `order-personnel-editor-retirement.md` | `server/api/order/v1/order_personnel.proto` | 已读，独立 Assign/Remove 退役，List 和核心人员分工保留。 |
| `order-shared-enum-field.md` | `server/internal/service/order_enum.go` | 已读，可空枚举省略保留/显式零清空规则与现有转换入口一致，保留。 |
| `organization-shared-masterdata.md` | `server/internal/data/bootstrap_administrator.go` | 修正仅唯一初始管理员补真实公司资格，与普通系统管理员无自动准入的区分。 |
| `partner-casual-contract.md` | `server/internal/biz/finance_bill_batch.go` | 补正式客户账期带入已实现，散客零天默认保持独立，不再以旧任务未交付作当前前提。 |
| `partner-role-blacklist.md` | `server/internal/data/order_sync.go` | 黑名单仅三类 Partner 引用；船代已为公共目录，删除旧 supplier 映射。 |
| `passkey-authentication.md` | `server/internal/data/passkey.go` | 已读，WebAuthn 独立凭证/仪式与身份准入入口存在；历史验收保留，未做设备或线上验收。 |
| `quality-guidelines.md` | `web/scripts/generate-services.mjs` | 生成物清单改当前 JSON 输入/roncin 客户端/枚举，Go 测试示例采用根规范 -p 32；Go generate 补 ./...，避免在无 Go 文件的 server 根包执行失败。 |
| `retire-role-data-scope.md` | `server/internal/data/order_lock.go` | 将已退役独立费用审核示例改为目标品类真实 lock grant 及必要读写权限。 |
| `role-workspace-ownership.md` | `server/internal/data/admin_role.go` | 已读，角色库读写锚点与正式归一入口保持，未引入角色范围配置。 |
| `sea-document-change-history.md` | `server/api/order/v1/sea_document.proto` | 移除仍当现行能力写出的 Switch/RPC/REPLACED，补模式切换与外部确认；明确已退役历史和仍存在的权限名称不能当可执行能力。 |
| `sea-export-document-contract.md` | `server/internal/biz/sea_document.go` | 修正显式 DIRECT/HOUSE 两态、UNDETERMINED 已退役、全局船公司及公司签发归属。 |
| `sea-hbl-production.md` | `server/internal/data/ent/schema/sea_house_bill_print.go` | 已读，同一 HBL 印字、公司抬头及无审核输出前置保持。 |
| `sea-order-restore.md` | `server/internal/data/sea_order_restore.go` | 已读，原主单/航程新 Link 原子恢复为当前权威，修正相邻旧规范。 |
| `sea-shared-write-consistency.md` | `server/internal/data/sea_order_change_reassignment.go` | 已读，候选加票正文隔离、共享箱成员锁及整票改配占用规则保持。 |
| `sea-terminated-order-read.md` | `server/internal/data/sea_document.go` | 已读，精确结束 Link 读取与终态只读保持；当前输入基线不误当持久化恢复。 |
| `user-membership-batch.md` | `server/internal/data/admin_user_membership.go` | 已读，批量加入与明确 membership 编辑、事务/角色隔离保持。 |
| `wecom-retirement.md` | `server/migrations/20261009130000_retire_wecom.sql` | 已读，企业微信纵向退役与历史字段时点区分保持。 |
| `workbench-todos.md` | `server/internal/data/workbench_todo.go` | 已读，本人协作/公司/具体品类权限交集与记账事实投影保持。 |

## server 目录说明覆盖

- `server/AGENTS.md`、`server/README.md`、`server/internal/{biz,data,service}/README.md`：更新，命令与 Makefile、根 package.json 核对。
- `server/migrations/README.md`：区分生产不可改历史、开发校验和迭代；重建库明确另需授权。
- `server/seeds/README.md`：公共目录无 org-code，更新预览输出、参数说明和箱号前缀示例。
- `server/seeds/{ports.zh-CN,shipping-lines.zh-CN,shipping-agents}.md`：已读，配套静态文件数量和摘要核对，不刷新原始编制/验收日期。

## 已执行检查

- `git diff --check -- server .trellis/spec/server/backend` 通过。
- 59 份后端 Markdown 的字面量本地内联链接校验：103 个目标、0 失效；围栏代码示例、远程地址和占位符排除，锚点由主会话最终全仓检查。
- CSV 只读计数：ports.zh-CN.csv 17,524 行、shipping-lines.zh-CN.csv 360 行；船代 JSON SHA-256 与来源文档一致。
- 修改的根 pnpm scripts、make 目标和生成输出均按源码核对；主会话只读 `go -C server generate -n` 实测 server 根目录无 Go 文件并退出 1，quality-guidelines 已补正确的 `go -C server generate ./...`。未真实执行生成器、apply、迁移或外部消息发送。

## 限制与后续

- 纯文档修改没有运行 Go/Vitest、构建、全量门禁、数据库或云端/钉钉验收；这些不是本轮通过事实。
- 保留规范中的业务要求，不因某个实现可能尚未完全覆盖就削弱约束；本轮对照关键行为与入口，不保证所有历史测试名、行号及字段矩阵逐项执行。
- Git 提交、ADR/domain 跨文档整合和最终独立 review 由主会话完成。
