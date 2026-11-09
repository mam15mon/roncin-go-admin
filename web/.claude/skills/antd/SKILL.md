---
name: antd
description: >
  查询本项目 Ant Design 的组件 API、示例、语义样式和设计令牌，分析用法或辅助版本迁移。
  适用于编写或排查 antd 组件，以及明确的 antd 查询和升级任务。
allowed-tools:
  - Bash(antd *)
  - Bash(antd bug*)
  - Bash(antd bug-cli*)
  - Bash(pnpm exec antd *)
  - Bash(pnpm exec antd bug*)
  - Bash(pnpm exec antd bug-cli*)
---

# Roncin 的 Ant Design 本地查询

`web/package.json` 已声明 `@ant-design/cli`。本文件中的命令均在 `web/` 目录执行，使用 `pnpm exec antd`；从仓库根目录执行时使用 `pnpm --dir web exec antd`。组件知识可从本地元数据查询，目标版本以本项目实际安装的 antd 为准。查询新发布版本时须核对官方发布记录，不能把 CLI 缓存当作外部最新状态。

## 组件编写与排查

先查看所改组件的 API，再按问题查询示例、语义结构或令牌；不依据旧版记忆猜测属性：

```bash
pnpm exec antd info Button --format json
pnpm exec antd demo Button basic --format json
pnpm exec antd semantic Button --format json
pnpm exec antd token Button --format json
pnpm exec antd doc Table --lang zh --format json
```

排查当前环境或使用方式：

```bash
pnpm exec antd env --format json
pnpm exec antd doctor --format json
pnpm exec antd lint ./src/components/MyForm.tsx --format json
pnpm exec antd usage ./src --filter Form --format json
```

最后两条路径按实际改动替换。优先检查修改文件，不把全仓扫描叠加为普通提交的固定门禁。统计与诊断不能替代项目的类型检查和业务测试；验证阶段以根 `AGENTS.md` 为准。

## 版本比较与迁移

仅在用户任务包含依赖升级时执行迁移；查询迁移建议不等于授权升级。按当前与目标的实际版本替换参数：

```bash
pnpm exec antd changelog <当前版本> <目标版本> --format json
pnpm exec antd migrate <当前主版本> <目标主版本> --format json
pnpm exec antd list --format json
```

不自动覆盖 Vite、React Router、公司工作区或权限实现；升级边界见相邻的 `pro-upgrade` 技能和根协作规范。

## 问题报告

只有用户明确要求向外部项目报告问题时才准备报告。先用 `env` 收集必要环境，移除秘密、公司数据及内部地址，使用 `antd bug` 或 `antd bug-cli` 的预览输出展示标题、复现、期望和实际行为；获得发送授权后才使用 `--submit`。用户已有明确发送授权时不重复请求；仅排查问题不意味着允许发送。

```bash
pnpm exec antd bug --help
pnpm exec antd bug-cli --help
```

未带 `--submit` 的预览按当前 CLI 帮助核对，不猜测参数。设置 `ANTD_NO_AUTO_REPORT=1` 时不主动建议报告。

## MCP 接入示例

仅在用户要求配置 MCP 时修改配置；下面的绝对路径占位符须替换为实际仓库的 `web/` 目录：

```json
{
  "mcpServers": {
    "antd": {
      "command": "pnpm",
      "args": ["--dir", "<仓库绝对路径>/web", "exec", "antd", "mcp"]
    }
  }
}
```

MCP 与 CLI 使用同一份组件知识，实际工具集合以当前安装版本暴露的能力为准。

## 参数约定

| 参数 | 用途 |
| --- | --- |
| `--format json` | 需要结构化处理时使用 |
| `--version <版本>` | 指定目标组件版本；省略时按 CLI 实际检测结果核对 |
| `--lang zh` | 中文输出 |
| `--detail` | 查询更完整的说明、引入版本等元数据 |

本技能不修改生成客户端，不引入另一套依赖安装入口，不替代根协作规范中的权限、生成、验证或提交要求。
