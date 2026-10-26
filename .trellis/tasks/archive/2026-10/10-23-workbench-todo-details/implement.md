# 实施与验证计划

## 启动条件

- [x] 用户于 2026-10-26 明确批准修订任务、实施、review、合并、安全清理分支和推送。
- [x] 刷新 git 状态、当前任务和分支，保留其他任务改动；在专属分支工作。
- [x] 加载前后端规范、roncin-web-stack 和 Ant Design 指南。

## 实施步骤

1. [x] 修改工作台 Proto：两类分页接口；补充原异常响应的可读名称。
2. [x] 增加 biz 投影、仓储接口、分页校验；同步受影响桩实现。
3. [x] 完成 service 会话/权限映射（逐笔费用必须 fee.read，lock 不替代），data 共享需记账/未替代/无有效账单谓词、分页与批量名称/阻塞状态装配；摘要同步使用相同谓词。
4. [x] `make -C server api`、`pnpm run generate:web-client`，检查生成差异。
5. [x] 完成工作台入口、两类分页抽屉、错误/空态和刷新。
6. [x] 完成订单异常入口意图的一次性消费、独立权限及名称显示；保留现有锁门禁。
7. [x] 功能实现后补充定向验证；独立 review 后修正问题并完成全栈门禁。
8. [x] 每组可验证改动按 Conventional Commits 提交，更新必要契约说明和验收记录。

## 验证矩阵

| 维度 | 场景 | 验收 |
| --- | --- | --- |
| biz/service | 分页默认值、0/200/201 边界、固定会话身份、业务权限映射 | AC5、AC6 |
| PostgreSQL | 本人/同事、当前/其他公司、SE/AE、无订单/费用/异常读取权限、管理员个人协作 | AC5 |
| PostgreSQL | 无账单、有效草稿、非活动行、已取消账单；更正替代/无需记账/冲回记账行；ACTIVE/RESOLVED/移除异常 | AC4 |
| PostgreSQL / Web | 锁定、退关/结案、源费待补审、无关费用及只有 lock 无 fee.read 的人员 | AC9 |
| PostgreSQL | 同票多项、同时间跨页、名称投影与停用类型、摘要列表同口径 | AC1、AC6、AC7 |
| Web | 类型切换、分页、空态、失败重试、显示格式、SE/AE 链接 | AC1、AC2、AC6、AC7 |
| 路由与权限 | 异常自动展开、参数消费、不重复弹出、已有页签；只读/解决/删除/锁定 | AC2、AC3 |
| 工作区 | 换组织/用户、旧请求迟到、返回工作台和重开后最新计数 | AC4、AC5、AC6 |

开发期先执行直接受影响测试文件及修改文件 Biome，再按类型影响执行 tsc。新测试文件随实现落地：

```bash
go -C server test -p 32 ./internal/biz ./internal/service -run 'Workbench|AbnormalCase'
go -C server test -p 32 ./internal/data -run 'Workbench.*Postgres|OrderAbnormalCase.*Postgres' -count=1
pnpm --dir web exec vitest run src/pages/workbench/WorkbenchSideCards.test.tsx src/pages/workbench/index.test.tsx
pnpm --dir web test:changed
pnpm --dir web tsc
git diff --check
```

集成验证使用 `RONCIN_INTEGRATION_DATABASE_SOURCE`，通过正式迁移建立独立测试 Schema；不输出连接串、不清空开发库，缺配置造成 skip 必须如实记录。重型组件测试超过 20 秒按功能拆分。

最终验收因涉及权限投影、接口及生成物运行 `pnpm run check:fast`，记录耗时/通过与失败数；本次不固定执行生产构建。

## 收尾

- [x] 变更仅属本任务，无秘密/认证材料/临时产物，生成物无手改。
- [x] 记录 review、实际测试和未运行验证原因；归档前确认验收完成。
- [x] 根据用户批准的交付方式处理分支，不携带其他任务改动或删除其分支。

