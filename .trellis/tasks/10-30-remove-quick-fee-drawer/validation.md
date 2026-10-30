# 实施与验证记录

## 实施结果

- 详情页移除快速费用抽屉入口与挂载；列表“费用核算”进入既有费用页面。
- 删除抽屉、专属列定义、汇率 Hook、独占基础列模块及对应专属测试。
- 清理四个详情测试的过时 mock，增加海运/空运列表真实菜单导航验证。
- 更新前端能力导航，费用维护统一到独立页面。

## 验证

- 定向六文件 52 用例通过；补齐主数据 mock 后列表六用例单跑通过，无网络/act 警告。
- 修改文件 Biome、前端 tsc、git diff --check 通过。
- 独立 review 暂无问题；check:web 前置生成物、架构、重复检查、lint 通过（已有非阻断提示）。
- 首次全量 Vitest：332 文件通过、2 文件失败、1 跳过；失败为未修改的海运单证路径。两失败文件分别单跑 8/8、1/1 通过，16 worker 复核时原两个失败全部通过，但他人实时修改的 sea-form-labels 两用例失败，不能报告完整门禁全绿。不修改配置或用例超时。

## 并行工作边界

其他会话持续修改表单标签相关文件，未纳入本任务提交；另有 `fix/sea-container-entry-permission` 在独立 worktree 工作，安全清理分支时保留该活跃且未合并分支。

## 追加排查要求

用户要求不能直接归因为并发负载，重点排查动画与异步时序。暂停合并；独立 review 代理增加临时阶段测时，另由只读代理分析本地动画库实现。单跑通过不视为根因结论。

## CPU 排查证据

两个 fixture 已设置 motion:false，全局禁 CSS 动画；但 dialog prepare 在该设置下仍经过两次 rAF。实际 identity 探针 rAF=0，HOUSE rAF=4，不能直接把帧等待当根因。

使用 node:inspector 采样（无 timer/style spy）复现 identity 查找失败：第二次请求在查找超时前未触发。HOUSE profile 约 24.95 秒，happy-dom 惰性样式计算约 16.15 秒（65%），queryAllByRole/可访问名称/可见性路径约 13.88 秒；动画 useStatus 累计约 65ms。此前只测 getComputedStyle 函数返回未包含惰性属性计算，已纠正。

证据位于临时 /tmp/house-cpu.cpuprofile、/tmp/identity-cpu.cpuprofile，排查探针不提交。后续做精确查询与廉价等待的 A/B 验证，保留可见性/启用状态断言，不放宽超时或改全局并发。

## 对照实验结论

首次改为按文本定位后，HOUSE 的按钮可见性断言失败，证实文本进入 DOM 早于 Modal prepare 结束；原昂贵 role 查询恰好消耗时间绕过了此窗口。改为显式等待已定位按钮可见/可用，不跳过可见性检查。

在相同 inspector 条件下：HOUSE 约 24.95s → 5.98s，identity 原 3.82s 失败 → 2.16s 通过。去掉 profiler/spies 再复核：HOUSE 4.304s、identity 1.445s，均通过。没有增加超时、降低正式配置并发或 mock 掉真实组件。两临时 diagnostic 测试已删除。

正式最小修复：精确文本定位真实 button，等待单节点可见且启用再交互，保留全部业务载荷与旧候选清空断言；同步记录质量规范，避免把昂贵轮询与动画帧等待混为一谈。
