# 实施计划

## 启动前门禁
- [x] 用户审批最终删除范围与数据迁移说明，并要求review、合并、清理分支及push。
- [x] 研究报告与PRD/design收敛，implement/check上下文清单均非空。
- [x] 已在codex/remove-order-milestones启动任务，产品修改前工作树干净。

## 分工与顺序
1. 后端源实现：删除里程碑proto/服务/biz/data、注册与注入；删除Manifest操作与依赖测试，改主单门禁与相邻后端用例，清理sync-dev种子。
2. Schema与迁移：移除Ent模型/关系，生成Ent，新增严格删表迁移和隔离前态升级/冷启动验证；验证现有权限同步能精确移除已删能力。
3. 前端：删除抽屉与列表接线、模板callback/菜单、审计专属展示；同步现行文档和角色夹具，补列表入口不再出现且操作进度仍保留的验证。
4. 串行生成：后端源与Schema修改完成后协调 `make -C server api`、`go -C server generate ./...`、`pnpm run generate:web-client`、`pnpm run generate:permission-keys`；不要多agent并发生成同一目录。
5. 核查孤立生成文件：仅清除已撤销源对应的专属生成物；二次生成差异必须稳定。

## 验证
- 开发过程：相关Go层/access/注册/主单门禁/sync-dev定向测试；前端列表/模板/角色授权/审计展示定向测试及修改文件Biome；`git diff --check`。
- 迁移：隔离库/随机Schema应用完整正式SQL链；构造有里程碑和关联授权的前态后升级，断言表与目标权限消失、非目标订单/授权/审计保持；禁止指向日常开发库，禁止Schema.Create。
- 生成：接口/OpenAPI/前端客户端与权限键均按源生成，二次生成幂等；当前源和生成物无活跃里程碑引用。
- 最终：完整 `pnpm run check:fast`，记录真实数据库用例PASS/SKIP区别；新增迁移的隔离验收不能以默认跳过当通过。
- 真实页面：更多菜单不含履约里程碑，订单操作进度/费用审核/附件/人员/箱货/POD入口保留。

## 收尾
审查范围与隐含调用关系，更新当前能力规范，按可验证组提交Conventional Commits。任务记录验证后归档，不改写历史迁移与其他会话任务。
