# 实施与验证

## 批准依据

用户已在三点明确方案之后回复「好，开始改」，本任务是其最小技术落盘。无新增产品决策，不重复同一授权。

## 步骤

- [x] 研究路由/草稿身份、HBL唯一注册位置与来源TE事实，形成PRD和设计。
- [x] 激活任务，代理A实现来源查询与新建实例/草稿/幂等隔离；代理B实现入口、HBL就近填写与指定航程候选。
- [x] 先实现后补真实表单与路由定向测试，验证权限、重复、模式、身份/迟到、来源失效。
- [x] 检查代理完整复核R1—R6并修复局部问题；主会话复核独立资料与共享航程边界。
- [x] 最终完整前端门禁与git diff --check，真实浏览器只读验证演示票入口与新表单。
- [x] 更新规范与验收记录，提交本组代码。
- [ ] 归档任务并记录开发者会话。

## 文件所有权

A：new.tsx、sea-house-order-context.ts及其测试、来源新建路由/页面测试、templates/types.ts两项来源属性与类型。B：detail.tsx的来源透传、HouseBillIdentityFields、SeaDocumentSection、SeaTransportSection、SeaBasicInfoSection的初始船司选项、useSeaMasterBillCandidate及相关字段测试。共享文件编辑须先协调；其他并行改动不回退不提交。

## 定向验证

pnpm --dir web exec vitest run <本次来源新建/字段/候选/单证相关测试>；修改文件Biome及Ant Design lint；pnpm --dir web tsc。重型文件超过10用例或20秒按功能拆分。最终pnpm run check:web；不涉及后端无需Go全量或build。
