# 信用额度统一本币口径执行计划

## 顺序

- [ ] 读取本任务三份文档与 JSONL，核对 Git 状态、影响文件和相邻入口。
- [ ] 按设计先实现功能，再补充触发场景和相邻路径回归（禁止 TDD）。
- [ ] 如有契约改动，与主协调者预约串行运行 `make -C server api`、`pnpm run generate:web-client`，核对生成差异。
- [ ] 定向 Go 测试用 `go -C server test -p 32 ./internal/biz ./internal/service ./internal/data -run <相关测试模式>`；涉及真实库只用独立测试库/测试 schema 和正式 SQL 迁移，不能对开发业务库跑重置脚本。
- [ ] 前端测试用 `pnpm --dir web exec vitest run <相对web的测试文件>`，只检查修改文件 Biome；类型影响运行 `pnpm --dir web tsc`。
- [ ] 检查 `git diff --check`，记录测试结果和未执行项，交独立 review，修复 review 问题。
- [ ] 主协调者按相关文件提交中文 Conventional Commit，同步规范并回填验收；不自行暂存/提交他人文件。

## 专项验证

CNY 与非 CNY 公司本币；外币结算+本币额度；伪造外币额度写入拒绝；既有异常额度显式暴露且不自动改值；嵌套和独立写入口一致；余额低于/等于/高于额度、零/未设额度、散客、提醒/干预；查询组织隔离。。

## 依赖和交付门禁

依赖 R1 后再修改共享 partner 写入口；与 R3 客商页面交叉由同一负责人或主协调串行整合。
最终父任务运行 `pnpm run check:fast` 后独立 review；通过后合并回 main，验证提交可达，再安全删除工作分支并 push。
