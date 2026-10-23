# 审查与验证记录

## 独立审查
由独立 trellis-check 代理完成两轮 review：进程展示改动，以及用户追加的操作人员姓名展示。无阻塞发现，无需修复产品代码。

已检查行模型全部生产者/消费者、列表筛选独立契约、列设置稳定 dataIndex、海空运操作节点差异、异常独立展示及真实单元格断言；追加操作人员列只显示姓名，未分配仍为 `-`。

## 定向验证
- 初始回归：2 文件、18 用例通过。
- 追加操作人员调整后，组件回归：1 文件、4 用例通过。
- 修改文件 Biome、antd lint、最终 `pnpm --dir web tsc`、`git diff --check` 通过。

## 完整前端门禁
`pnpm run check:web` 中权限生成物、Proto 枚举、架构、重复函数、Biome 与类型检查通过。既有 `master-data-template/types.ts:107` 有一条 `noConfusingVoidType` 警告及七条提示，不属于本次修改。
默认并发和 16 worker 的全量测试均有两个未改提单页签测试可见性断言失败（`sea-create-layout.test.tsx` 与 `sea-document-validation.test.tsx`）；其余 297 个文件通过。这两个文件单独复核 17 用例全部通过。最终以 `pnpm --dir web exec vitest run --maxWorkers=4` 完整回归：299 个文件、1659 个用例全部通过；另 1 个文件、12 个用例按既有配置跳过，总耗时 390.04 秒。无测试或检查规则被修改或关闭。全量仍有既有本地 3000 端口连接错误输出，不影响测试退出码（0）。
