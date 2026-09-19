# ADR 0002: 权限清单单一真相源 manifest.go 与编译期权限键

- 状态：已采纳
- 日期：2026-08-20 建立清单与默认拒绝中间件；2026-08-29 补齐编译期生成
- 主题：契约与生成物

## 背景

系统是多组织 RBAC：后端 HTTP 中间件做鉴权，前端需要权限码做路由级
（`access`）与按钮级控制。权限码若在后端清单和前端常量各写一份，改名、
新增时必然漂移——前端引用已删除的键不会报错，只会静默失去权限判断。

## 决策

- 权限码与 `Requires` 依赖只在
  [../../../server/internal/access/manifest.go](../../../server/internal/access/manifest.go)
  统一定义，是唯一真相源。
- 后端接入「默认拒绝」权限中间件（2026-08-20）：未显式授予权限的接口
  一律拒绝，不靠白名单遗漏兜底。
- 前端不复制权限真相，只消费 `/api/v1/auth/me` 返回的权限集。
- `pnpm run generate:permission-keys` 从清单生成
  `web/src/permissions.generated.ts`；`web/src/access.ts` 的权限键名与之
  编译期对齐，清单改名后不重新生成会直接 tsc 失败。
- 权限目录（`permissions` 表）随 `cmd/migrate` 迁移流程幂等同步，并为
  `administrator` 角色补挂缺失权限；开发期脚本 `dev:permit` 仅作手工兜底。

## 理由

- 权限是安全边界：漂移的后果不是 bug 而是越权或误拒。让后端清单成为
  执行真相、前端键名由清单生成，任何一侧都无法单独腐化。
- 编译期对齐（而非运行期对账）把错误暴露在 CI 阶段，修复成本最低。
- 默认拒绝避免了「新接口忘了挂权限就是裸奔」的经典事故。

## 后果

- 修改 `manifest.go`（新增/改名/删除权限码或调整 `Requires`）后必须执行
  `pnpm run generate:permission-keys`，生成物与源文件同提交。
- 按钮级权限必须复用路由级 `access` 的同一判断结果，页面内禁止硬编码
  第二套规则。
- 新增权限码不需要单独跑同步脚本，正常迁移流程自动落库。
- 契约细节见 [../server/backend/index.md](../server/backend/index.md)
  与 [../../../AGENTS.md](../../../AGENTS.md)。
