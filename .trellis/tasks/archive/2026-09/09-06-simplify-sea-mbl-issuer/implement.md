# 实施计划：简化海运出口主单签发方录入

## 阶段 1：只读数据审计与后端订单不变量

- [x] 对开发数据库执行只读审计，记录 SE Order carrier、活动 TE carrier、MBL issuer 的空值、
  不一致及潜在唯一冲突；四项计数均为 0，未修改数据，详见 `research/data-audit.md`。
- [x] 在 `internal/biz` 中要求 SE 船公司非空，并以 carrier 规范化订单 MBL issuer；非 SE 不受影响。
- [x] 在 `internal/data/order_write.go` 的现有事务与锁序内，确保新建/更新后的 Order、TE、MBL
  carrier/issuer 一致，保留候选确认、单成员更正原因和共享 MBL 修改限制。
- [x] 补充 Biz 与目标 PostgreSQL 测试，覆盖缺 carrier、issuer 不一致输入、创建、单成员修改、
  共享 MBL 阻断及候选确认。

## 阶段 2：订单新建与详情 UI

- [x] 海运出口“船公司”增加必填校验；不改字段名称。
- [x] 删除 `SeaTransportSection` 的 MBL issuer 选择器与 `searchIssuers` 透传，MBL 号保持原位置和必填。
- [x] 创建与更新 payload 从 carrier 派生现有 `issuerPartnerId`；候选匹配使用 carrier。
- [x] carrier 或 MBL 号变化触发单成员更正原因；多成员共享 MBL 时同时禁用 carrier 和 MBL 号。
- [x] 删除 MBL 摘要中的重复签发方展示，但保留所有 HBL 签发主体 UI。
- [x] 补充海运模板、创建 payload、详情 payload和候选匹配的定向测试。

## 阶段 3：拆票与整票改配

- [x] 拆票 NEW/CANDIDATE 目标删除 MBL issuer 输入，船公司默认沿用来源并允许修改，提交时必填。
- [x] 整票改配删除“发单人 / 船代”，将 carrier 作为唯一船公司输入并设为必填、默认当前值。
- [x] Biz/Service/Data 对 NEW 目标以 carrier 维护 MBL issuer，对 CANDIDATE 使用候选权威身份并
  锁内验证 MBL issuer 与 TE carrier 一致。
- [x] 补充拆票、改配的前端/Biz/PostgreSQL 测试，断言目标 Order、TE、MBL 三方一致。

## 阶段 4：定向验证与独立检查

- [x] 对修改文件运行定向 Biome/gofmt 和 `git diff --check`。
- [x] 运行受影响的前端 Vitest、Go Biz/Data 测试和显式 PostgreSQL 目标用例；显式 PostgreSQL
  用例实际执行，未把 SKIP 记为 PASS。
- [x] 由独立检查代理核对真实差异、跨层数据流、固定锁序、共享 MBL 与 HBL 边界；检查发现并
  修复锁单相邻路径的 MBL 内容独立更新回归，增加对应集成测试。

## 阶段 5：最终验收与提交

- [x] 根据风险执行完整 Web 与 Server 门禁；Web lint/tsc 通过，完整 Vitest 79 文件、328 条测试
  通过；Server 的 Proto lint、全量 Go 测试、vet 与 govulncheck 通过。本任务不涉及构建配置，
  未运行 build。
- [x] 确认没有 Proto、OpenAPI、Web Client、Schema 或迁移差异；如实施中发现必须改契约，返回规划
  阶段重新说明范围，不直接扩大。
- [x] 更新海运共享 MBL 规范，记录“用户维护 carrier、系统维护 MBL issuer”的可执行契约。
- [ ] 使用准确的 Conventional Commit 提交代码、测试、规范与任务材料。
- [ ] 完成 Trellis finish/archive 和开发日志记录。

## 风险与回滚点

- 最大风险是误删 HBL issuer；任何全局替换都禁止，检查必须按 MBL/HBL 类型逐入口核对。
- 共享 MBL carrier 是共享事实，UI 禁用不能代替服务端锁内保护。
- 既有不一致数据可能导致普通更新或候选匹配失败；本任务不通过兼容分支隐藏，先审计再决定。
- 拆票和改配沿用现有 DTO 是有意的最小范围；内部字段仍存在不等于业务继续手工维护它。
