# 客商业务身份资料保护执行计划

## 顺序

- [ ] 读取本任务三份文档与 JSONL，核对 Git 状态、影响文件和相邻入口。
- [ ] 按设计先实现功能，再补充触发场景和相邻路径回归（禁止 TDD）。
- [ ] 如有契约改动，与主协调者预约串行运行 `make -C server api`、`pnpm run generate:web-client`，核对生成差异。
- [ ] 定向 Go 测试用 `go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run <相关测试模式>`；涉及真实库只用独立测试库/测试 schema 和正式 SQL 迁移，不能对开发业务库跑重置脚本。
- [ ] 前端测试用 `pnpm --dir web exec vitest run <相对web的测试文件>`，只检查修改文件 Biome；类型影响运行 `pnpm --dir web tsc`。
- [ ] 检查 `git diff --check`，记录测试结果和未执行项，交独立 review，修复 review 问题。
- [ ] 主协调者按相关文件提交中文 Conventional Commit，同步规范并回填验收；不自行暂存/提交他人文件。

## 专项验证

带完整资料及散客标识的客商切换/增加角色；停用客商保持停用；角色未变；非法/空角色；跨公司/无编辑权限；错误回滚；全量档案正常更新相邻路径。。

## 依赖和交付门禁

独立交付；R3 与本任务有客商入口交集，先完成本任务再审 R3 客商分支；共享生成物统一串行生成。
最终父任务运行 `pnpm run check:fast` 后独立 review；通过后合并回 main，验证提交可达，再安全删除工作分支并 push。
