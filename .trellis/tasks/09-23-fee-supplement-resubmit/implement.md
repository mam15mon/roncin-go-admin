# 执行计划：撤回并预填重提

1. 核对 `FeeSupplementSection` 的能力投影、撤回成功/冲突处理和 `FeeSupplementModal` 的初始化生命周期，确认原字段到表单字段的完整映射。
2. 在现有表单增加可选预填入口及失效选项/手动汇率提示；保持普通新建路径不变。
3. 在列表行增加“撤回并重提”动作：确认、调用既有撤回接口、成功后预填并打开、失败只刷新；新提交使用新幂等键。
4. 实现后补针对性测试：新建默认值、重提字段、失效选项/权限、撤回竞争、订单切换迟到响应、关闭不提交。
5. 运行 `pnpm --dir web exec vitest run src/pages/orders/components/fees/FeeSupplementSection.test.tsx src/pages/orders/components/fees/FeeSupplementModal.test.tsx`（若第二文件不存在则替换为实际新增文件）、修改文件 Biome、`pnpm --dir web tsc`、`git diff --check`；与兄弟任务合并后运行全栈门禁。

风险点：不要让表单打开时的默认值覆盖预填值；不要在撤回失败后显示新建表单；不要以旧申请幂等键创建新申请。若撤回功能回归，先在 `FeeSupplementSection` 恢复独立的旧动作行为，再重试本任务改动。
