# 技术设计：取消详情持久草稿，统一有损离开确认

## 已批准的交互边界

用户最终选择最简方案：切换、关闭或其他会丢输入的离开均先确认；取消则留在原页，确认则放弃修改。此前“切换后保留输入”要求已被替代。本期不增加内存编辑会话、整页缓存或新的保存恢复平台。

## 现有机制与行为缺口

- `AppLayout` 使用单一 `Outlet`，系统页签只记录导航地址。切换路由就会卸载订单表单。
- `OrderFormTemplate` 已集中维护 dirty、保存、重置及可选持久草稿；详情页还自行读取草稿恢复版本。本任务删除详情依赖，保留模板对新建页的草稿能力。
- `tabCloseGuard` 支持多 guard、批量一次确认及临时确认放行；`useRouteLeaveGuard` 已用于客商、HBL，订单主详情尚未接入。
- 工作区没有原生 `beforeunload` 保护；关闭页签菜单没有显式“关闭全部”入口。

研究证据见 `research/order-drafts.md` 和 `research/tab-close.md`。

## 方案

### 1. 订单详情只保留当前活表单

- 删除 `detail.tsx` 的 `restoredDraft`、storage 导入、旧值/旧版本恢复分支；本次挂载初值均来自服务端响应。
- 不再向详情模板提供持久化所需的完整草稿身份；保留稳定 `tabKey`、用户公司 scope 和 React 实例 key。scope 同时服务外部变化查询，不能因去掉草稿而误删。
- `OrderDetailFormTemplate` 固定首次初值、`documentBaseline`、显式刷新令牌、15 秒/回焦变化检查保持现有职责：当前表单编辑过程中不让后台重读覆盖输入或版本。
- 不再支持详情旧存储键，不恢复、不迁移、不建立兼容分支；共享 dirty 的持久草稿分支只识别仍被支持的新建与独立 HBL 编辑草稿，避免遗留详情记录让干净页面误提示。不得扫描清理其他用户、其他公司或待确认创建记录。

### 2. dirty 仍由模板所有，离开守卫读实时状态

- 模板通过同步 ref 提供实时 dirty，真实用户编辑和 `applyValues` 标记；初始化、只读浏览、候选刷新不标记。
- 保存明确成功、`resetTo` 和原生重置清 dirty；失败、校验未过、409、读取最新数据失败继续保留。
- 已有 dirty 不因外部锁单变 readonly 而注销关闭保护；readonly 仅禁止继续编辑。
- 订单详情复用 `useRouteLeaveGuard`，查询当前页签实时 guard，不依赖旧持久草稿。具体接入在详情编排与共享模板的现有公开边界内完成，不添加页面受控 dirty 镜像。
- 数据路由测试使用真实 data router 验证导航阻断；不能在产品代码里为非 data router 测试环境静默跳过守卫。

### 3. 一次动作只确认一次

- 站内跳转：`useRouteLeaveGuard` 对实际 pathname/search 变化提示“有未保存的修改，是否放弃并离开？”，提供“继续编辑／放弃并离开”。不对同页锚点滚动新增确认。
- 页签关闭：现有 `confirmIfTabsDirty` 统一提示“有未保存的修改，是否放弃并关闭？”，提供“继续编辑／放弃并关闭”。单关、关其他、关左/右和新增关闭全部共用同一入口。
- 批量先计算真实关闭范围，确认一次后才关闭；保留固定工作台及不在关闭范围的页签。
- 保留 `runConfirmedDeparture` 和 `isTabDepartureConfirmed`，已确认关闭或切换工作区时不再触发路由二次确认；异步失败撤销临时放行。
- 保留 `currentLocation.key` 的源路由检查，防止路由提交后尚未清理的旧 blocker 拦截下一次前进或后退。
- 主动退出登录在销毁身份之前复用离开确认；失败保留当前页面。强制鉴权失效不允许被旧表单阻止。

### 4. 浏览器原生离开提醒

- 在 `OrganizationWorkspace` 生命周期内统一注册一个 `beforeunload` 监听。
- 事件发生时调用现有实时 `getAnyDirtyTabKey()`，存在未保存编辑才 `preventDefault` 并设置 `returnValue`。
- 不扫描 storage，不弹 Antd 模态模拟关窗，不为每个字段重复注册；工作区卸载后移除监听。
- 浏览器决定原生提示的具体文案及展示条件，产品不承诺自定义关窗文本。

### 5. 保存结果尚未确认的边界

继续现有 `useSeaOrderSave` 的在途冻结、unknown 同键重试、committed 后只重读行为。本期不持久化详情待确认保存状态。用户确认离开意味着放弃本地待处理状态，不能宣称已经取消发送到服务端的请求；必要提示复用现有状态说明“保存结果尚未确认，离开后请核对订单”。再次进入以后端实际状态为准。新建订单的独立持久幂等记录完全保留。

## 文件职责与预计改动

| 文件或目录 | 必要性 |
| --- | --- |
| `web/src/pages/orders/detail.tsx` 及详情相关回归 | 删除持久草稿读取/身份，接入离开确认，保留版本及刷新保护 |
| `web/src/components/ui/order-template/OrderFormTemplate.tsx` 及相关类型/测试 | 确保关闭 guard 使用即时 ref，readonly 后保护已有 dirty，保持新建持久化 |
| `web/src/components/layout/tabCloseGuard.ts`、`useRouteLeaveGuard.ts`、`formDraft.ts` 及对应测试 | 有效持久草稿边界、统一动作确认、已确认导航复用 |
| `web/src/components/layout/TagsView.tsx` 及独立关闭测试 | 关闭全部入口、真实关闭范围、一次确认 |
| `web/src/components/layout/OrganizationWorkspace.tsx` 及独立测试 | 原生关窗监听及身份卸载清理 |
| `web/src/components/RightContent/AvatarDropdown.tsx` 及测试 | 用户主动退出前确认，接口失败不丢输入 |
| `web/tests/` 必要测试包装 | 仅在现有非 data router 测试不能运行真实 blocker 时补最薄包装 |
| `.trellis/spec/web/frontend/state-management.md`、`organization-switch.md`、`capability-navigation.md` | 同步详情不再恢复草稿的契约与能力导航 |

`new.tsx`、`new.test.tsx`、`new-source.test.tsx`、`component-guidelines.md` 含其他窗口未提交改动，本任务不接管、不覆盖；若共享接口确有影响，先报告并只合并必要变更。

## 验证与风险控制

- 用真实路由加真实 Form 验证编辑→切工作台→取消留原页→确认离开→返回显示服务端值；包含海运和空运。
- 历史详情草稿预埋后无恢复、无假 dirty；新建草稿、独立 HBL 草稿及创建幂等仍通过。
- 单关及批量确认各覆盖取消/确认，关闭全部只确认一次且路由不重复阻断。
- 编辑后立即关闭、保存失败、外部变 readonly、保存/重置成功、刷新失败、旧 blocker 迟到注销、工作区卸载均覆盖。
- 原生事件验证 clean 放行、dirty 阻断、成功后放行、StrictMode/身份切换无重复监听；条件允许用浏览器补验原生提示。
- 共享守卫与模板影响多个模块，最终运行完整前端门禁；不修改服务端，不需要全后端测试或生产构建。

## 回退方式

按本任务代码提交整体回退；不涉及数据库迁移、服务端契约或外部状态变更。回退必须与对应测试和规范同步，不保留双模式详情恢复兼容分支。
