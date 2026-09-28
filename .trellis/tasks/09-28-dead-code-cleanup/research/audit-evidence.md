# 只读审查交接

审查时工作区干净；未编辑产品文件。`pnpm run report:duplicates --format json --output /tmp/roncin-code-audit-duplicates.json` 扫描 414 个 JS/TS 和 414 个 Go 手写文件、7904 个函数、错误 0。两组结果是已记录的跨协议 DTO 映射和跨层订单类型映射，不作为清理目标。

`pnpm --dir web exec tsc --noEmit --noUnusedLocals --noUnusedParameters` 输出 `/tmp/roncin-code-audit-unused-typescript.log`，因未使用声明退出 1：212 行，其中 189 处为未使用 `React` 默认导入，`web/src/pages/orders/split.tsx` 有约 21 项未使用导入。此命令是诊断，不代表项目常规 `tsc` 失败。本任务仅清理 `split.tsx` 与触碰文件，其他导入留待独立批次。

前端候选的具体引用核查见 PRD；`DocumentDetailLayout` 和 `orderFeeColumns` 是高置信孤立模块。`tailwind.config.js` 仅为条件候选，不可凭文本无引用断言构建工具不会隐式读取。

服务端候选见 PRD；`CanTransitionTo` 与 `ComputeFingerprint` 属于“自身测试调用、生产未接入”，必须结合状态事务和指纹契约判定。`LegacyReadOnly`、旧单证接口、开发免密数据和 Admin 旧 query 重定向都有活跃使用或明确测试，不是本任务垃圾代码。
