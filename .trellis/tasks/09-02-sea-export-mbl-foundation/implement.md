# 海运出口共享主单基础实施计划

## 实施约束

- agy 从本任务路径开始，依次完整读取 `implement.jsonl` 及引用、`prd.md`、`design.md`、本文件和根 `AGENTS.md`。
- 只实现阶段 1；禁止提前实现完整提单内容、箱货分配、拆票、Switch 或共享费用。
- 禁止兼容双写、虚拟 HBL、号码猜测、自动关联、强制合并和静默清理字符。
- agy 不执行 Git commit、push、merge、rebase、reset 或 Trellis finish。

## 实施步骤

1. 数据模型与迁移
   - 建立运输执行、共享 MBL、当前/历史成员实体和索引。
   - 从旧 `OrderConsolidation`/间接归组迁移到新真相，清除 MBL 归组双写。
   - 生成 Ent 代码和正式迁移；执行前核对相关表为空。
2. 契约
   - 增加 MBL 选择、候选摘要、航程冲突和订单当前主单响应。
   - 增加候选查询/匹配 RPC，确保静态路由顺序正确。
   - 生成 API、OpenAPI 和 Web 客户端。
3. biz/data/service
   - biz 定义领域对象、格式校验、候选/关联意图、冲突和仓储接口。
   - data 全部经 `Data.client(ctx)` 和统一共享事务，实施唯一约束、锁与转换。
   - service 只做 DTO/UUID 转换并原样返回领域错误。
4. 订单写入与查询消费者
   - SE 新建必须原子创建或确认关联 MBL。
   - 更新实现单票更正门禁和共享改号阻断。
   - 列表、详情、费用标题等消费者从当前主单摘要读取，不再从第一张 HBL 推断。
5. 前端
   - 配舱信息提供必填 MBL 和主单签发方。
   - MBL 实时转大写，其他非法字符明确报错。
   - 命中候选展示摘要、冲突和明确确认动作。
6. 实现后补测试
   - biz/service/data/HTTP 路由测试。
   - 前端表单、payload、候选确认和冲突测试。
   - 真实 PostgreSQL 唯一约束、并发建单、单票一 ACTIVE 关系和回滚测试。

## 验证

- [ ] `make -C server api`
- [ ] `go -C server generate ./...`
- [ ] `pnpm run generate:web-client`
- [ ] 如修改权限：`pnpm run generate:permission-keys`
- [ ] 上述生成命令重跑无新增差异。
- [ ] `go -C server vet ./...`
- [ ] `go -C server test ./...`
- [ ] 专用 PostgreSQL 测试在显式连接串下为 PASS，不是 SKIP。
- [ ] `pnpm --dir web test`
- [ ] `pnpm --dir web tsc`
- [ ] `pnpm --dir web biome:lint`
- [ ] `git diff --check`

## 主会话检查重点

- 一票一当前 MBL 是否由数据库和领域层共同保证。
- 候选确认最终是否在事务中重验，而不是相信前端查询。
- 关联已有 MBL 是否绝不修改共享内容。
- 锁顺序是否固定，是否存在先 `FOR SHARE` 后升级。
- 当前最终目的地是否仍为订单所有，未被共享卸货港覆盖。
- 旧 MBL/HBL 推断路径是否还有残留消费者。

## 回滚点

阶段 1 一个原子产品提交；失败时同时回滚 Schema、迁移、契约、生成物、后端和页面。没有历史数据前提失效时停止，不执行破坏性迁移。
