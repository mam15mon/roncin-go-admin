# 海运出口主分单内容实施计划

## 实施约束

- Agy 从本任务目录开始，依次完整读取 `implement.jsonl` 及引用、`prd.md`、
  `design.md`、本文件、父任务文档、阶段 1 归档文档和根 `AGENTS.md`。
- 只实现阶段 2，不提前实现箱货定量分配、拆票/改配、不可变版本、外部状态、
  Switch B/L 或费用分摊。
- 禁止虚拟 HBL、无 HBL 自动 DIRECT、DIRECT 直接添加 HBL、隐式删除、旧模型双写、
  本公司 Partner 自动创建和号码字符静默清理。
- Agy 不执行 Git commit、push、merge、rebase、reset 或 Trellis finish。

## 实施步骤

1. Schema 与迁移
   - 为活动 MBL 成员增加三态单证结构。
   - 为共享 MBL 增加当前可编辑提单内容字段。
   - 新建 `SeaHouseBill`，实现双类型签发主体、原始/规范化号码、版本、内容、外键、
     CHECK 和两个条件唯一索引。
   - 生成 Ent 代码和正式迁移；迁移前检查 SE 业务表为空。
2. 契约与生成
   - 增加结构枚举、通用内容、HBL、聚合读写及结构命令契约。
   - 订单创建/更新接入明确的 `sea_document`，SE 拒绝旧 HBL 写入口。
   - 生成 API、OpenAPI、Web 客户端和枚举；不得手改生成物。
3. biz
   - 实现 HBL 原号保留与唯一规范化纯函数。
   - 实现结构不变量、DIRECT 取消路径、HBL 主体解析输入和内容字段校验。
   - 定义聚合读取/写入仓储接口、领域错误、版本冲突和审计事件。
4. data/service
   - 通过当前活动成员定位 MBL；固定锁顺序并在首次写读使用 `FOR UPDATE`。
   - 实现本公司组织祖先解析、委托单位/其他 Partner 校验、条件唯一冲突映射。
   - 订单事务创建初始结构/内容/HBL；订单更新和独立命令复用同一规则。
   - service 只做 DTO/UUID 转换，不复制业务规则。
5. 页面
   - 从海运“配舱信息”移除旧的平铺 HBL 写入口，保留 MBL 身份和航程。
   - 新增默认展开“提单信息”区块、三态摘要/动作、MBL/HBL Tabs 和完整内容表单。
   - 实现签发主体三选一、其他 Partner 服务端搜索、一次性复制上一张内容。
   - DIRECT 隐藏添加入口；删除最后一张 HBL 要求明确确认回到未确定。
   - 列表/详情展示简洁结构和 HBL 数量摘要，不把长内容塞入列表。
6. 实现后补测试
   - biz：状态转换矩阵、DIRECT 阻断、原号无损、Unicode/大小写规范化、内容边界。
   - data：主体互斥/外键/条件唯一索引、不同主体同号、共享内容、固定锁顺序、版本
     冲突、审计失败回滚。
   - service/API：DTO、必填/UUID、路由和错误语义。
   - 前端：默认展开、状态动作、无默认签发主体、DIRECT 无添加入口、Tabs 独立、
     复制不联动、删除最后一张确认。
   - 真实 PostgreSQL：两个操作票读取同一 MBL 内容、并发版本冲突、HBL 唯一范围、
     创建/更新失败完整回滚。

## 验证

- [ ] `make -C server api`
- [ ] `go -C server generate ./...`
- [ ] `pnpm run generate:web-client`
- [ ] 如修改权限：`pnpm run generate:permission-keys`
- [ ] 所有生成命令重跑无新增差异。
- [ ] `go -C server vet ./...`
- [ ] `go -C server test ./...`
- [ ] 专用 PostgreSQL 集成测试显式连接开发库运行并 PASS，不是 SKIP。
- [ ] `pnpm --dir web test`
- [ ] `pnpm --dir web tsc`
- [ ] `pnpm --dir web biome:lint`
- [ ] `git diff --check`
- [ ] 迁移在当前开发库应用成功，并核对 CHECK、FK 和条件唯一索引。

## 主会话检查重点

- 单证结构是否在成员关系而非共享 MBL 上，DIRECT 是否绝无 HBL。
- DIRECT → HOUSE 是否被拒绝，是否必须先取消直单标记。
- 本公司是否使用真实 company/headquarters ID，是否存在隐式 Partner 或 seed。
- HBL 原号是否无损，规范化是否保留内部标点/空白/前导零。
- 同一真实主体规范化同号是否冲突、不同主体同号是否允许。
- 共享 MBL 内容是否只有一份，关联已有 MBL 是否绝不夹带覆盖。
- 事务是否固定锁序，版本冲突和审计失败是否完整回滚。
- SE 是否还存在旧 `shipping_documents` 双写，空运路径是否保持可用。
- 页面是否默认展开，复制后修改是否不联动。

## 回滚点

阶段 2 使用一个产品提交；失败时同时回滚 Schema、迁移、契约、生成物、后端、
页面和测试。迁移前提（无 SE 历史数据）失效时停止，不执行破坏性迁移。
