# 全业务类型订单锁定与解锁实施计划

## 开始前检查

- [ ] 重新检查 `git status`，只接管本任务目录和明确列出的产品文件，不触碰用户的
      `clash-for-linux/`。
- [ ] 运行 `trellis-before-dev`，完整加载实施清单中的服务端、前端、事务、锁和生成规范。
- [ ] 确认当前 Trellis 任务仍为 `universal-order-lock`，且没有其他客户端或 agy 正在实施同一
      范围；本任务未获得“开始用 agy 实施”授权，不启动 agy。
- [ ] 先实现功能，再按风险补测试，不采用 TDD。

## 第一组：统一类型、权限、契约与持久化模型

- [ ] 在 `internal/access` 补齐 LAND/RAIL 业务类型，把 lock 操作扩展到六种，保持每个 lock 只
      依赖同类型 read/update；补权限唯一性、依赖和六类型隔离测试。
- [ ] 补齐 HTTP 鉴权对 LAND/RAIL 的 API/Biz 映射与任一订单权限遍历；补按订单 ID 解析及跨
      类型拒绝测试。
- [ ] 修改 OrderLockRecord Ent Schema：增加不可变业务类型，SE 引用改可空，添加正式 CHECK；
      修改 OrderUnlockRequest 增加不可变业务类型。
- [ ] 新增正式 SQL 迁移，按关联订单/锁记录回填现有 SE 历史，校验后设 NOT NULL，解除 SE 引用
      非空并增加同名 CHECK。
- [ ] 扩展 Biz 领域对象和 Proto DTO，锁记录 MBL ID 改可选，追加业务类型字段；运行
      `make -C server api`、`go -C server generate ./...`、`pnpm run generate:web-client` 和
      `pnpm run generate:permission-keys`，禁止手改生成物。
- [ ] 补 service 映射、Proto 注释、Ent migrate metadata 和迁移测试。
- [ ] 运行第一组局部测试、生成幂等和 `git diff --check`；通过后提交：
      `feat: 扩展全业务订单锁契约与权限`。

## 第二组：锁定、解锁、写门禁与 OA 审批

- [ ] 抽取权威业务类型到 lock/update 权限的解析原语；角色资格和候选查询按目标类型接收权限，
      删除 SE 权限字符串与 SE 专属错误文案。
- [ ] 重构 `GetOrderLockState`：六种类型共用生命周期、角色和动作计算，只有 SE 检查活动海运
      提单；返回业务类型和通用阻断原因。
- [ ] 重构 `LockOrder`：先锁 Order、解析类型和资格；公共状态/幂等/记录/审计只实现一次，SE
      条件执行现有 MBL/HBL 快照，其他类型保存空引用；保持版本、代次和锁序语义。
- [ ] 修改三路解锁：请求保存业务类型，普通编辑权限和候选按类型解析；直解、管理员紧急解锁、
      活动申请取代和幂等行为不变。
- [ ] 扩展 OA 命令与网关表单，使用一个 process code 发送业务类型、票号、申请人、代次和可选
      原因；批准生效时校验三份业务类型并按目标类型实时复验审批人。
- [ ] 删除 `ensureOrderBusinessEditable` 的 SE 条件；为非 SE 通用装运单证增改、流转、删除补上
      事务内 Order `FOR UPDATE` 门禁并清除事务外检查写入窗口。
- [ ] 搜索复核所有订单业务子实体写入口；保留费用读和账单生成，确认账单/发票/核销/提成没有
      被误接入业务锁。
- [ ] 实现后补单元测试和真实 PostgreSQL 表驱动测试：六类型锁定/解锁、跨类型权限隔离、非 SE
      空快照、SE 快照回归、订单与费用写阻断、通用装运单证并发、三路解锁、旧 SE 审批生效、
      双锁/双解锁竞争和 DB CHECK。
- [ ] 使用假钉钉 HTTP 服务验证一个模板的字段、OR 候选、明确失败和既有回调流程；不访问真实
      生产钉钉。
- [ ] 运行第二组局部 Go、PostgreSQL、迁移、vet、生成幂等和 `git diff --check`；通过后提交：
      `feat: 统一全业务订单锁与解锁审批`。

## 第三组：可复用前端锁控件与费用只读

- [ ] 在订单领域抽取跨详情/费用页复用的锁状态 hook，使用生成客户端并暴露加载失败状态；不
      新建前端权限真相。
