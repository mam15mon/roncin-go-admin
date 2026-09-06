# 质量规范

## 包管理器

- 统一使用 `pnpm`；不得新增 `npm`、`npx`、Yarn 入口（含文档与脚本）。
- 历史文档中的其他命令以根目录 `AGENTS.md` 和 `package.json` 为准。

## 分阶段校验规则

### 1. 开发过程中

只验证当前改动，不重复运行完整前端门禁：

```bash
pnpm --dir web exec vitest run <test-file> [more-test-files...]
pnpm --dir web exec biome check <changed-file> [more-changed-files...]
```

当前 `web` 的 `test` 脚本固定执行 `vitest run`。禁止使用下面的写法作为定向测试，
它会意外运行全量套件：

```bash
pnpm --dir web test -- <test-file>
```

### 2. 普通本地提交

- 运行受影响的定向测试与 `git diff --check`。
- 改动公共类型、Hook/组件接口、路由或难以由定向测试覆盖的类型链时，增加
  `pnpm --dir web tsc`。
- 提交动作本身不触发全量 `pnpm --dir web test`、`pnpm run check:web` 或构建。

### 3. 最终验收

整个任务或批准阶段完成、准备最终验收、Trellis 归档或推送时，才执行一次风险
匹配的完整前端门禁：

```bash
pnpm --dir web lint
pnpm --dir web test
pnpm --dir web tsc
pnpm --dir web biome:lint
```

`pnpm run check:web` 用于完整前端验收，`pnpm run check` 用于前后端全量验收。
`pnpm run build` 只在构建配置、依赖、生产入口等相关变更、发布前验收或用户明确
要求时运行，不是普通 UI 修改的固定门禁。

契约、权限、生成物、跨领域公共组件或大范围状态管理等高风险变更，可以在最终
验收前增加完整检查，但必须说明风险依据。若只是局部页面样式、文案或单组件行为，
最终验证也应避免无依据地扩大到后端全量门禁。

## 禁令

- 页面自行拼接后端主机地址（必须走统一请求配置 / 生成客户端）。
- 手改任何生成文件（见 type-safety.md 清单）。
- 硬编码第二套权限规则或复制后端权限清单。
- 引入无关的大型聚合组件；无关格式化混入功能提交。
- 不为通过检查而关闭 lint 规则、跳过类型错误或提交临时产物。
