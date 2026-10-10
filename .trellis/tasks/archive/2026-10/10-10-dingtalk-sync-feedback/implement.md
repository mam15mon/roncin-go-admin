# 执行与验证

## 实施前

- [x] 创建任务并定位失败展示及邮箱校验。
- [x] 完成 PRD、设计、计划及上下文清单。
- [x] 用户批准最终方案，包括缺邮箱时清空旧邮箱。
- [x] `task.py start`，实施已交由 Trellis implement，完成后交由 Trellis check 复核。

## 实施顺序

1. 创建独立 codex 分支（保留尚未合并企业微信下线提交），检查并保护已有改动。
2. 后端调整邮箱可选语义、各失败路径名称 DO/PO/DTO，修改 Proto 和 Ent JSON类型，再生成官方绑定/Ent/Wire/客户端；禁止手改生成物。
3. 前端失败列名称主展示与 ID 次要展示，邮箱正常缺失不增加额外提示；待分配页签隐藏所属组织列，验证其他页签及权限维持。
4. 实现后补充定向验证：部门列表和独立人员详情 email 缺键/null/空/非空；保留手机/头像错误；名称贯穿网关/biz/持久化/service/API/UI；新名和缺名称展示。
5. 独立 PostgreSQL 验证缺邮箱清空旧值、正常更新其他资料、保留人员身份/权限/启停；验证真实错误名称存储与分页转换。
6. 更新钉钉规范，运行 full-scope check、生成一致性、`git diff --check`、`pnpm run check:fast`；开发服务重新同步与页面验收。
7. 提交完整代码及生成物，记录验收/归档；不触碰其他钉钉任务状态。

## 验证命令

- 前端定向 `pnpm --dir web exec vitest run src/pages/admin/components/users/DingTalkDirectorySyncPanel.test.tsx`，修改文件 Biome、`pnpm --dir web tsc`。
- Go 定向 biz/service/data/server 的钉钉相关用例，独立 roncin_go_admin_integration 经进程内环境注入，不输出凭据或人员资料。
- API/Ent生成幂等、前端客户端与权限/枚举一致性按变化适用。
- 全栈门禁 `pnpm run check:fast`，路由或构建配置未变化时不固定运行生产构建。

## 实施结果

- [x] 前后端实施及源契约/生成物同步。
- [x] 前端定向与后端网关/biz/service验证。
- [x] 隔离 PostgreSQL 实际执行验证。
- [x] 规范更新、独立复核、全栈门禁、生成幂等与差异检查。
- [x] 真实开发同步及页面验收。
- 验收细节见 `validation.md`；完成本组提交后按 finish-work 归档任务并记录日志。

## 回退

代码可按提交回退；正式同步中的旧邮箱清空不能靠代码回退恢复，必须在批准规则后运行。本次不删除历史同步任务、用户或组织角色数据。
