# 资金登记幂等重试执行计划

## 顺序

- [ ] 读取本任务三份文档与 JSONL，核对 Git 状态、影响文件和相邻入口。
- [ ] 按设计先实现功能，再补充触发场景和相邻路径回归（禁止 TDD）。
- [ ] 如有契约改动，与主协调者预约串行运行 `make -C server api`、`pnpm run generate:web-client`，核对生成差异。
- [ ] 定向 Go 测试用 `go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run <相关测试模式>`；涉及真实库只用独立测试库/测试 schema 和正式 SQL 迁移，不能对开发业务库跑重置脚本。
- [ ] 前端测试用 `pnpm --dir web exec vitest run <相对web的测试文件>`，只检查修改文件 Biome；类型影响运行 `pnpm --dir web tsc`。
- [ ] 检查 `git diff --check`，记录测试结果和未执行项，交独立 review，修复 review 问题。
- [ ] 主协调者按相关文件提交中文 Conventional Commit，同步规范并回填验收；不自行暂存/提交他人文件。

## 专项验证

第一次已成功但返回 undefined/抛超时后第二次键一致；普通失败保留输入；内容变更键不同，改回原载荷键恢复；成功后同内容新登记键不同；组织切换隔离；后端同键回放、不一致拒绝。。

## 依赖和交付门禁

可与 R1/R2/R6 并行，生成物和提交由主协调者串行。
最终父任务运行 `pnpm run check:fast` 后独立 review；通过后合并回 main，验证提交可达，再安全删除工作分支并 push。
