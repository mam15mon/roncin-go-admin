# 海运出口单证版本、订单锁与换单实施计划

## 实施约束

- Agy 从本任务目录开始，依次完整读取 `implement.jsonl` 及其全部引用、`prd.md`、
  `design.md`、本文件和根 `AGENTS.md`。
- 本阶段同时涉及 Schema、跨聚合事务、全量写入口门禁、固定锁序、管理员安全分支和钉钉
  原生审批，按仓库约定使用 `gemini-3.8-flash-high`，新建独立 Agy 会话。
- 只实现阶段 5；禁止实现阶段 6 共享费用分摊、自动红冲、核销撤销、提成重算，也禁止新增
  SI/VGM/舱单人工状态或伪造外部回执。
- 不添加自动解锁、机器人消息冒充审批、管理员自动进入审批人、用户名识别系统账号、盲目
  重试未知钉钉结果、虚拟 HBL、旧模型双写或其他未批准兜底。
- 禁止 TDD；先完成每组功能，再按本计划补针对性单元、层级和 PostgreSQL 并发测试。
- Agy 可以编辑、生成和测试，但不得执行 git commit、push、merge、rebase、reset、数据库
  备份或 Trellis finish。每组实现后由 Codex 检查真实差异、独立 `trellis-check` 并提交。

## 实施步骤

### 1. Schema、枚举、边与正式迁移

- `User` 增加不可变 `is_bootstrap_admin`，普通管理接口不接受该字段；`bootstrap-admin` 创建
  唯一初始化账号时显式设为 `true`。
- `Order` 增加 `locked_by`、`lock_generation` 和必要边，保留 `locked_at`、`version`。
- 新建 `OrderLockRecord`、`OrderLockHouseBillSnapshot`、`OrderUnlockRequest`、
  `OrderUnlockApproverCandidate`，落实代次、请求状态、路由、幂等、活动请求部分唯一和只写
  一次的解锁事实。
- 新建 `SeaMasterBillVersion`、`SeaHouseBillVersion`，复用完整单证内容字段，身份表增加最近
  有效版本指针；新增作废事件与 HBL Switch 事件/替代链。
- 新建 `DingTalkApprovalDispatch` 和 `DingTalkApprovalInboxEvent`；扩展
  `BackgroundTaskKind`，不改变普通 `NotificationDelivery` 的含义。
- 为新实体补 Organization、Order、User、MBL、HBL、BackgroundTask 等必要边和索引；正式
  SQL 迁移核对 FK、枚举 CHECK、互斥单证 CHECK、部分唯一和循环当前版本外键创建顺序。
- 迁移不按用户名或角色猜测 bootstrap admin，不做历史回填。开发库如需重建，使用用户已
  授权的清空流程后重新迁移并运行 bootstrap。

### 2. 权限、配置、Proto 与生成物

- 权限 Manifest 新增仅适用于 SE 的 lock、amend、void、switch，
  设置读取/编辑依赖并生成前端权限键。
- 新增 `order_lock.proto`：锁摘要、锁定、解锁请求、请求列表/详情、真实处理路径和状态，包含
  “已同意待本地生效”和“被直接解锁取代”的可观察状态。
- 扩展单证契约：版本列表/详情、改单预览/执行、作废预览/执行、Switch 预览/执行和事件历史。
- 全部写请求包含 expected order/document version 和 idempotency key；分页复用最大 200。
- `order/v1/error_reason.proto` 增加设计中的稳定 reason；结构化 metadata 能返回锁定订单号、
  代次、配置缺失和下游阻断类型。
- `Security.DingTalk` 增加审批 process code 与事件回调配置；示例配置只保留空值/占位说明。
- 运行 API、config、Ent/Wire、permission keys 和 Web client 生成，不手改任何生成文件。

### 3. 业务角色资格与三路解锁用例

- 新增可复用的锁定/解锁角色资格查询，严格要求启用 User/Membership/Role/Assignment、订单
  组织数据范围、非 `administrator` 角色显式持有 SE lock 权限，并排除 bootstrap admin。
- `LockOrder` 除中间件权限外再次验证实际业务角色资格；bootstrap admin 不能用全量权限充当
  日常锁单人。
- 实现 `RequestOrderUnlock` 的固定分流：bootstrap admin 紧急直解 → 业务角色成员直解 →
  普通订单编辑人钉钉审批。真实员工 administrator 未加入业务角色时不得进入前两条。
- 同步直解在一个共享事务内完成请求、Order 解锁、LockRecord 关闭和审计；管理员原因可空，
  不自动填充虚假原因。
- 普通申请解析全部有效成员，严格验证申请人及全部候选 DingTalk UserID，保存或签候选快照、
  审批请求和 outbox；没有成员或任一相关用户未绑定时记录配置失败，不缩小候选集或加入
  bootstrap admin。
