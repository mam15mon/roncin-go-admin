# 实施与验证计划

## 批准依据

用户已明确同意上一轮五项方案并要求实施。PRD/设计为同一方案的技术落盘，没有新增产品决策；主会话复核后激活任务，不重复索取同一授权。

## 执行顺序

- [x] 创建任务，读取领域与前端规范，核对真实候选/写入及共享更新契约。
- [x] 完成只读研究并纳入上下文清单，确认复用现有接口。
- [x] 激活任务；Trellis 实施代理 A 负责候选、继承、差异确认、只读字段、核对提示及相关测试。
- [x] Trellis 实施代理 B 负责共享更新入口文案、影响订单展示及相关测试，不改代理 A 表单文件。
- [x] 先实现，再补测试；执行修改文件 Biome 与相关 Vitest，主会话整合类型影响。
- [x] Trellis 检查代理复核 R1—R6、修复明确局部问题；主会话复核共享资料与单票资料边界。
- [x] 最终运行 `pnpm run check:web --maxWorkers=32` 与 `git diff --check`；完整覆盖前端门禁，1677 项测试通过，12 项既有跳过。
- [x] 浏览器只读复核既有演示票与未保存的新建交互，不额外写业务数据。
- [x] 更新规范并提交本组变化；任务归档与会话记录按收尾流程执行。

## 定向检查

- `pnpm --dir web exec vitest run src/pages/orders/templates/components/sea/SeaMasterBillFields.validation.test.tsx` 及新增继承测试。
- `pnpm --dir web exec vitest run src/pages/orders/templates/sea-create-layout.test.tsx src/pages/orders/templates/sea-template.test.tsx`。
- `pnpm --dir web exec vitest run src/pages/orders/components/drawers/SeaTransportExecutionUpdateModal.test.tsx src/pages/orders/order-kinds/sea-export/SeaExportDetailFeatures.test.tsx`。
- 修改文件 `pnpm --dir web exec biome check <paths>`；类型检查 `pnpm --dir web tsc`。
- 重型单文件超过 20 秒按功能拆分，避免超过 10 个重型渲染用例。

## 风险检查点

- 回填 watcher 不触发查询/渲染死循环，未注册字段监听使用 `preserve: true`。
- 身份变化不残留旧候选/航次/版本/关联分单号；迟到响应不回写新身份。
- 多航程、资料缺失、用户已填值、直单与详情单票更正分别验证。
- 共享修改只用服务端预览范围与现有权限，不用普通保存绕过命令。
- 随时复查 `git status`，保留并行工作，仅提交本任务文件。
