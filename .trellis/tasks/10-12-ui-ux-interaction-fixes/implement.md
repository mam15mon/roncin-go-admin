# 修复执行路线（实现与验收完成）

- [x] 用户同意创建任务并进入规划。
- [x] 创建父任务和5个子任务，12项原编号全部分配，证据就近引用。
- [x] 写入每组prd/design/implement、文件所有权及真实spec/research manifests。
- [x] 独立规划复核通过，补齐路由保护与快照隔离/关闭边界；本轮草案提交留档。
- [x] 用户明确选择移除未完成计息入口，已同步子任务和总PRD。
- [x] 完成最终PRD收敛及最终规划摘要，记录目标/范围/非目标/验收/关键决定/风险及文档状态。
- [x] 用户已回复“好的”，明确批准855e467d最新最终规划，可启动子任务。
- [x] 按design协调顺序逐个启动子任务，trellis-implement→trellis-check。
- [x] 各组定向测试、改动文件Biome、tsc、diff、spec判断/同步和中文提交完成。
- [x] 最终pnpm run check:web及桌面/窄屏/失败模拟复测通过，12项结果见research/final-acceptance.md。
- [x] 全部AC满足，父子任务进入归档与会话记录阶段。

## 本轮规划验证

JSON/JSONL解析、父子引用/问题编号/源码路径检查、task.py validate及git diff --check。只有规划文档不跑无关产品测试/build。

## 实施门禁

先每组最小定向回归，再必要tsc/架构检查，最终公共层大范围前端门禁。禁TDD；不把所有重型回归堆进同一测试文件。具体命令见子任务implement.md。

## 最终集成验收

2026-10-12完整`pnpm run check:web`退出码0；218测试文件、1181例通过，保留既有1文件/12例跳过。全部12项及62项浏览器证据通过，集成结果见同月archive中父任务research/final-acceptance.md。
