# 无用代码清理最终验证

## 删除与保留

- 删除 `DocumentDetailLayout` 及公共出口、旧 `orderFeeColumns` 与唯一的测试 mock；清理拆票页 `split.tsx` 的无用导入。真实费用表格列定义与拆票请求、指纹算法保持原样。引用证据与前端命令记录见 [web-verification.md](web-verification.md)。
- 删除后端无消费者的经营办理权限分类链、`RequireBaselineWrite`、只由自测调用的 `CanTransitionTo` 与 `ComputeFingerprint`。费用补录三个真实写入路径的事务内加锁、版本及 `PENDING` 校验保留；拆票服务端请求指纹比对保留。引用证据及真实 PostgreSQL 用例见 [server-verification.md](server-verification.md)。
- 删除 `web/tailwind.config.js`。前端配置删除前后均可生产构建，三份产出 CSS 的文件名、SHA-256 和逐字节内容相同。
- 保留有运行时消费者或现行契约的 `LegacyReadOnly`、旧单证接口、开发免密数据、Admin 旧 query 重定向、现行费用列、权限 Manifest、生成物与历史迁移；未卸载依赖，也未合并两组合理重复映射。

## 验证结果

| 检查 | 结果 |
| --- | --- |
| 前端费用、拆票、费用列定向测试 | 3 文件、22 项 PASS；0 FAIL。 |
| 前端 Biome、`pnpm --dir web tsc`、架构检查 | PASS；架构扫描测试 18 项通过，413 个产品文件无违规。 |
| 前端生产构建及 CSS 删除前后比对 | PASS；三份 CSS 逐字节一致。 |
| 服务端 access/biz/data 定向测试、相关包 `go vet` | PASS。 |
| 一次性 PostgreSQL 库的费用补录决策及拆票用例 | 2 个顶层、23 个子项 PASS；0 FAIL、0 SKIP。测试库和角色已清理。 |
| `pnpm run check:fast` | PASS；Web 184 个文件通过、1 个文件跳过，1089 项通过、12 项跳过；Server 的包测试、vet、漏洞检查均通过。 |
| `git diff --check` | PASS。 |

Web 全量 Vitest 的 `localhost:3000` `ECONNREFUSED` stderr 不影响退出结果；7 条 Biome info 位于未修改的提成测试文件。Go/前端门禁均以实际零退出判定通过。定向服务端真实库测试已经启用集成用例，而 `check:fast` 本身按默认开关运行。此次没有人工浏览器操作，清理范围没有修改用户可见交互。

`noUnusedLocals/noUnusedParameters` 临时诊断从 212 项降为 189 项，其中未使用 `React` 默认导入从 189 项降为 187 项；`split.tsx` 从 22 项降为 0 项。该临时诊断不属于正式门禁，本任务有意保留全仓其他 189 项供独立批次处理。

工作提交：`b34fdb26`（服务端）、`8d30142f`（前端）。未更改 Proto、Ent Schema、正式迁移、权限码或生成文件，未清空或重建开发数据库。
