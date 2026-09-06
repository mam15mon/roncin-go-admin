# 实施计划：海运出口船公司改用航运公司主数据

## 阶段 1：Schema、迁移与契约

- [x] 更新 Order、SeaTransportExecution、SeaMasterBill、SeaMasterBillVersion Ent Schema，统一
  `shipping_line_id` 字段和 ShippingLine edge，删除 MBL issuer/carrier 重复身份。
- [x] 更新 PartnerRole Ent/Biz/Proto，只保留 customer、supplier、foreign_agent，并 reserved Proto
  carrier 枚举值与名称。
- [x] 新增增量迁移：数据前置阻断、列/约束/索引调整、ShippingLine FK、Partner role CHECK；补充迁移
  PostgreSQL 测试，验证失败原子性及 358 条主数据保留。
- [x] 修改 Order、Sea document、Sea order change Proto，把 SE/MBL 的 carrier/issuer 字段统一为
  `shipping_line_id` / `shipping_line_name`，保留 HBL Partner issuer。
- [x] 按顺序运行服务端 API/Ent 生成及 `pnpm run generate:web-client`、权限/枚举相关生成，审阅生成差异。

## 阶段 2：服务端领域与持久化

- [x] 更新 Order、MBL、版本、锁定快照、拆票和改配领域类型及校验，建立 ShippingLine 三方一致性。
- [x] 在现有事务与固定锁序内校验 ShippingLine 的组织和启用状态；保证内容更新、单成员更正和共享 MBL
  多成员保护不回归。
- [x] 更新 MBL 候选唯一身份和查询，所有船公司名称从 ShippingLine 解析，不再查询 Partner carrier role。
- [x] 更新列表筛选、资源解析、审计差异和不可变版本投影；HBL Partner issuer 路径保持不变。
- [x] 删除 Partner carrier 转换和验证路径，补充 Biz、Service、Data 定向测试。

## 阶段 3：前端选择器与业务入口

- [x] 新增共用 `searchShippingLineOptions`，调用现有服务端分页关键字查询，输出中英文名与 SCAC 标签。
- [x] 新建、详情、拆票、改配和订单列表筛选改用 ShippingLine；更新表单值、payload、候选匹配、摘要和
  当前停用值的只读展示。
- [x] 更新费用页：应付默认值只取订舱代理，无订舱代理则为空；概要显示 ShippingLine 可读名称。
- [x] 从 Partner 页面、结算规则和订单公共常量删除 carrier 角色 UI。
- [x] 补充前端测试，覆盖搜索参数/标签、组织切换隔离、四个写入口、候选确认、费用默认和 HBL 相邻路径。

## 阶段 4：定向验证与独立检查

- [x] 对所有手写 Go/TS/TSX/Proto/SQL 文件运行格式化和 `git diff --check`。
- [x] 运行受影响的 Go Biz/Data/Service 测试、显式 PostgreSQL 迁移与订单集成测试；SKIP 不计为通过。
- [x] 运行受影响的 Vitest、TypeScript 和 Biome/Lint 检查。
- [x] 使用 Trellis 独立检查代理核对真实差异、跨层命名、生成物、锁序、HBL 边界、费用 Partner 边界及
  用户未提交文件隔离；修复 P1/P2 后重跑定向验证。

## 阶段 5：完整门禁、规范与提交

- [x] 执行 `pnpm run check:web`、`pnpm run check:server` 及风险需要的完整测试/构建，记录结果。
- [x] 运行生成命令第二次并确认 tracked/untracked 指纹稳定；确认旧 SE carrier/MBL issuer 仅在历史迁移、
  归档任务或 HBL 合法语境中保留。
- [x] 更新 `.trellis/spec/server/backend/sea-export-document-contract.md`，记录 ShippingLine 权威身份、
  三方不变量、费用边界和迁移前置条件。
- [x] 逐文件暂存并提交本任务代码、测试、迁移、生成物、规范和任务材料；排除用户已有的船公司同步改动。
- [x] 完成 Trellis finish/archive 和开发日志记录。

## 验证记录

- API、Ent 与 Web Client 生成命令重复运行，任务差异指纹保持一致。
- `pnpm run check:web` 通过：79 个测试文件、330 个测试全部通过。
- `pnpm run check:server` 通过：Proto lint、全量 Go 测试、vet 与 govulncheck 均通过。
- `pnpm run build` 通过；真实 PostgreSQL 迁移测试三个场景全部通过。
- 本地开发库迁移通过，实测保留 371 条启用 ShippingLine、建立 4 个 ShippingLine 外键、清除 0 条数据。
- 开发进程曾执行含 `DROP INDEX IF EXISTS` 的同版本迁移；修订为严格 DDL 后按迁移规范精确登记旧校验和，
  未知校验和仍拒绝，旧库与新库最终 Schema 等价。

## 建议验证命令

```bash
go -C server test ./internal/biz ./internal/service ./internal/data
go -C server test ./internal/platform/migration -run 'ShippingLine|Sea|Migration'
go -C server vet ./...
pnpm --dir web test -- --run
pnpm --dir web tsc
pnpm --dir web biome:lint
pnpm run check:web
pnpm run check:server
git diff --check
```

## 高风险文件与回滚点

- `server/internal/data/order_write.go`、`sea_order_change.go`：不得改变固定锁序或绕过事务内复验。
- `server/internal/data/ent/schema/sea_house_bill*.go` 与 `sea_document.proto`：HBL Partner issuer 不得误删。
- `server/migrations/`：只新增迁移，禁止编辑历史文件；阻断错误必须发生在任何 DDL 前。
- `web/src/pages/orders/fees.tsx`：ShippingLine 不得进入 `settlementPartyId`。
- 用户未提交的同步文件：任何自动格式化/生成触碰都必须恢复到用户原差异，但禁止使用破坏性 reset。
