# 根目录与 docs 文档现状审计

- Query: 核对 README、PLAN 和 docs 全部 Markdown 是否符合当前仓库实现。
- Scope: internal，静态审计；未运行数据库、云端、浏览器或验收写入。
- Date: 2026-11-09

## 已读文档清单

已阅读根目录 `README.md`、`PLAN.md` 及 docs 下全部 12 份 Markdown：

- `docs/local-development.md`
- `docs/airport-port-sync.md`
- `docs/cross-repository-transactions.md`
- `docs/eyun-config-management.md`
- `docs/migration-matrix.md`
- `docs/order-attachment-oss.md`
- `docs/plans/业务标签建设计划.md`
- `docs/plans/企业资源备忘录建设计划.md`
- `docs/plans/客户页面.md`
- `docs/plans/提成管理增强计划.md`
- `docs/plans/海运出口订单闭环验收.md`
- `docs/plans/费用明细批量转账单.md`

## 已确认差异与处理建议

### G01：README 错称空运未开放

- 文档：`README.md:9`、`docs/plans/海运出口订单闭环验收.md:6`。
- 实现：`web/src/pages/orders/order-kinds/registry.ts:1` 和 `:11` 注册 air-export；`web/src/pages/orders/order-kinds/air-export/definition.ts:6` 至 `:14` 明确 AE 表单和详情；`web/config/routes.ts:140` 空运出口入口，`:148` 共享主运单详情。
- 建议：根 README 同列海运出口和空运出口已开放，其余线路按注册表现说明。海运验收文档保留 SE 专项范围，但改为“空运出口另行验收”，不能称无入口。

### G02：README 把退役标签当核心能力

- 文档：`README.md:10`。
- 实现：`server/api/enterprise_resource/v1/enterprise_resource.proto:96` 保留已删除 TAG 枚举名；`server/api/order/v1/order.proto:399` reserved tags；`server/api/finance/v1/settlement.proto:258` 和 `:301` reserved tag_ids/tags；`server/migrations/20261109100000_retire_business_tags.sql:29` 至 `:39` 移除五张专属表和 TAG 资源。
- 建议：删除当前核心能力中的标签项，增加状态说明及业务标签退役规范入口；历史建设计划保留历史正文但首页明确“已退役，不作为继续实施依据”。

### G03：README Go 最低版本滞后

- 文档：`README.md:32`、`:80` 写 Go 1.26.9。
- 实现：`server/go.mod:3` 为 go 1.26.10。
- 建议：写当前 go.mod 版本或以 go.mod 为准，避免两处手动版本分叉。

### G04：README 的 JWT Session 不符本站会话实现

- 文档：`README.md:34`。
- 实现：`server/internal/biz/auth.go:690` 创建随机会话令牌并持久化摘要；`:707` 至 `:724` 按摘要查库及续期；`:807` 至 `:813` 用 crypto/rand 生成随机令牌并 hex 编码。
- 建议：改“数据库会话（随机令牌摘要）”，不能因为 go.mod 存在 JWT 依赖就声称本站登录使用 JWT Session。

### G05：本地开发错误描述 bootstrap 为不可重复

- 文档：`docs/local-development.md:98` 至 `:99` 声称有用户即拒绝。
- 实现：`server/cmd/bootstrap-admin/main.go:95` 至 `:117` 应用迁移后调用 BootstrapAdministrator；`server/internal/data/bootstrap_administrator.go:178` 起支持定位现有唯一初始管理员及补齐授权，`:191` 仅拒绝用户名与初始管理员不一致；`:223` 首次创建要求凭据。
- 建议：同步根 README 的首次空库创建、重复只补授权、停用状态保留和冲突拒绝口径，尽量链接专门规范，避免复制整段。

### G06：本地开发错误描述真实库测试的建表与清理

- 文档：`docs/local-development.md:72` 至 `:73` 写自动建表、唯一后缀组织及清理业务数据。
- 实现：`server/internal/data/integration_test_helper_test.go:26` 至 `:27` 明确正式 SQL；`:52` 至 `:64` 为每测试创建随机隔离 Schema、结束删除该 Schema；`:80` 之后收集正式 migrations；未配置连接时 `:30` 至 `:33` 跳过。
- 建议：改“在专用集成库创建隔离 Schema，按顺序执行正式 SQL 迁移，结束清理隔离 Schema”；不得暗示 Ent Schema.Create。文档中的三个测试名仍存在，本轮无需改名。

### G07：本地开发与机场港口方案的组织级同步过时

