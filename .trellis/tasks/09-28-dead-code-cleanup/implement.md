# 执行清单

- [ ] 启动前检查 `git status`，记录已有工作；重复运行引用核查，特别检查动态路由、barrel、测试 mock 和 Tailwind 配置加载。最终“删／保留”证据写入 `research/verification.md`。
- [ ] 阅读 `.trellis/spec/domain/`、前后端规范及 `trellis-before-dev`；确认不修改 `.proto`、Ent schema、正式迁移或权限码。
- [ ] 第一批：删除前端两个无消费者模块与过期 mock、导出；只清理 `split.tsx` 已诊断的未使用导入。运行费用页／拆票页定向测试、修改文件 Biome、`pnpm --dir web tsc`、`pnpm run check:architecture`、`git diff --check`，通过后提交。
- [ ] 第二批：删除后端权限分类链和 `RequireBaselineWrite`；运行相关 access/biz 包测试、`go -C server vet ./internal/access ./internal/biz`、`git diff --check`，通过后提交。
- [ ] 第三批：核对补录三条事务内状态路径和拆票指纹的真实调用。按设计的判定实施最小清理；验证审批/驳回/撤回终态竞争、拆票幂等和相邻拒绝路径，通过后提交。若证据支持保留，记录原因，不强删。
- [ ] 条件项：核对 Tailwind 配置；只有证明不读取且前端构建通过才删除，并与验证同批提交。构建仅因本项涉及构建配置而运行。
- [ ] 记录 `noUnusedLocals/noUnusedParameters` 初始 212 处（其中 189 处 `React`）与本次剩余数量；不为清零跨越预定文件范围。两组合理重复映射保留。
- [ ] 最终 `pnpm run check:fast`、`git diff --check` 和 `git status`；检查无新增死导入、无生成物／迁移／数据库变动，报告 PASS/FAIL/SKIP；按 Trellis 检查流程复核并提交、归档。

## 验证命令提示

```bash
pnpm --dir web exec vitest run src/pages/orders/fees.test.tsx <受影响拆票测试>
pnpm --dir web exec biome check <本批次修改文件>
pnpm --dir web tsc
pnpm run check:architecture
go -C server test ./internal/access ./internal/biz ./internal/data -run '<受影响用例>' -count=1
go -C server vet ./internal/access ./internal/biz
pnpm run check:fast
git diff --check
```

具体测试文件与 `-run` 模式在实现后按真实影响填写，不能直接执行占位符命令。测试在实现后进行，不采用 TDD。
