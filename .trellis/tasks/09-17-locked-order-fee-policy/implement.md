# 实施计划：锁单后费用补录与提成冲减调整

## 0. 开发前复核

- 完整读取 prd.md、design.md、本文件及 implement.jsonl 引用规范。
- 检查 git status，保留所有与本任务无关的用户改动。
- 核验普通费用门禁、直接解锁资格谓词、提成调整状态机和汇率解析入口，禁止复制第二套规则。
- 本功能跨越契约、Schema、权限、事务和前端，必须作为一个原子闭环交付，不得先上线可绕过锁但没有冲减建议的半成品。

## 1. Schema、领域对象与迁移

- 新增 OrderFeeSupplementRequest Ent Schema、边、状态枚举、版本字段和唯一索引。
- 给 OrderFee 增加补录申请来源关联。
- 给 FinanceCommissionAdjustment 增加 LOCKED_FEE_SUPPLEMENT 来源及补录申请关联。
- 在 internal/biz 增加补录申请领域对象、命令、错误和仓储接口；扩展调整来源类型。
- 生成 Ent 代码与正式迁移，保证数据库 CHECK、外键删除策略和唯一约束与 Schema 同源。
- 先完成实现，再补 Schema 元数据及真实 PostgreSQL 迁移测试；禁止 TDD。

## 2. 服务端补录申请与审批事务

- 在订单费用 Proto 源文件增加创建、列表、通过、驳回接口和 DTO。
- Service 只做 UUID、版本、分页和 DTO 转换；权限注解分别使用 fee.create/read 与目标订单 lock 操作。
- Biz 用例负责状态机、参数规则、审计语义和共享事务编排。
- Data 层实现申请持久化、直接解锁资格复用、专用费用创建和固定锁序。
- 审批通过在同一共享事务中完成费用、冲减草稿、申请终态、审计和通知 outbox。
- 普通费用入口保持原门禁，禁止增加公开的 skipLock、force 或布尔绕过参数。

针对性验证：

- 未锁订单拒绝走补录入口；
- 业务锁、财务锁和双锁订单允许提交；
- 无 fee.create 不可提交，无实时 lock grant 不可审批；
- 合格发起人可以自行审批；
- 版本冲突、旧锁代次、重复幂等键和并发审批均稳定失败或返回原结果；
- 驳回不创建费用；
- 通过创建且只创建一条 CONFIRMED 费用，订单仍保持锁定；
- 任一步错误时费用、调整、申请和审计全部回滚。

## 3. 提成影响与现有调整复用

- 抽取或复用现有提成计算函数，以原提成线冻结的实现范围、比例和版本计算本次补录的边际差额。
- 只为毛利口径的负向差额创建 DECREASE + DRAFT + LOCKED_FEE_SUPPLEMENT 调整。
- 按提成父单 UUID 固定顺序加锁；来源唯一键保证审批重试不重复创建。
- 草稿金额限制在原提成仍可冲减范围内，理论超出额进入审计而非员工负债。
- 继续使用现有 ConfirmCommissionAdjustment 和 MarkCommissionAdjustmentPaid，不新增确认状态机。
- 增加可分页的调整列表查询，显式支持系统来源、状态、员工、组织和关键字过滤。

针对性验证：

- 已确认与已发放提成都能生成建议；
- DRAFT 和 CANCELLED 父提成不参与；
- 收入口径补录应付不生成建议；
- 毛利口径按部分实现范围计算，不把未实现收入对应成本提前全部冲减；
- 多员工、多父提成分别生成，金额与来源可追溯；
- 零差额不生成；
- 两笔补录并发审批不会重复或覆盖调整；
- 财务确认后变 CONFIRMED，标记已扣回后变 PAID；
- 人工调整导致额度不足时仍由现有状态机拒绝负数有效金额。

## 4. 前端订单费用体验

- 使用生成客户端接入补录申请接口。
- 在订单锁定或财务锁定时保留普通写入禁用，同时按 fee.create 能力显示【补录费用】。
- 复用费用表单字段，增加必填补录原因和“不会修改原费用”的说明。
- 展示申请历史、状态、发起人、审批人和生成费用；具备后端返回审批能力的用户可通过或驳回。
- 订单身份变化时清理申请、弹窗和异步状态，遵循页面复用隔离规范。
- 增加发起、自审、驳回、锁状态和迟到响应的定向测试。

## 5. 前端提成体验

- 在现有提成页增加【待处理冲减】视图，使用服务端分页接口，不在前端循环翻页。
- 默认只展示 DECREASE + DRAFT + LOCKED_FEE_SUPPLEMENT。
- 展示员工、订单、原提成、补录费用、建议金额、原因和时间；支持下钻来源。
- 【确认冲减】调用现有确认接口；CONFIRMED 后沿用现有【标记已扣回】。
- 明确状态文案：待处理不等于已确认，已确认不等于已扣回。
- 增加列表筛选、权限、确认冲突和刷新行为测试。

## 6. 生成与验证

契约和 Schema 修改后按顺序运行：

    make -C server api
    go -C server generate ./...
    pnpm run generate:web-client

只有新增或修改 Manifest 权限码时运行：

    pnpm run generate:permission-keys

开发期先运行受影响的定向测试：

    go -C server test ./internal/biz ./internal/data ./internal/service -run 'Test.*(FeeSupplement|CommissionAdjustment)' -count=1
    pnpm --dir web exec vitest run src/pages/orders/fees.test.tsx src/pages/finance/commissions/index.test.tsx
    pnpm --dir web exec biome check <本次修改的前端文件>
    git diff --check

事务、锁、唯一约束和迁移必须注入专用测试库执行真实 PostgreSQL 用例，并确认是 PASS 而不是 SKIP：

    RONCIN_INTEGRATION_DATABASE_SOURCE="<专用测试库连接串>" go -C server test -v ./internal/data -run 'Test.*FeeSupplement.*Postgres$' -count=1

任务最终验收执行一次：

    go -C server test ./...
    go -C server vet ./...
    pnpm --dir web tsc
    pnpm run check:web
    pnpm run check
    git diff --check

不把 pnpm run build 作为普通固定门禁；只有生产入口、依赖或构建配置变化时才运行。

## 7. 提交与独立检查

每完成一组可独立验证的修改，由主会话复核差异后提交：

1. feat: 增加锁单后费用补录与冲减建议后端
2. feat: 接入费用补录审批与待处理冲减页面

实现后独立检查：

- 普通费用门禁没有被放宽；
- 专用补录入口不能被当作通用绕过；
- 同一事务、固定锁序、乐观锁和唯一索引真实生效；
- 建议确实复用现有调整状态机，DRAFT、CONFIRMED、PAID 文案不混淆；
- 生成物来自生成器，迁移与 Ent Schema 同源；
- 所有验收条件均有代码和测试证据。

## 8. 停止条件

以下情况必须停止并回到规划，不自行扩大范围：

- 必须支持既有费用减额、负数费用、红字或已核销冲销；
- 客户要求系统自动控制工资、打款或跨月员工余额；
- 现有提成快照不足以复算锁后成本影响，且需要改变已批准计算口径；
- 需要清空、重建或自动修复数据库。