- 文档：`docs/local-development.md:129` 至后续主数据同步节；`docs/airport-port-sync.md` 的 D1/D3、CLI org-code 参数、已知限制 3 和实施状态仍为 HQ 单组织。
- 实现：`server/internal/data/ent/schema/airport.go:40` iata_code 全局唯一、`:41` icao_code 全局唯一；`server/internal/data/ent/schema/port.go:35` un_locode 全局唯一；`server/internal/data/industry_reference_sync.go:97` 至 `:120` CheckAirports/CheckPorts 全局读取，无组织过滤；`server/cmd/sync-airports/main.go:96` 至 `:101` 无 org-code；`server/cmd/sync-unlocode/main.go:126` 至 `:156` 同样无 org-code。
- 建议：机场港口方案改为公共全局目录现状，系统工作台维护权，移除失效 org-code 用法与“尚待多组织共享”描述；2026-08-23 向 HQ 写入数量留作历史验收，不冒充现状。

### G08：同步数据来源与中文补齐策略过时

- 文档：`docs/airport-port-sync.md` D5 仍写 OurAirports 旧下载地址、港口必须手动下载指定包、航司/船司不接入外部源；`:D3` 说港口首次不写中文；`docs/local-development.md` 同步节也称必须显式提供 ZIP 与版本。
- 实现：`server/cmd/sync-airports/main.go:24` 实际下载 davidmegginson.github.io 归档源；`server/cmd/sync-unlocode/main.go:128` 至 `:156` 默认找 seeds/loc251csv.zip 并读取配套 ports.zh-CN.csv；`:83` 至 `:89` 附加中文映射；`server/cmd/sync-shipping-lines/main.go:114` 起为 SMDG xlsx 与中文对照；`server/cmd/sync-airlines/main.go:26` OpenFlights 地址、`:98` 起支持数据源参数，默认优先归档。
- 建议：明确当前根脚本默认带 -apply；港口、船司、航司使用归档快照及受控补齐，不混同为权威联网实时自动同步；具体中文覆盖/保留规则按对应仓储核查后写，不能全局说从不写中文。

### G09：跨仓储文档保留了禁止继续扩散的 d.db 指南

- 文档：`docs/cross-repository-transactions.md:113` 写只改试点仓储、不全仓替换 d.db。
- 实现：`server/internal/data/transaction.go:100` 至 `:108` 提供统一 Data.client(ctx)，`:105` 已关闭上下文立即错误；根 AGENTS 要求仓储必须经 Data.client(ctx)。
- 建议：将“只改试点所需”明确为当时实施记录，现行仓储访问统一用 Data.client(ctx)，说明 shared tx 回调只校验/写入、完整响应提交后重读以及 withSQLTx 不加入共享事务。

### G10：海运闭环验收仍把独立费用审核当建账前置

- 文档：`docs/plans/海运出口订单闭环验收.md:45`、`:52`、`:56`、`:78`。
- 实现：`server/internal/data/order_fee_review.go:203` 至 `:211` 人工订单审核锁决定可编辑，CanBill 取建账权限，`:529` 费用按当前/需建账/无账单占用/无待审变动判定 CanBill；`server/internal/biz/order_fee_correction.go:37` 至 `:41` 当前有效费用与需记账语义；根 AGENTS 与 order-fee-review/order-fee-correction 规范明确审核和订单锁已统一，建账无需预审。
- 建议：验收分开验证“未锁且未建账可改、按权限直接建账”和“人工审核锁固化内容利润、锁后更正需补审”；不能继续要求先提交独立整票审核。待审影响按源费用粒度描述，不能写所有后续一概阻断。

### G11：批量转账单旧方案使用已废止的费用状态

- 文档：`docs/plans/费用明细批量转账单.md` 状态 CONFIRMED/BILLED、按老 split_by_* 字段及已确认费用选择。
- 实现：`server/internal/data/fee_bill_association.go:16` 按未取消账单的活动行推导占用；`server/internal/biz/order_fee_test.go:80` 至 `:82` 明确保存即未建账且可建账；`server/internal/data/finance_bill.go:399` 与 `:574` 返回 CanBill；当前 `PreviewFinanceBillBatchInput` 使用 GroupingPolicy 与 GroupConfigs（`server/internal/data/finance_bill_batch_integration_test.go:213` 至 `:239` 可查调用）。
- 建议：旧实施方案补当前契约摘要与规范入口，若作为现行操作说明则改成当前可建账费用、活动行投影、当前分组策略；不要改旧验收记录为新结果。

### G12：业务标签计划、企业资源计划、根 PLAN 与迁移矩阵需要统一历史标识

- 文档：`PLAN.md:41` 标“当前状态 2026-08-30”，`:60` 写标签已交付；`docs/migration-matrix.md:40` 一带 D07/D12 仍含标签现行能力；`docs/plans/业务标签建设计划.md:3` 至 `:10` 说本文指导设计实施；`docs/plans/企业资源备忘录建设计划.md:7`、`:29`、`:317` 等保留 TAG；竞品笔记 §9 建议复制标签。
- 实现：见 G02。`server/migrations/20261016100000_retire_role_data_scope.sql:2` 至 `:4` 还明确停用 self 并删角色 data_scope，与 PLAN/迁移矩阵的角色范围现行描述冲突。
- 建议：历史计划首页加最新状态、日期、现行契约链接；PLAN 明确当前里程碑状态导航和早期阶段历史含义，避免重写历史阶段正文；竞品笔记保留 2026-08-29 观察事实但补“非当前功能清单，标签已退役”。迁移矩阵的当前状态、财务金额精度及工作台授权摘要必须更新。

