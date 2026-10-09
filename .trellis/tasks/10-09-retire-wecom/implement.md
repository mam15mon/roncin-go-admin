# 执行与验证计划

## 启动前

- [x] 创建任务并调查前后端、权限、配置、Schema、迁移与相邻测试。
- [x] 落盘 PRD、设计和计划，配置 implement/check 上下文。
- [ ] 用户审阅最终方案并确认企业微信专属绑定字段删除范围。
- [ ] `task.py start` 后按 Trellis 规则派发 implement/check 子代理。

## 实施顺序

1. 创建 `codex/retire-wecom` 分支并确认工作区，保留其他任务记录。
2. 修改 auth/admin/conf Proto 和权限 Manifest；删除后端实现、注入及状态/检索引用。
3. 修改 Ent Schema，新增正式迁移；保留历史迁移/前态夹具；实施后补充迁移保留用户与组织角色记录的验证。
4. 删除前端企业微信入口、列、授权分支、回调、权限判断和样式。
5. 官方命令再生成 API/config/Ent/Wire、客户端、权限键和错误目录，核查差异。
6. 更新相邻测试及规范，定向检查后提交源文件与生成物。
7. 最终门禁、记录证据、提交收尾及归档。

## 验证

- 前端：登录、用户表单/列表、access、组织切换、钉钉回调定向 Vitest；修改文件 Biome 和 TypeScript。
- 后端：`go -C server test -p 32 ./internal/biz ./internal/service ./internal/data ./internal/access ./internal/server ./cmd/server`，确认数据库集成测试实际执行。
- 迁移：隔离 Schema 或临时测试库按正式迁移验证升级与冷启动，不清空开发数据库，不并发 Ent Schema.Create。
- 一致性：企业微信引用仅允许出现在历史迁移、历史前态夹具和下线任务说明，当前运行代码、配置、客户端不得残留。
- 最终：`git diff --check`、生成物一致性及 `pnpm run check:fast`；生产入口/构建配置实质变更时补相应构建。

## 交付限制

实际数据库迁移依赖已确认的绑定信息删除范围，用户与业务记录不删除。记录是否执行开发迁移，不把代码完成表述为生产上线。
