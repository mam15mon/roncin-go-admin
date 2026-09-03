# 海运出口箱货分配实施计划

## 实施约束

- Agy 从本任务目录开始，依次完整读取 `implement.jsonl` 及引用、`prd.md`、
  `design.md`、本文件、父任务文档、阶段 1/2 归档文档和根 `AGENTS.md`。
- 本阶段涉及 Schema、共享事务、固定锁序和跨入口并发门禁，按仓库约定使用
  `gemini-3.8-flash-high`。
- 只实现阶段 3；禁止提前实现拆票、改配、Switch B/L、签发版本、外部状态和费用
  重分摊。
- 禁止浮点容差、自动舍入、虚假箱号、虚拟 HBL、HBL 内容静默改写、旧
  `shipping_document_id` 双写和任何兼容兜底。
- Agy 不执行 Git commit、push、merge、rebase、reset 或 Trellis finish。

## 实施步骤

1. Schema 与迁移
   - 新建 `SeaCargoAllocation`，增加组织/Order/Link/Cargo/HBL/Container 外键、
     十进制数量、CHECK 和有箱/无箱条件唯一索引。
   - 为 Link 增加分配状态、聚合版本和确认人/时间。
   - 为 CargoItem/Container 增加组织和版本，为 Container 增加件数；重量/体积列
     使用规定 numeric 精度。
   - 删除 Container 的旧 `shipping_document_id` 字段、边和索引。
   - 生成 Ent 和正式迁移；迁移以“相关表为空”为前提，非空立即停止。
2. 契约与生成
   - 新增箱货分配聚合、十进制数量、进度汇总、允许动作和六条读写命令契约。
   - 货物/实际箱接入 version，实际箱接入件数并移除旧单值单证字段。
   - 生成 API、OpenAPI、Web 客户端和 Wire/Ent 代码；不得手改生成物。
3. biz
   - 使用 `decimal.Decimal` 实现输入尺度校验、三维汇总、草稿超分校验和确认守恒。
   - 实现确定性错误选择、中文精确消息和结构化 metadata。
   - 定义聚合、行、进度、仓储接口、状态机和审计事件。
   - 货物/实际箱输入拒绝 NaN/Inf、超尺度和非法件数；不做静默纠正。
4. data/service
   - 实现聚合读取与批量草稿替换，按固定顺序锁表、锁后重验当前 Link，并比对版本。
   - 接入货物、实际箱、HBL、订单装载类型的 CONFIRMED 门禁和草稿超分门禁。
   - 实现确认、撤回、逐张 HBL 汇总填入和 DIRECT 的 MBL 货物汇总填入。
   - 所有 Ent 客户端经 `Data.client(ctx)`；写入和审计同事务，提交后普通上下文重读。
   - service 只做 UUID/DTO/十进制字符串转换，错误原样外传。
5. 页面
   - 新增海运 HOUSE “箱货分配”宽抽屉及列表/详情入口，默认不折叠。
   - 实现全量分配表、FCL 实际箱选择和 LCL/散杂无箱行为。
   - 使用十进制字符串实时汇总货物、HBL、实际箱；蓝/绿/红显示进行中/完成/超出。
   - 超分禁用保存；未分完可保存草稿但禁用确认，并列出具体未完成项。
   - 将服务端 metadata 定位到对应行；逐张提供 HBL 汇总填入按钮。
   - DIRECT 的 MBL 卡增加明确货物汇总填入动作；移除实际箱旧 HBL 下拉。
6. 实现后补测试
   - biz：正数/尺度/NaN/Inf、精确十进制、超分、未分完、FCL/LCL、确定性错误文案。
   - data：跨组织/Order/旧 Link 引用、条件唯一键、批量原子性、版本冲突、状态门禁、
     审计失败回滚、集合变化使聚合版本失效、显式提单填入。
   - service/API：DTO、路由、十进制字符串和 error metadata 透传。
   - 前端：每次输入实时剩余、三种颜色、超分禁存、未完可存不可确认、错误定位、
     确认不改 HBL、逐张显式填入、DIRECT 明确填入。
   - PostgreSQL：一箱多 HBL、一 HBL 多箱、并发保存/货物更新/HBL 删除无死锁、
     stale version 409、全部失败路径无部分写入。

## 验证清单

- [ ] `make -C server api`
- [ ] `go -C server generate ./...`
- [ ] `pnpm run generate:web-client`
- [ ] 如未修改权限 Manifest，记录无需运行 permission-keys；如修改则生成并校验。
- [ ] 所有生成命令重跑无新增差异。
- [ ] `go -C server vet ./...`
- [ ] `go -C server test ./...`
- [ ] 专用 PostgreSQL 集成测试连接隔离 Schema 并 PASS，不得 SKIP。
- [ ] `pnpm --dir web test`
- [ ] `pnpm --dir web tsc`
- [ ] `pnpm --dir web biome:lint`
- [ ] `git diff --check`
- [ ] 正式迁移在已获授权的当前开发库应用成功，并核对列类型、CHECK、FK 和索引。

## 主会话检查重点

- 分配是否绑定历史 Link，而不是只绑定 Order/当前 MBL。
- HBL 是否仍是真实多实体，一箱多 HBL 和一 HBL 多箱是否都能表达。
- 货物是否唯一来源；HBL 显示值是否绝未成为守恒真相或被确认静默覆盖。
- DRAFT 是否允许不完整但拒绝超分，CONFIRMED 是否逐货物/逐箱完整守恒。
- FCL 草稿是否允许暂缺箱但确认必须真实落箱，LCL/散杂是否绝不制造虚假箱。
- 十进制是否全链路精确，是否存在 epsilon、float 求和、自动舍入或 NaN/Inf 漏洞。
- 错误是否带稳定 reason、具体中文消息和可定位 metadata。
- UI 是否每次输入立即显示数字和颜色，超分禁存、未完成禁确认。
- 货物/实际箱/HBL 集合变化是否使旧分配版本失效；确认后的所有旁路是否阻断。
- 固定锁序、锁后关系重验、版本冲突和审计失败是否完整回滚。
- 旧 `shipping_document_id` 是否从 Schema、API、后端、页面和测试彻底移除。

## 回滚点

阶段 3 使用一个产品提交；失败时同时回滚 Schema、迁移、契约、生成物、后端、
页面和测试。迁移非空前提失效时停止，不执行破坏性迁移或兼容双写。
