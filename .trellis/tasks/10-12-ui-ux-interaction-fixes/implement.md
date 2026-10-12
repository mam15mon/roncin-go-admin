# 修复执行路线（planning）

- [x] 用户同意创建任务并进入规划。
- [x] 创建父任务和5个子任务，12项原编号全部分配，证据就近引用。
- [x] 写入每组prd/design/implement、文件所有权及真实spec/research manifests。
- [x] 独立规划复核通过，补齐路由保护与快照隔离/关闭边界；本轮草案提交留档。
- [ ] 用户决定A03计息范围，更新该子任务和总PRD。
- [ ] 进行最终PRD收敛，呈现目标/范围/非目标/验收/关键决定/风险及文档状态。
- [ ] 等待随后明确批准最新规划；此前不task.py start、不改产品、不dispatch implement。
- [ ] 按design协调顺序逐个启动子任务，trellis-implement→trellis-check。
- [ ] 各组实现后定向测试、改动文件Biome、必要tsc、diff检查、spec同步和中文提交。
- [ ] 最终pnpm run check:web + 桌面/窄屏/失败模拟复测，逐项记录通过或仍未通过。
- [ ] AC满足后归档子任务和父任务、记录会话；未实现不得归档为完成。

## 本轮规划验证

JSON/JSONL解析、父子引用/问题编号/源码路径检查、task.py validate及git diff --check。只有规划文档不跑无关产品测试/build。

## 实施门禁

先每组最小定向回归，再必要tsc/架构检查，最终公共层大范围前端门禁。禁TDD；不把所有重型回归堆进同一测试文件。具体命令见子任务implement.md。
