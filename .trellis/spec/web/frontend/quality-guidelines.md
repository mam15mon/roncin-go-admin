# 质量规范

## 包管理器

- 统一使用 `pnpm`；不得新增 `npm`、`npx`、Yarn 入口（含文档与脚本）。
- 历史文档中的其他命令以根目录 `AGENTS.md` 和 `package.json` 为准。

## 校验命令（按风险选取最小集）

```bash
pnpm --dir web lint
pnpm --dir web test
pnpm --dir web tsc
pnpm --dir web biome:lint
```

仓库根目录：`pnpm run check:web`、`pnpm run check`（全量）、`pnpm run build`。

## 禁令

- 页面自行拼接后端主机地址（必须走统一请求配置 / 生成客户端）。
- 手改任何生成文件（见 type-safety.md 清单）。
- 硬编码第二套权限规则或复制后端权限清单。
- 引入无关的大型聚合组件；无关格式化混入功能提交。
- 不为通过检查而关闭 lint 规则、跳过类型错误或提交临时产物。
