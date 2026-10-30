# Research: 订单履约里程碑前端与周边引用清理

- Query: 彻底删除独立订单履约里程碑，需要清理哪些前端、权限夹具、脚本与当前规范引用，如何验证不影响订单操作进度。
- Scope: internal
- Date: 2026-10-30

## Findings

### 页面和模板源代码

| 文件 | 当前职责及证据 | 删除动作 |
| --- | --- | --- |
| `web/src/pages/orders/components/drawers/MilestoneDrawer.tsx` | 独立里程碑抽屉；调用生成客户端 List/Set；文件 23-192 行均为该功能，99 行输出用户看到的标题 | 整文件删除；没有同名独立测试文件 |
| `web/src/pages/orders/list.tsx` | 37-39 行抽屉导入；70 行 ref；275-278 行 `onOpenMilestones` 事件；375-381 行组件与 `milestone.set` 权限接线 | 删除四处接线；其余 activeOrder/写策略仍供相邻抽屉使用，保留 |
| `web/src/components/ui/order-list-template/OrderListTemplate.tsx` | 60 行参数；505-510 行“履约里程碑”菜单；596 行 useMemo 依赖 | 删除对应参数、菜单和依赖，不能仅隐藏菜单 |
| `web/src/components/ui/order-list-template/types.ts` | 185-186 行 `onOpenMilestones` 组件契约 | 删除注释和 prop，仓库内没有额外调用方 |
| `web/src/components/ui/dialog-sizes/constants.ts` | 22 行 MD=860 注释仍列举“履约里程碑” | 仅删除该示例；常量仍被其他业务使用 |
| `web/src/features/audit/audit-presentation.ts` | 191-196 行 `milestone.set` 专属展示，240-245 行 `order.milestone.set` 专属展示 | 删除两条已退役动作展示配置；不删除审计日志或通用未知动作处理 |

**重要保留点**：`NodeIndexOutlined` 除里程碑外还用于协作人员（模板551行）、自拼汇总（561行），不能随菜单一起删除 import。模板里的 `stage` / `statusName` 列、订单详情状态卡和状态流转都属于保留范围；它们不是独立里程碑。

### 权限与生成物

- `web/src/permissions.generated.ts` 含六业务类型（se/si/ae/ai/land/rail）各 `.milestone.read`、`.milestone.set`，合计 12 项。删除后端 Manifest 后通过 `pnpm run generate:permission-keys` 更新，禁止手改。
- `web/tests/fixtures/role-permission-capabilities.json` 的 company 数组含上述 12 项；应同步删除。未找到此夹具的生成入口或服务端对文件名的直接引用，不能假定权限生成命令会自动更新它。
- `web/src/pages/admin/roles.grantable.test.tsx:147-153` 直接使用该夹具并硬断言数量：当前总权限305、company287、system44；仅删除这12项后应为总293、company275、system44。用例标题“完整305项”也需更新。公司/系统有重叠，不能把两者数量相加当总数。
- `web/src/access.ts`、`web/src/app/`、`web/config/routes.ts` 未找到独立里程碑专属路由/access 分支；无需制造额外清理。
- `web/src/services/roncin/orderMilestoneService.ts`、`typings.d.ts` 和 `index.ts` 的里程碑导出、`web/config/openapi.generated.json` 的 `/orders/{orderId}/milestones` 两路由与 schemas/tags 均由 API 生成链清理。`web/scripts/generate-services.mjs` 直接调用 `@umijs/openapi`，未看见项目脚本显式清除已消失服务文件的代码；实施时须检查旧 service 文件是否仍残留，清理失效生成产物要归入生成过程，不能手工修补生成内容。

### 测试与验收脚本

