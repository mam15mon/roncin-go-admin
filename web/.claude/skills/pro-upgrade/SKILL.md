---
name: pro-upgrade
description: >
  对 Roncin 的 Ant Design、ProComponents 或前端工程依赖执行已授权的升级，
  核对 API 变化并保留 Vite、React Router、工作区和业务边界。
  适用于明确的依赖升级或版本迁移，不用于一般文档更新，也不从上游模板覆盖项目。
allowed-tools:
  - Bash(git clone *)
  - Bash(diff *)
  - Bash(pnpm install)
  - Bash(pnpm run lint*)
  - Bash(pnpm run build*)
  - Bash(pnpm run tsc*)
  - Bash(pnpm exec antd *)
  - Read
  - Edit
  - Write
  - Glob
  - Grep
---

# Roncin 前端依赖升级

本项目的应用壳源自 Ant Design Pro，现行工程采用 Vite、React Router、React Query 与 Go 同域交付。上游模板是参考资料，不拥有本项目的路由、权限、运行时配置或业务代码；不得用“同步最新模板”恢复 Umi 或直接覆盖框架文件。

## 先确认实际差异

1. 读取根 `AGENTS.md`、任务要求、根及 `web/package.json`、根 `pnpm-lock.yaml` 与现有配置。目标是完成用户授权的依赖升级范围，不主动更新全部依赖。
2. 检查 `git status`，保留已有未提交工作；仅在改动确实无法安全分离时向用户说明冲突，不要求为了升级清空工作区或自动 stash。
3. 从本项目安装版本与官方发布记录确认当前、目标及兼容性约束。需要上游模板作对照时放在独立临时目录，只比较相关 API/配置差异，不把模板依赖集合当目标版本清单。
4. 对公开 API 变化核对所有真实调用方；本项目不默认保留历史兼容分支，契约变化一次性同步实现与验证。

## 保护的项目入口

| 入口（相对 `web/`） | 升级时的处理 |
| --- | --- |
| `vite.config.ts`、`config/` | 保留 Vite 构建、测试、代理、主题和集中路由；仅按目标依赖实际 API 调整 |
| `src/router/`、`src/app/` | 保留 React Router 装配、认证工作区与身份隔离，不采用上游 Umi 运行时 |
| `src/access.ts`、`src/requestErrorConfig.ts`、`src/utils/requestClient.ts` | 保护后端有效能力判定、请求与错误处理契约 |
| `src/pages/`、`src/features/`、`src/components/` | 保留业务语义、公开边界、表格采纳快照和表单生命周期；只改目标 API 必需的调用 |
| `src/services/roncin/`、`config/openapi.generated.json` | 自动生成；禁止手改或按普通业务文件覆盖 |
| `src/permissions.generated.ts` | 后端权限清单生成；禁止手改 |
| `src/enums.generated.ts`、`src/errorReasons.generated.ts` | Proto 常量生成；禁止手改 |
| `tests/`、就近单测、`scripts/`、根 `patches/` | 核对与目标依赖相关的测试基础设施、生成脚本和补丁适用性，不机械删除 |

实际文件与目录以仓库为准，不创建旧模板的 `src/app.tsx`、`config/config.ts` 或样例服务目录。通用模块不得反向依赖 pages/features，提取能力时更新导航并运行架构检查。

## 修改与验证

依赖安装在仓库根目录执行 `pnpm install`，提交对应清单和根锁文件。不要采用另一个包管理器或直接复制上游锁文件。涉及 antd 时可在 `web/` 目录用本地 CLI 查询：

```bash
pnpm exec antd env --format json
pnpm exec antd info <组件> --format json
pnpm exec antd changelog <当前版本> <目标版本> --format json
pnpm exec antd migrate <当前主版本> <目标主版本> --format json
```

先实现 API 适配，再按风险补针对性测试，禁止 TDD。开发期使用修改文件的 Biome 检查和相对 `web/` 路径的定向 Vitest；不要运行全仓自动修复去改无关文件。依赖和生产入口的最终验收从仓库根目录按风险执行：

```bash
pnpm run check:web
pnpm run build
```

如果同时影响后端或跨层契约，使用根规范规定的全栈门禁。测试失败时修正实际 API 或实现，不通过关闭规则掩盖错误。构建会写入前端产物及服务端资源暂存，只提交仓库规定的源文件、必要生成物和锁文件，不提交缓存。

客户端及常量需要更新时使用真实源与生成入口：

- API 变化：先改 Proto，执行 `make -C server api`，再在根目录执行 `pnpm run generate:web-client`。
- 权限 Manifest 变化：根目录执行 `pnpm run generate:permission-keys`。
- 仅依赖升级时不为“刷新”而改 API 契约或生成器。

完成后报告实际版本变化、必要配置/API 适配、验证结果、尚未覆盖的真实业务场景；按根协作规范提交可验证变更组。清理仅限本次创建且确认无需保留的临时目录，保留用户文件和其他会话工作。