- 同代次已有活动审批时普通编辑人返回既有请求；角色成员/admin 直接解锁先将旧请求置
  STALE 并建立 superseded 关联，保证旧钉钉回调不能误生效。
- 实现幂等同键同/异指纹、同代次单活动请求、提交后普通上下文重读和稳定错误。

### 4. 订单锁定与不可变单证版本

- 实现完整锁定事务：Order → MBL → Link → HBL 固定锁序、版本重验、MBL/HBL 版本创建或
  合法复用、代次递增、锁定记录/HBL 版本关联和同事务审计。
- 版本包含身份、签发主体、号码、全部单证内容和 MBL 权威航程；历史 API 只从版本读取。
- DIRECT 锁定只保存 MBL 版本，不创建 HBL；HOUSE 保存当前操作票全部有效 HBL 版本。
- 共享 MBL 已有相同 source version/content hash 时复用；工作内容变化时追加版本并更新身份
  当前版本指针，绝不覆盖旧版本。
- 审计或版本写失败时锁定整体回滚；提交后重读响应。

### 5. 全量订单业务写入门禁与共享 MBL 门禁

- 抽取 data 层统一 `lockOrderAndEnsureBusinessEditable` 原语：首次读取 Order 直接
  `FOR UPDATE`、校验预期版本和 `locked_at`，返回稳定 reason/metadata。
- 逐一接入订单、状态/终止/结案、标签、里程碑、附件、人员、箱计划/实际箱、货物、装运
  单证、POD、异常、海运单证、箱货分配、费用、拆票、改配和身份更正所有写入口。
- 不修改只读查询；不把业务锁当成财务门禁替代，解锁后仍继续执行原有费用/账单/核销/提成
  检查。
- 修改共享 MBL 前按 UUID 锁全部活动成员 Order，再锁 MBL；任一成员锁定即返回数量和稳定
  排序订单号并完整回滚。
- 对齐阶段 4 相关入口的 Order-first 锁序；同类多行按 UUID 排序，禁止 SHARE 升级和死锁
  自动重试。

### 6. 钉钉原生审批适配器、Worker 与回调

- 新增 `DingTalkApprovalGateway`，使用当前官方 OA 审批实例创建/查询契约；与现有登录、
  `SendText` 保持接口隔离。
- Worker 复用 `BackgroundTask` 领取/租约，读取 `DingTalkApprovalDispatch` 后创建或签审批；
  成功保存外部实例 ID 并把请求置 `PENDING_APPROVAL`。
- 将外部失败严格分类为可安全重试、明确失败和结果未知；未知结果置
  `DISPATCH_UNKNOWN` 且不盲目创建第二实例。
- 增加钉钉事件专用原始 HTTP handler：验签、解密、企业/事件校验、Inbox 幂等落库和协议
  应答；在 SPA fallback 前注册，不走浏览器会话鉴权。
- Inbox Worker 查询权威审批详情；批准先持久化 `APPROVED_PENDING_APPLY`，再复验当前锁定
  代次、预期版本、候选快照和审批人当前业务角色资格后原子解锁；本地失败只重试生效，不
  重建钉钉实例；拒绝保持锁定；旧代次/旧版本/被直接解锁取代置 STALE；重复事件幂等。
- 验签、解析和外部 payload 类型停留在 data/integration 层，biz 不依赖钉钉 SDK 类型。

### 7. 单改、作废、Switch B/L 与历史查询

- 实现 MBL/HBL 单改 Preview/Execute：订单未锁、权限、预期版本、非空差异、原因和财务门禁；
  执行时追加版本、切换当前指针并审计。
- 实现作废 Preview/Execute：不物理删除身份/版本，追加 VOID 版本/事件并进入 `VOIDED`。
- 实现 HBL Switch Preview/Execute：同订单/同当前 MBL 下创建真实新 HBL 和首个版本，旧 HBL
  进入 `REPLACED`，替代链只保留一个当前末端。
- HBL 号码和签发主体继续复用阶段 2 规范化/唯一约束；Switch 不创建订单、虚拟 HBL 或自动
  复制财务事实。
- 下游财务事实存在时按设计明确阻断；改单/作废/Switch 不创建自身审批状态，未锁定且有权限
  时由用户确认后直接执行。
- 版本和事件列表稳定排序，历史 DTO 始终取不可变版本，不回读当前工作字段。

### 8. Web 订单详情、状态抽屉与单证历史

- 详情头显示锁定摘要；根据服务端允许动作和权限分别显示锁定、直接解锁、紧急解锁、申请
  解锁。管理员紧急解锁原因可选并展示审计提醒。
- 锁定时保持信息可读、禁用所有本地写入口；后端错误仍统一触发锁定摘要和申请入口。
- 增加解锁请求抽屉，准确映射待派发、审批中、已同意待本地生效、配置失败、派发失败、
  结果未知、拒绝、过期和已解锁；不由前端猜审批结果。