## 交付顺序（2026-10-26 已授权）

- 在 `codex/workbench-todo-details` 实施并提交，完成独立审查与全栈门禁后合并 `main`。
- 核验本地及远端分支是否已合并、是否有独有提交或被工作区占用，仅删除已证明安全的非主分支；不强制删除未合并工作。
- 归档本任务并记录验收后推送 `main`，核对远端提交与工作区状态。

## 实际实施与审查（2026-10-26）

- 两类分页 Proto、biz/service/data、工作台抽屉、订单异常直达及全部生成物已同步；无 Schema 迁移。
- 摘要/列表共用本人公司及品类读取交集、accounting/effectiveBillLine 查询；冲回方向和源费待补审沿用现有规则。
- 异常列表、登记及解决响应补充可读名称；修复原 service 向 Mark/Resolve 传反操作者与订单 ID，并有回归验证。
- 独立 `trellis-check` 审查发现并修复两处权限强依赖：登记候选误用管理列表改为登录可读 SearchOptions；
  详情聚合的人员列表与创建候选按各自权限请求，权限进入缓存键，无权区块隐藏、无候选字段只读。
- 工作台使用真实 ProTable 验证分页、重试及链接；真实 OrganizationWorkspace 验证切公司首帧清理与迟到隔离；
  真实 Router + 详情 Hook 验证 SE/AE 最小异常读取权限、重复直达与权限收回后的缓存。
- 新增 Router Hook 后更新相邻面包屑测试的真实 Router 包装，保留原业务断言。
- 并行列设置改动由另一窗口独立提交 `9a03f7b6`；其新增 ConfigProvider 引用需要核销测试补齐桩，
  已单独验证 7 项并提交 `67936c82`，没有改动该窗口产品实现。

## 实际验证

- `make -C server api`、`pnpm run generate:web-client` 完成；服务端生成权限规则与 Proto 一致性测试通过。
- 专用临时 PostgreSQL 经正式迁移验证，开发库未清空/重建；临时实例已停止并清理。受影响 biz/service/data/server 全通过，data 用时 15.849 秒。
  覆盖本人/组织/业务权限、账单草稿占用与取消、更正/冲回、稳定分页、锁/退关/结案、异常解决/移除、
  停用名称及公共候选中文/英文/编码/全拼/首字母检索。
- 完整 `pnpm run check:server` 通过（全部 Go 测试、vet、分层、重复扫描、Proto、漏洞扫描）。
- 开发期工作台 23 项、orders 61 项及公共订单模板 5 项定向回归通过；独立审查补充的组合/相邻 25 项通过，tsc、Biome 与 Ant Design lint 通过。
- 初次 `check:fast` 前端 10 项失败：本任务 3 项缺 Router 上下文已修，其他窗口 4 项 ConfigProvider 桩已修，
  原有确认框卸载和费用列的 3 项为并发超时/相邻失败，原样定向重跑通过。生成中途启动的一次后端复跑有临时契约不一致，
  完成生成后完整后端已通过。最终前端按全覆盖、4 worker 运行，以降低资源争抢；不删测试或修改超时时间。
- 本次未修改构建配置、依赖或生产入口，按仓库规则不追加生产构建。未执行浏览器端到端测试；UI 由上述真实组件/路由组合验证。

- 最终前端完整门禁通过：权限键/Proto 常量、模块架构、重复扫描、Biome、tsc 与全部 Vitest；
  `vitest run --maxWorkers=4` 为 303 个文件通过、1 个既有文件跳过，1651 项通过、12 项既有跳过，耗时 385.79 秒。
  Biome 有既有 20 条 warning、7 条 info，无 error；部分旧测试输出未 mock 网络的 ECONNRESET，退出码为 0。
- 业务提交：`1bec915e`；最终验收后按已授权流程归档、合并 main、仅删除已合并且无工作区占用的分支、推送并核对远端。