### G13：企业资源旧计划的图片删除顺序与现实现冲突

- 文档：`docs/plans/企业资源备忘录建设计划.md:655` 至 `:656` 声称先删对象，再事务删记录。
- 实现：`server/internal/data/enterprise_resource.go:289` 至 `:343` 在同事务删资源及关联、登记对象删除任务、写审计，`:337` 至 `:340` enqueueObjectStorageDeletion，外部删除由 worker 执行。`docs/cross-repository-transactions.md` 已描述新顺序。
- 建议：在该设计决策处直接标明后续已替换为事务任务，并链接跨仓储文档；不能只在页头加日期后留下相反的无限定“本期采用”指令。

### G14：提成计划仍称未实现已有字段与导出

- 文档：`docs/plans/提成管理增强计划.md:12` 说不代表已完成，§3 声称无 CNY 字段/归属日/导出权限。
- 实现：`server/internal/data/ent/schema/finance_commission.go:52` 至 `:56` 归属日期和 CNY 快照，`:113` 月份索引；`server/api/finance/v1/settlement.proto:133` ExportCommissions RPC；`:896` 月份筛选；`:1046` 主单字段，`:1047` 至 `:1051` 导出契约；`server/internal/access/manifest.go:69` 独立导出权限。
- 建议：改为历史设计基线，补“当前接口、模型已具备该能力”与代码入口；代码存在不等于实际生产/验收通过，不能把完成定义全勾选。存量数据重建须授权的旧文字也应明确不得自动执行。

### G15：客户计划的“待解决问题”已有实现

- 文档：`docs/plans/客户页面.md` §4.3 列 Promise.allSettled、币种备用数据、两段式主体/结算保存仍待实现。
- 实现：`web/src/pages/partners/partner-detail.tsx:184` 起 React Query，`:225`、`:236`、`:241` 分模块请求；`:608` 至 `:617` 将 settlementRule 嵌入角色输入，`:620` 起单次 UpdatePartner（创建同理，需按实际语句核对）。
- 建议：整份作为早期设计记录加现状摘要，不再把问题清单当当前缺陷清单；其他人员、行政区划、金额字段逐项据代码确定，不笼统宣称所有规划已完成。

### G16：迁移矩阵金额精度草案过时

- 文档：`docs/migration-matrix.md` M-005 现行草案写金额 numeric(18,6)。
- 实现：`server/internal/data/ent/schema/order_fee.go:39`、`:41`、`:42`、`:53` 金额 numeric(28,8)，`:44` 汇率 numeric(18,8)；`server/internal/data/ent/schema/finance_bill.go:46` 至 `:49` 同金额精度。
- 建议：当前摘要同步金额/汇率/数量各自精度，展示层统一两位；旧草案保留时明确已被现行 schema 和格式规范替代。

## 已审阅但没有确证实现差异的现行文档

- `docs/order-attachment-oss.md`：核心链路与现行 OSS 规范一致；其中具体 bucket 名和开发机私网 IP 属当时部署信息，不可仅凭仓库断言目前仍生效，建议改为环境示例或标注配置核对日期。真实环境运行结果本轮未核实。
- `docs/eyun-config-management.md`：历史竞品调研记录本体不应按当前第三方页面重写；仅需增加它与当前项目功能状态的边界。

## 相关规范与实施核查入口

- `.trellis/spec/server/backend/business-tag-retirement.md`
- `.trellis/spec/server/backend/air-export-contract.md`
- `.trellis/spec/server/backend/organization-shared-masterdata.md`
- `.trellis/spec/server/backend/company-admin-provisioning.md`
- `.trellis/spec/server/backend/order-fee-review.md`
- `.trellis/spec/server/backend/order-fee-correction.md`
- `.trellis/spec/web/frontend/finance-current-workspace.md`
- `.trellis/spec/web/frontend/state-management.md`

## 不确定项与本轮限制

- 未检查外部线上域名、真实 OSS、钉钉凭据或第三方竞品当前页面，不声称上线已完成。
- 未重跑历史验收脚本，不修改原有已记录的历史通过事实；所有当前验收结果仍需明确执行。
- README 的“表单草稿按工作台隔离”需前端研究进一步按新建/详情区分，不能只根据变量名 draftScope 推断持久草稿仍存在。
- 这份报告不作文件全局链接校验，交由主会话统一盘点；未扩展查阅用户未跟踪任务内容。
