# 实施计划

1. 只读复核外部报告前端、后端、依赖候选，写入 research/web-cleanup.md、research/server-cleanup.md 和 research/dependencies.md；标明可删、保留、存疑。
2. 最终规划复核后启动任务，按前后端分工实施，避免同时修改同一文件。
3. 清理已确认代码与连带导出/测试桩，运行受影响定向测试及类型检查。
4. pnpm 移除确认的未使用依赖，审阅锁文件；检查并整理 Go 模块差异。
5. 运行 git diff --check、pnpm run check:fast、pnpm run build；构建涉及 Sentry 上传脚本时仅做本地构建验证，不上传外部系统。
6. 独立审查删除清单及差异，分组提交并记录验证结果、保留事项，归档。

用户已确认规划，任务已启动；前后端逐项复核后清理完成。独立审查未发现误删。

## 验证记录
- 前端类型检查及路由/格式化/订单字段10个定向用例通过。
- 后端biz/data/service包测试通过。
- 生产构建通过，仅保留大chunk提示；本地构建明确取消Sentry上传令牌，未执行外部上传。
- pnpm run check:fast 全部通过：前端937项通过、12项跳过，后端测试/vet/漏洞检查通过。默认Go测试不代表专用PostgreSQL集成执行。
