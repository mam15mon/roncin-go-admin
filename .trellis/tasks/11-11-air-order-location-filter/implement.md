# 空运地点筛选与权威投影一致执行计划

## 顺序

- [ ] 读取本任务三份文档与 JSONL，核对 Git 状态、影响文件和相邻入口。
- [ ] 按设计先实现功能，再补充触发场景和相邻路径回归（禁止 TDD）。
- [ ] 如有契约改动，与主协调者预约串行运行 `make -C server api`、`pnpm run generate:web-client`，核对生成差异。
- [ ] 定向 Go 测试用 `go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run <相关测试模式>`；涉及真实库只用独立测试库/测试 schema 和正式 SQL 迁移，不能对开发业务库跑重置脚本。
- [ ] 前端测试用 `pnpm --dir web exec vitest run <相对web的测试文件>`，只检查修改文件 Biome；类型影响运行 `pnpm --dir web tsc`。
- [ ] 检查 `git diff --check`，记录测试结果和未执行项，交独立 review，修复 review 问题。
- [ ] 主协调者按相关文件提交中文 Conventional Commit，同步规范并回填验收；不自行暂存/提交他人文件。

## 专项验证

A→B 后按 B 命中按 A 不命中，起运与目的分别及组合；同主单多票；总数/分页一致；终止或缺关联按现有读投影语义；海运回归；公司隔离。。

## 依赖和交付门禁

独立交付，可与前端任务并行。
最终父任务运行 `pnpm run check:fast` 后独立 review；通过后合并回 main，验证提交可达，再安全删除工作分支并 push。
