# 提成 CNY 快照与归属日期实施计划

## 实施步骤

- [ ] 修改 `settlement.proto`，增加日期筛选和 CNY 字段。
- [ ] 修改 `FinanceCommission` Ent Schema，增加六个 immutable 字段和日期索引。
- [ ] 生成并审阅 Proto、HTTP/gRPC、OpenAPI、Ent 与迁移差异。
- [ ] 在 biz 层增加 CNY 快照领域类型、日期筛选校验、decimal 公式和用例依赖。
- [ ] 调整 `CommissionRepo`，增加生成上下文读取并让创建接受 CNY 快照。
- [ ] 用 `Transactor.WithinTransaction` 编排汇率解析和提成创建，提交后重读响应。
- [ ] 把本任务触及的提成仓储读取迁移到 `Data.client(ctx)`。
- [ ] 数据层持久化/读取快照，派生动态 CNY 调整和有效金额。
- [ ] 列表增加日期谓词与稳定排序，保留可供导出复用的私有谓词构造函数。
- [ ] Service 完成 DTO/DO 转换，审计详情增加折算依据。
- [ ] 更新 Wire 依赖并重新生成。
- [ ] 运行 `pnpm run generate:web-client`，不手改前端生成客户端。

## 针对性验证

- [ ] biz 测试：CNY、非 CNY、倒数精度、金额公式、非法日期。
- [ ] data/事务测试：生成上下文在事务内首次读取即 FOR UPDATE（核销单行从入口串行化，禁止先 ForShare 再升级）、普通上下文无锁读取、写入回滚、日期筛选和稳定排序。
- [ ] service 测试：单边/双边日期、DTO 字段和错误映射。
- [ ] 回归测试：生成、确认、支付、取消、调整与核销撤销。

## 验证命令

```bash
go -C server test ./internal/biz/... ./internal/data/... ./internal/service/...
go -C server test ./...
go -C server vet ./...
pnpm run generate:web-client
pnpm --dir web tsc
```

## 风险与回滚

- 共享事务是最高风险点；检查所有事务内读取是否经 `Data.client(ctx)`。
- 生成差异异常时修正 Proto/Ent 源文件后重新生成，禁止手补生成物。
- Schema、迁移、契约、实现和生成物保持同一提交，失败时整体回滚。
