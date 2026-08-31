# 提成导出实施计划

## 实施步骤

- [ ] 在 `settlement.proto` 增加 Export RPC、HTTP 路径、权限规则和扁平导出 DTO。
- [ ] 在 biz 层增加导出上限常量、领域错误、用例编排和成功审计详情。
- [ ] 扩展 CommissionRepo 的 Count/Export batch 能力，复用阶段 1 筛选谓词与排序。
- [ ] Service 完成请求和响应转换，不包含 CSV 逻辑。
- [ ] 在权限 Manifest 登记 export 权限及 read 依赖。
- [ ] 生成服务端访问规则、Proto/OpenAPI、前端客户端和前端权限键。
- [ ] 补充上限、批次、筛选、排序、字段和审计测试。

## 针对性验证

- [ ] 无权限请求被中间件拒绝。
- [ ] 10001 行只执行 Count，不查询数据、不写成功审计。
- [ ] 10000 行按不超过 200 的批次完整返回。
- [ ] 固定数据集下列表与导出的筛选、顺序和动态金额一致。
- [ ] 审计写入失败时响应失败且不返回导出数据。
- [ ] 失败请求没有额外业务审计要求。

## 验证命令

```bash
go -C server test ./internal/biz/... ./internal/data/... ./internal/service/...
go -C server test ./...
go -C server vet ./...
pnpm run generate:web-client
pnpm run generate:permission-keys
pnpm --dir web tsc
```

## 风险与回滚

- 不允许为了复用列表接口而循环调用 Service；批处理编排属于 biz/data。
- 不允许超过上限后截断返回或先加载全量再检查长度。
- RPC、权限、生成物和实现保持同一提交，失败时整体回滚。