- `web/src/pages/orders/orders-breadcrumbs.test.tsx:110-112` 有废弃客户端 mock，客户端删除后同步删掉整个 mock。
- 未发现 MilestoneDrawer 的独立单测；不可凭删除文件名猜有同名测试。
- `web/src/components/ui/order-list-template/OrderListTemplate.test.tsx:99-124` 已验证“操作进度”“已配舱”“已退关”“已完结”与更多菜单，是最合适的相邻回归位置。建议在现有用例打开更多菜单并等到“费用核算”显示后，断言不存在“履约里程碑”，同时保留既有操作进度状态断言，无需新增重量级套件。
- `web/src/pages/orders/list.test.tsx:185-218` 已按海运/空运验证更多菜单费用入口，可补同一断言或只在模板测试覆盖，避免重复测试实现。
- `web/src/features/audit/audit-presentation.test.ts` 已覆盖未知动作显示“未识别的系统操作”，不需要为已退役动作保留兼容映射。
- `scripts/*.mjs`、`web/scripts/*.mjs`、`web/tests/e2e/*.ts` 未找到里程碑引用；当前不用改额外E2E/根脚本。

定向验证建议：

```bash
pnpm --dir web exec vitest run src/components/ui/order-list-template/OrderListTemplate.test.tsx src/pages/orders/list.test.tsx src/pages/orders/orders-breadcrumbs.test.tsx src/pages/admin/roles.grantable.test.tsx src/features/audit/audit-presentation.test.ts
pnpm --dir web tsc
```

改完后对实际修改文件执行 Biome；最终跨层验收执行 `pnpm run check:fast`，不要新写只扫描源码是否含字符串的镜像测试。

### 当前规范与旧历史记录

- `.trellis/spec/server/backend/capability-navigation.md:40` 当前能力导航仍列出 `order_milestone.go` 和“里程碑”，删除该项，不影响同一行的其他辅助实体。
- `.trellis/spec/server/backend/order-lock-and-document-version.md:130` 当前验证清单用“草稿、POD、货物、里程碑等”举例，去掉退役实体或换成仍存在的子资源。
- `.trellis/spec/domain/`、`.trellis/spec/web/` 全量检索未发现独立里程碑介绍，不能为“同步规范”虚构新导航项。
- `PLAN.md` 与 `docs/migration-matrix.md` 有多处项目早期完成记录/旧迁移分析，已知不少内容甚至声称已删除的里程碑模板仍存在。建议在当前状态说明中明确“独立订单里程碑已退役”，并删除当前能力描述，避免继续指导实现已删除功能；旧系统对照列、原阶段历史完成记录可保留其历史语义，不宜为本次功能删除重写全部早期规划。
- `.trellis/tasks/archive/**`、`.trellis/workspace/**` 是历史任务/会话记录，不应批量替换或清空；正式 SQL 旧迁移也由 Schema 研究负责保持迁移链可重放，不作为“垃圾代码”删除。

## Code Patterns

- 当前删除应贯穿公开模板 props → 页面 ref/回调 → 抽屉组件 → 生成客户端，而不是只修改菜单项的可见性。
- 图标、状态流转和通用抽屉模板存在真实共享；删除专属能力不等于删除共享基础设施。
- 权限真相仍为服务端 Manifest，前端生成键和真实能力夹具应同批同步；角色授权测试当前有固定计数约束。

## External References

本题依赖仓库源码即可回答，未使用外部资料。前端当前规范版本为 Vite 7 / React Router v8 / Ant Design 6 / ProComponents，见前端规范索引。

## Related Specs

- `.trellis/spec/web/frontend/index.md`
- `.trellis/spec/web/frontend/component-guidelines.md`
- `.trellis/spec/web/frontend/quality-guidelines.md`
- `.trellis/spec/server/backend/capability-navigation.md`
- `.trellis/spec/server/backend/order-lock-and-document-version.md`
- 根 `AGENTS.md` 的权限/生成链、保留既有工作、相关验证和提交规则。

## Caveats / Not Found

- 没有产品代码修改、没有运行数据库或 Git 操作，仅写入此研究文件。
- 没有发现独立 MilestoneDrawer 单测、页面路由、root验收脚本引用或权限夹具生成器。
- 旧审计数据保留不等于保留退役专属显示兼容层；专属 action 映射删除后由已有未知动作UI处理。若主会话决定保留审计动作中文解释，应明确这是审计可读性决策，不把它当运行功能入口。
