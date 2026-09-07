# 简化海运出口主分单模型实施计划

## 实施原则

- 本任务跨数据库、Proto、Biz/Data、订单锁、改单历史与前端，按可验证阶段提交；每阶段只在前一
  阶段检查通过后继续。
- 不触碰或提交用户现有的 ShippingLine/UNLOCODE 同步、`package.json` 与 `data/` 改动。
- 不手改 Ent、PB、OpenAPI、Web Client、枚举或权限生成物；先改源，再运行项目生成命令。
- 因迁移删除旧结构，实施前再次核对开发库和工作区；发现任何 SE 数据或来源不明的重叠改动时
  停止并报告。

## 阶段 1：Schema、迁移与领域骨架

- [x] 再次执行相关 SE 表行数审计并保存结果，确保迁移空库前提成立。
- [x] 调整 Ent Schema：MBL/TE 解耦、Link 增加 TE、两态结构、当前 HBL 条件唯一、Booking No.、
  TE Version、ModeChangeEvent、SharedContainer/Allocation、确认字段、锁记录 TE 快照。
- [x] 删除 SeaCargoAllocation、SeaHouseBillSwitchEvent 及所有 Schema 边；删除 `REPLACED` 与
  `SWITCH` 来源。
- [x] 新增不可变增量迁移，先做全量相关数据 preflight，再执行目标 DDL；更新 migration revision。
- [x] 运行 Ent 生成和服务端编译，补数据库约束/迁移集成测试。
- [x] 提交 `refactor: 重构海运主分单数据模型`。

## 阶段 2：Proto、创建/查询与单值 HBL

- [x] 修改 `order.proto`、`sea_document.proto`、`sea_order_change.proto`；增加 Booking No. 和两种
  号码筛选，收敛单值 HBL、模式切换与 MBL/TE 关系，reserved 被删除编号和名称。
- [x] 生成 PB、HTTP/gRPC、OpenAPI、Web Client 和枚举。
- [x] 重构 Order/SeaMasterBill/SeaDocument 的 service、biz、data：HOUSE 建单在同事务创建唯一
  HBL，DIRECT 零 HBL，SE 响应通过 Link→TE 投影运输字段。
- [x] 增加按客户业务号、Booking No.、活动 MBL 的精确/模糊查询及同批订单只读摘要。
- [x] 移除 Add/Remove HBL、UNDETERMINED、Mark/Cancel Direct 旧调用链和相关测试。
- [x] 验证创建零写入、当前 HBL 条件唯一、同批号码不联动、组织隔离及分页上限。
- [x] 提交 `refactor: 收敛海运订单单值分单契约`。

## 阶段 3：运输执行版本、共享修改与改配

- [x] 实现 TransportExecution Version 创建/复用与共享航次 Preview/Execute。
- [x] 重构候选关联和改配：同船公司可保留 MBL 换 TE；换船公司必须同时换 MBL/TE；Link 记录
  当前实际执行。
- [x] 在改配/共享修改请求和事件中加入外部确认，附件归属锁内复验。
- [x] 更新 Order Lock：锁定时固定 MBL Version + TE Version + 唯一当前 HBL Version。
- [x] 调整列表、详情、费用页和锁定历史的 SE 运输投影，删除 Order 上 SE 重复字段的写入依赖。
- [x] 覆盖共享成员锁序、成员集合变化、单票甩柜不传播、同 MBL 跨 TE 和锁定历史重现测试。
- [x] 提交 `refactor: 解耦海运主单与实际航次`。

## 阶段 4：改单、作废、模式切换与 Switch 删除

- [x] 给 MBL/HBL amendment、void 和模式切换增加外部确认 DTO、校验、不可变保存和历史输出。
- [x] HOUSE→DIRECT 原子作废当前 HBL；DIRECT→HOUSE 原子建立新当前 HBL；两者保留下游事实。
- [x] 删除 Switch Proto RPC/消息、领域命令、仓储实现、事件表引用、页面按钮与测试；保留 reserved。
- [x] 调整下游影响规则：ETD/离港/财务/放货只提示，不自动改写且不一概阻断；真正结构冲突仍
  返回稳定 400/409。