- 共享 MBL 阻断展示具体订单数量/号码，可编辑订单提供逐票申请解锁入口。
- 提单区块保持默认展开；版本、差异、作废和 Switch 历史放抽屉。Preview 成功并展示最终
  差异后才允许 Execute。
- 只消费 OpenAPI 生成客户端、权限生成键和服务端动作摘要，不拼接 URL、不复制资格规则。

### 9. 实现完成后补测试

- biz：三路分流、bootstrap 优先、真实 administrator 边界、可选解锁原因、幂等、状态机、
  版本追加/复用、单改差异、作废和 Switch 链。
- data：角色资格全部条件、候选快照、代次/版本、部分唯一、共享 MBL、全量写入口锁门禁、
  财务门禁不被解锁绕过、审计失败回滚。
- DingTalk：假 HTTP 服务覆盖创建成功、明确拒绝、可安全重试、未知结果、权威查询、验签失败、
  重复/乱序/旧代次回调和审批人资格撤销。
- service/API：权限组合、DTO、分页 1/200/201、错误 metadata、静态回调路由和敏感信息边界。
- web：三种解锁入口、可选原因、状态抽屉、锁定表单、共享订单号、默认展开与版本差异。
- PostgreSQL：锁与普通写竞争、双锁/双解锁、回调与直解竞争、共享 MBL 多成员、历史重现、
  替代链、幂等重放和所有关键中途失败，证明无死锁、误解锁、双当前版本和部分更新。

## 分组提交与检查门

建议按三组可验证改动提交，后一组以前一组检查通过为前提：

1. `feat: 增加订单锁定与单证不可变版本`
   - Schema/迁移、权限/契约、角色资格、锁定/直解、全量门禁、版本与基础 UI。
2. `feat: 接入钉钉订单解锁审批`
   - 原生审批适配器、outbox/inbox、回调、状态抽屉和专项测试。
3. `feat: 增加海运提单改单作废与换单`
   - 单改、作废、Switch、版本历史 UI 和专项测试。

每组 Agy 完成后都由 Codex 查看真实 Git diff、运行独立 `trellis-check`、修复 P0/P1、重跑该组
风险测试并提交；不能把未验证的三组合并后一次性押注。

## 验证清单

- [ ] `make -C server api`
- [ ] `make -C server config`
- [ ] `go -C server generate ./...`
- [ ] `pnpm run generate:web-client`
- [ ] `pnpm run generate:permission-keys`
- [ ] 所有生成命令重跑无新增差异。
- [ ] `go -C server vet ./...`
- [ ] `go -C server test ./...`
- [ ] 专用真实 PostgreSQL 集成/并发测试使用隔离 Schema 且不得 SKIP。
- [ ] `pnpm --dir web test`
- [ ] `pnpm --dir web tsc`
- [ ] `pnpm --dir web biome:lint`
- [ ] `git diff --check`
- [ ] 正式迁移应用到开发库，核对列、FK、CHECK、部分唯一、状态和索引。
- [ ] 使用假钉钉服务器验证创建、查询、回调验签与重复事件，不调用真实生产钉钉。

## 主会话复核重点

- bootstrap admin 是否只获得紧急直接解锁，不被计为业务角色、不进入候选、不能日常锁单；
  操作人/时间必记且原因确实可选。
- 真实 administrator 是否只有显式加入非管理员锁定角色后才成为业务审批人。
- 角色直解是否与人数无关且不创建钉钉实例；普通编辑人是否始终进入所有有效角色成员或签。
- 钉钉网络调用是否完全在事务外；未知结果是否停止盲重试；回调是否验签、查询权威结果、
  复验代次/版本/审批人资格后才解锁。
- 所有业务写是否真正从 Order-first 门禁进入，锁定与并发写是否没有检查后写入窗口。
- 共享 MBL 是否检查全部活动成员 Order，返回具体号码并完整回滚。
- 解锁是否只解除订单业务锁，未绕过确认费用、账单、发票、核销和提成门禁。
- 锁定是否原子创建完整 MBL/HBL 版本；历史引用是否只读版本 ID，DIRECT 是否零虚拟 HBL。
- 单改、作废、Switch 是否无第二审批流、无空版本、无物理删除、无财务自动调整。
- 是否完全没有新增 SI/VGM/舱单伪状态、机器人消息审批、用户名特判或兼容兜底。

## 回滚点

每组提交独立可审阅，但第二、三组依赖第一组 Schema 与契约。若第一组失败，必须同时回滚
Schema、迁移、Proto、生成物、后端、页面和测试；若钉钉组失败，可停用审批配置但不能把
普通申请静默改为直接解锁；若改单/Switch 组失败，不得删除已经形成的不可变版本历史。