- [ ] 把 `OrderDetailHeader` 的 SE 锁按钮、摘要、弹窗和历史抽屉拆成通用订单锁控件；文案根据
      服务端业务类型显示，只有 SE 提示不可变提单版本。
- [ ] 订单详情对锁状态无条件加载并失败关闭；锁定/解锁成功同时刷新 Order 和锁状态，更新
      version、允许动作和所有子面板写权限。
- [ ] 费用页接入锁状态：加载失败或已锁定时保持列表/汇总/账单创建可用，禁用新增、编辑、
      确认、撤回、作废和标签；显示业务锁提示和重试。
- [ ] 快速费用抽屉接收只读状态，并在已打开时也禁用所有费用与标签写操作。
- [ ] 不新增 SI/AE/AI/LAND/RAIL 页面、菜单或 `ORDER_KIND_CONFIGS` 配置。
- [ ] 实现后补 hook/策略/组件测试，覆盖加载失败、锁定、未锁定、SE 专属提示、非 SE 通用提示、
      锁后费用只读和账单按钮仍可用。
- [ ] 运行前端测试、tsc、Biome、Web 检查和 `git diff --check`；通过后提交：
      `feat: 接入通用订单锁前端交互`。

## 完整验证门

- [ ] `make -C server api`
- [ ] `go -C server generate ./...`
- [ ] `pnpm run generate:web-client`
- [ ] `pnpm run generate:permission-keys`
- [ ] 重跑全部生成命令后无新增差异。
- [ ] `go -C server test ./...`
- [ ] `go -C server vet ./...`
- [ ] 为 `RONCIN_INTEGRATION_DATABASE_SOURCE` 显式提供隔离 PostgreSQL，运行目标集成/并发用例并
      在 `-v` 输出确认 PASS，不能把 SKIP 当通过。
- [ ] 把正式迁移应用到临时空 Schema，核对 revision/checksum、NOT NULL、FK、CHECK 和索引；
      再构造升级前 SE 锁/审批记录验证回填兼容。
- [ ] `pnpm --dir web test`
- [ ] `pnpm --dir web tsc`
- [ ] `pnpm --dir web biome:lint`
- [ ] `pnpm run check`
- [ ] `pnpm run build`
- [ ] `git diff --check`
- [ ] 调用独立 `trellis-check`，修复 P0/P1 并重跑受影响检查。

## 主会话复核重点

- [ ] 任一业务类型都只有一个 Order 锁，锁/解锁版本和代次递增正确。
- [ ] SE 锁定仍原子形成 MBL/HBL 版本；非 SE 不访问海运表且锁记录 SE 引用为空。
- [ ] 只持有一种 lock 权限的角色无法作用于其他类型，bootstrap admin 只保留紧急解锁分支。
- [ ] 现有 SE 活动审批经回填后仍能拒绝、批准或判 stale，候选和外部实例不被重写。
- [ ] 所有业务写先锁 Order 再检查；费用写被阻断，费用读与合法账单生成不被误伤。
- [ ] OA 外部调用仍在事务外，一个 process code、通用字段、OR 候选和 UNKNOWN 不重发语义不变。
- [ ] 页面不硬编码 SE 权限，不推导候选；锁状态失败时所有业务写入口确实关闭。
- [ ] 未触碰无关未提交文件，未提交秘密、真实钉钉调用、缓存或临时数据库产物。

## 风险文件与回滚点

- `server/internal/data/order_lock.go`：锁、幂等、版本和三路解锁集中，任何修改都要用 PostgreSQL
  并发回归证明没有部分写、重复事实或锁序倒置。
- `server/internal/data/ent/schema/order_lock_record.go` 与正式迁移：产生非 SE 记录后不能直接回滚
  到 MBL 必填 Schema；应用回滚前必须先停写并检查数据，禁止删除历史记录。
- `server/internal/access/manifest.go` 与 `server/internal/server/auth.go`：类型集合漂移会导致权限
  已登记但路由拒绝，必须用六类型穷举测试锁定。
- `web/src/pages/orders/fees.tsx` 与快速费用抽屉：业务锁不能误禁用账单生成，也不能只隐藏入口而
  留下已打开弹窗可写。
- 钉钉模板是外部部署前置条件；代码回滚不能把审批申请自动降级为直接解锁。
