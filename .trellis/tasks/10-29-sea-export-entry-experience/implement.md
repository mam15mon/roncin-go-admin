# 执行计划

## 开始条件

- 用户审阅本任务规划后明确同意实施；再执行 task.py start。
- 读取相关技能和规范，检查 git status，保留其他工作。
- 复核 R5 实际禁用行为及新建成功导航守卫；只修真实问题。

## 实施顺序

1. 人员联动：默认值尊重手动编辑；加入就地加载、失败重试、缺配置提示与权限适配入口。
2. 草稿：覆盖程序回填与清空，保持身份隔离、隐藏字段和日期恢复；验证切客户后的完整快照。
3. 新建提交：冻结载荷与键、明确失败/结果待确认区分、刷新恢复、禁止重复提交及成功清理导航。
4. 新建吸底操作栏；ETA 保留并报错；浏览器验证并修复具体键盘和遮挡问题。
5. 实现后补正式回归测试（禁止 TDD）；按各组运行定向测试与静态检查，再提交。
6. 全任务完成执行前端门禁、浏览器验收，记录结果与无法执行的项目，更新受影响规范并提交、归档。

## 验证

- 定向：pnpm --dir web exec vitest run src/pages/orders/new-create-idempotency.test.tsx src/pages/orders/use-sea-order-save.test.tsx src/pages/orders/templates/sea-create-error-reveal.test.tsx src/pages/orders/templates/sea-document-validation.test.tsx src/pages/orders/order-kinds/sea-export/form-adapter.test.ts
- 同步运行新增复现回归，以及客户人员、SeaScheduleDateFields、OrderFormTemplate、formDraft 和受影响空运测试；路径必须相对 web。
- 增量检查：修改文件 Biome、pnpm --dir web tsc、git diff --check。
- 最终门禁：pnpm run check:web；本期无构建/依赖修改，不强制构建或后端全量测试。
- 浏览器：真实表单 + 可控延迟/失败接口，覆盖 AC1–AC8；业务写入仅使用明确的测试环境与测试数据，不依赖生产数据。
- 重型测试单文件超过 20 秒或超过 10 个重型渲染用例时拆分。

## 协作和提交

本任务按同一录单体验闭环顺序推进；若使用实现/检查子代理，主会话负责协调、审批边界、规范更新及 Git 提交，不并行修改同一文件。