- [x] 让携带外部确认的专用变更命令显式绕过普通业务锁门禁，同时保持订单锁定且不改旧快照；
  普通订单更新继续被锁阻断。
- [x] 验证 Preview 输入变化失效、Execute 锁内重算、确认必填、幂等/并发、历史版本不漂移。
- [x] 提交 `refactor: 收敛海运单证变更流程`。

## 阶段 5：普通箱与共享箱例外

- [x] 常态订单沿用 OrderContainer 直接归属，移除旧 SeaCargoAllocation 服务、页面和权限入口。
- [x] 实现 SharedContainer/Allocation 的 Proto、service、biz、data 和跨订单守恒/版本/锁序。
- [ ] 新增仅在显式“共享箱/客户拼货”下出现的工作台；按 TE 选择 HOUSE 订单并展示逐票件重尺。
- [ ] 重构拆票 Preview/Execute 输入和算法为“结果订单 + HBL + 货物/箱/草稿费用显式分配”；共享
  物理箱只移动 allocation，不复制箱号。
- [ ] 覆盖超分、确认守恒、跨组织/跨 TE/DIRECT 拒绝、并发版本、审计失败回滚和拆票守恒测试。
- [ ] 提交 `refactor: 简化海运箱货分配与拆票`。

## 阶段 6：前端收敛与全链验证

- [x] 创建页增加 HOUSE/DIRECT 必选、唯一 HBL 和 Booking No.；详情页改为单一分单区块。
- [x] 订单列表复合号码筛选增加客户业务号、Booking No.，同批订单区分三种来源且只读。
- [x] 重构改配/改单/作废/模式切换弹窗，强制外部确认并展示影响；删除 Switch 和旧分配入口。
- [ ] Shared Container 工作台只对明确共享箱开放；普通订单维持直接箱号交互。
- [ ] 补前端交互、payload、失败关闭、缓存失效与生成类型测试。
- [ ] 运行全量生成幂等、质量门禁和构建，执行 `git diff --check`，核对无用户文件混入。
- [ ] 提交 `refactor: 简化海运出口单证交互`。

## 验证命令

按阶段先小后大执行：

```bash
go -C server generate
make -C server api
pnpm run generate:web-client
pnpm run generate:permission-keys
go -C server test ./internal/biz/... ./internal/service/... ./internal/data/...
go -C server vet ./...
pnpm --dir web test
pnpm --dir web tsc
pnpm --dir web lint
pnpm run check:server
pnpm run check:web
pnpm run check
pnpm run build
git diff --check
```

数据库用设置了 `RONCIN_INTEGRATION_DATABASE_SOURCE` 的独立 PostgreSQL Schema 验证：

- 空基线迁移成功；相关任一表非空时 DDL 前原子失败；
- MBL/TE/Link 三方组织和 ShippingLine 不变量；
- 一订单单活动 Link、一 HOUSE 订单单当前 HBL；
- HOUSE/DIRECT 创建与切换失败零写入；
- 同 MBL 不同 TE、共享 TE 单票分离与共享更新；
- 外部确认附件 `NO ACTION`、历史版本和下游引用不漂移；
- SharedContainer 跨订单守恒和并发冲突；
- 全迁移链与连续生成幂等。

## 高风险点与停止条件

- `SeaMasterBillVersion`、`OrderLockRecord` 和 ReleasePod 是历史引用核心；任何自动级联删除或
  回读当前实体的实现都必须停止并修正。
- MBL/TE 共享写入涉及多订单锁；锁序不一致、成员集合未在锁后重验或事务内直连 `d.db` 均不得
  合入。
- SharedContainer 的件重尺必须使用 decimal/numeric；不得延续 `OrderContainer` float 误差到
  守恒判断。
- Proto 删除必须 reserved；生成结果异常时修源文件，不手工修生成物。
- 若实施时发现真实 SE 数据不再为空，立即停止迁移和删除结构工作，回到规划重新决定迁移策略。
- 若用户工作区文件与本任务目标文件发生重叠，停止对应阶段并报告，不覆盖或重置。
