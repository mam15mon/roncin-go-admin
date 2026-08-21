# 数据库迁移

生产环境关闭 Ent 自动迁移，通过服务端内置迁移命令顺序执行本目录中的版本化 SQL。
执行器使用 PostgreSQL advisory lock 防止并发部署；每个文件与迁移记录在同一事务
中提交，并校验已执行文件的 SHA-256，禁止修改历史迁移。

在仓库根目录配置 `DATABASE_URL` 后执行：

```powershell
pnpm run migrate:server
```

迁移不使用 `IF EXISTS` 或 `IF NOT EXISTS`。数据库结构与预期不一致时会立即失败，
由部署人员核查差异，禁止静默跳过。
