# 停用状态往返保护执行计划

## 顺序

- [ ] 读取本任务三份文档与 JSONL，核对 Git 状态、影响文件和相邻入口。
- [ ] 按设计先实现功能，再补充触发场景和相邻路径回归（禁止 TDD）。
- [ ] 如有契约改动，与主协调者预约串行运行 `make -C server api`、`pnpm run generate:web-client`，核对生成差异。
- [ ] 定向 Go 测试用 `go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run <相关测试模式>`；涉及真实库只用独立测试库/测试 schema 和正式 SQL 迁移，不能对开发业务库跑重置脚本。
- [ ] 前端测试用 `pnpm --dir web exec vitest run <相对web的测试文件>`，只检查修改文件 Biome；类型影响运行 `pnpm --dir web tsc`。
- [ ] 检查 `git diff --check`，记录测试结果和未执行项，交独立 review，修复 review 问题。
- [ ] 主协调者按相关文件提交中文 Conventional Commit，同步规范并回填验收；不自行暂存/提交他人文件。

## 专项验证

省略 false、显式 false、true 的编辑往返；组织树与详情一致；新建仍启用；成员关系停用不恢复权限；optional 未设置保持未设置；加载/失败不伪造业务状态；缺必填 ID 显式失败。。

## 依赖和交付门禁

客商分支在 R1 之后整合；资源页面同 R2 负责人协调，禁止互相覆盖。
最终父任务运行 `pnpm run check:fast` 后独立 review；通过后合并回 main，验证提交可达，再安全删除工作分支并 push。
