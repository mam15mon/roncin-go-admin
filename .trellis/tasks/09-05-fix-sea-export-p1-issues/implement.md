# 海运出口四项 P1 修复实施计划

## 开始前检查

- [ ] 重新运行 `git status --short`，记录所有用户/其他窗口改动；计划外文件不读取写入意图、不暂存、不提交。
- [ ] 完整读取本任务 `prd.md`、`design.md`、根目录 `AGENTS.md` 和 `implement.jsonl` 引用。
- [ ] 核对相关路径没有更近的 `AGENTS.md`；如有则补充读取。
- [ ] 针对任何与计划文件重叠的来源不明改动停止该组实现并报告。
- [ ] 记录生成前 tracked/untracked 状态，确保生成命令不会吞入其他窗口文件。

## 第一组：ReleasePod 真实海运单证契约与持久化

- [ ] 修改 `server/api/order/v1/order_release_pod.proto`：为响应、新增和更新请求增加显式 `SeaDocumentType + sea_document_id`。
- [ ] 修改 `OrderReleasePod` Ent Schema，增加 Sea MBL/HBL 两个可空真实外键、索引和三引用互斥 CHECK；在 Sea MBL/HBL Schema 增加 `NO ACTION` 反向边。
- [ ] 增加正式 SQL 增量迁移；不改写历史迁移，不自动转换历史 SE 旧引用。
- [ ] 更新 biz 领域对象与引用组合校验，复用已有 `biz.SeaDocumentType`。
- [ ] 更新 service DTO 转换；非法枚举、UUID 或不完整组合返回 `ORDER_RELEASE_POD_DOCUMENT_INVALID`。
- [ ] 重构 `server/internal/data/order_release_pod.go`：所有读取改走 `Data.client(ctx)`；新增/更新在锁住订单后按业务类型验证旧分单或当前真实 MBL/HBL，并持久化到正确外键。
- [ ] 补齐 biz、service、data 单元测试与真实 PostgreSQL 测试：无关联、非 SE 旧分单、SE MBL、SE HBL、共享 MBL 成员、跨订单/跨组织/非当前关系和互斥组合。
- [ ] 运行服务端生成和针对性测试；检查生成物只来源于 Proto/Ent Schema。

验证点：

```bash
go -C server test ./internal/biz ./internal/service
go -C server test ./internal/data -run 'OrderReleasePod|Migration|Schema' -count=1
```

## 第二组：HBL 页面展示与关联删除

- [ ] 在 `RemoveSeaHouseBillRequest` 增加 `remove_related_release_pods`；增加“需确认”和“已回单阻断”稳定错误 reason。
- [ ] 扩展 biz/repo 签名与 service 映射；请求关联删除时额外校验 SE `release_pod.delete` 组织范围权限。
- [ ] 在现有 HBL 删除事务中，按 ID 锁定关联 ReleasePod：`RETURNED` 全量阻断，`PENDING/SIGNED` 未确认则要求确认，明确确认后原子删除并写操作日志。
- [ ] 日志写入失败、HBL 删除失败或关联记录状态并发变化时验证全事务回滚。
- [ ] 前端 ReleasePod 面板按订单类型加载候选：SE 使用真实 MBL/HBL，非 SE 沿用旧分单；实现显式类型/ID 映射、加载失败关闭和正确回显。
- [ ] 海运单证页面按 MBL/HBL 分组展示关联放货记录及状态，并尊重 `release_pod.read` 权限。
- [ ] 合并 HBL 删除确认：列出关联记录；存在已回单时阻断；全为待签收/已签收且有删除权限时允许一次确认并发送单一原子请求；取消不调用 API。
- [ ] 新增、编辑、删除 ReleasePod 或删除 HBL 后同步刷新单证与放货记录。
- [ ] 补 service/data/PostgreSQL 与前端组件测试。

验证点：

```bash
go -C server test ./internal/biz ./internal/service -run 'ReleasePod|SeaHouseBill'
go -C server test ./internal/data -run 'ReleasePod|SeaHouseBill' -count=1
pnpm --dir web test -- --runInBand
```

如果前端测试脚本不接受 `--runInBand`，使用仓库现有 `pnpm --dir web test`，不得改用 npm/npx。

## 第三组：SE 列表入口与箱货删除版本

- [ ] 为 `OrderListTemplate` 单证动作增加默认兼容的可配置文案。
- [ ] SE 列表使用“海运单证”并跳转真实订单详情；非 SE 继续打开旧抽屉。
- [ ] `ContainerDrawer.removeItem` 与 `CargoItemDrawer.removeItem` 发送记录真实 `expectedVersion`。
- [ ] 缺失/零版本时失败关闭并提示刷新，不使用默认版本、不调用 API、不自动重试。
- [ ] 补列表入口、非 SE 回归、两个删除请求和版本缺失测试。

验证点：

```bash
pnpm --dir web test
pnpm --dir web tsc
```

## 第四组：并发拆票错误优先级

- [ ] 保留 Execute 的纯输入结构、幂等和必填版本校验。
- [ ] 删除 Execute 事务前依赖当前可变状态的 Preview 门禁；独立 Preview API 保持不变。
- [ ] 逐项核对仓储事务内已覆盖 Order、Link、Allocation、HBL、货物、箱、费用、候选 MBL/TE、守恒和业务门禁；发现缺项先补锁内重验。
- [ ] 更新 usecase/mock 测试，证明 Execute 不依赖 Preview，静态非法输入/守恒/门禁错误仍保持原 reason。
- [ ] 运行并发子场景和完整真实 PostgreSQL 父测试 `-count=3`；断言每轮一成功一 409，无双成功、孤儿行或重复写入。

验证点：

```bash
go -C server test ./internal/biz -run 'SeaOrderChange|Split' -count=1
go -C server test ./internal/data -run 'TestSeaOrderSplitAndReassignment_PostgresIntegration' -count=3
```

## 第五组：生成、全量检查与差异复核

- [ ] 按仓库生成顺序运行 Proto、Ent/Wire、OpenAPI/Web Client 和错误 reason/常量生成。
- [ ] 在临时 PostgreSQL Schema 执行完整正式迁移链，核对新列、CHECK、外键删除策略、索引和迁移记录。
- [ ] 重跑全部生成命令，对比前后 `git status`/内容指纹，确认生成幂等且没有手改生成物。
- [ ] 运行针对性检查后执行风险匹配的全量检查：

```bash
go -C server test ./...
go -C server vet ./...
pnpm --dir web test
pnpm --dir web tsc
pnpm --dir web biome:lint
pnpm run build
```

- [ ] 独立 `trellis-check` 复核 PRD/设计、跨层字段、权限、事务锁序、迁移、生成物、错误 reason 和前端交互。
- [ ] 对检查发现的问题在同一实现任务内修正并重跑相关门槛。
- [ ] `git diff --check` 通过；暂存前逐文件确认只包含本任务改动。

## 提交分组与回滚点

每组达到可验证状态后由 Codex 主会话提交，不由实施子代理提交：

1. `feat: 支持放货记录关联真实海运单证`：Proto、Schema、迁移、生成物、后端和关联删除/展示的完整跨层契约。
2. `fix: 修正海运单证入口和箱货删除版本`：列表入口与两个前端删除缺陷。
3. `fix: 统一并发拆票版本冲突语义`：Execute/Preview 边界和并发回归。

如果跨层契约未完整通过，不得只提交页面或只提交迁移。回滚优先回退未应用迁移的完整提交；迁移已应用后通过新迁移回滚，不改写历史 SQL。任何回滚都不得覆盖其他窗口的未提交文件。
