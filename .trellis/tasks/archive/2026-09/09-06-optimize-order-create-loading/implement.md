# 实施计划：优化新建订单页面主数据加载体验

## 阶段与执行步骤

### 阶段 1：数据层重构（组织隔离缓存与按需加载）

- [x] **步骤 1.1**：新增 `web/src/utils/order-options-cache.ts`
  - 新增基于 `organizationId` 的 `Map` 缓存：`masterOptionsCache`、`portsCache`、`airportsCache`、`personnelOptionsCache`；
  - 实现并发防抖共享与 Promise reject 自动清理机制；
  - 导出 `clearOrderMasterDataCache(organizationId?: string)`，支持当前组织定向失效与测试期全量清理；
  - 导出 `getMasterDataOptions`、`getCachedPorts`、`getCachedAirports`、`getOrderPersonnelOptions`，供订单模块和登出流程向下依赖，禁止全局组件反向导入 `pages/orders`。
- [x] **步骤 1.2**：修改 `web/src/pages/orders/common.ts`
  - 消费共享缓存函数并重构 `fetchOrderMasterData(organizationId: string, category?: 'sea' | 'air')`；
  - 保持候选项格式转换与远程联想职责不变。
- [x] **步骤 1.3**：修改 `web/src/pages/orders/list-resources.ts`
  - 接入组织级缓存与 `config?.category` 按需拉取逻辑，打通列表与新建页的缓存复用。
  - 将组织 ID 纳入加载依赖，增加过期请求结果丢弃保护。
- [x] **步骤 1.4**：修改 `web/src/pages/orders/use-order-create-options.ts`
  - 读取当前 `organizationId` 与 `config?.category` 传入 `fetchOrderMasterData`；
  - 导出 `{ loading, error, retry, ... }`，缺少有效组织时不发请求并返回明确错误；
  - `retry` 先失效当前组织缓存，再重新执行加载；
  - 接入共享的 `getOrderPersonnelOptions`；
  - 通过请求序号与当前组织 ID 校验丢弃组织切换后晚到的旧响应。
- [x] **步骤 1.5**：修改 `web/src/pages/orders/use-order-detail-data.ts`
  - 传入 `organizationId` 与 `category` 复用主数据与地点缓存；
  - 扩展现有请求序号保护，确保组织切换后旧组织响应不能写回当前详情状态。
- [x] **步骤 1.6**：修改 `web/src/components/RightContent/AvatarDropdown.tsx`
  - 登出接口成功后调用无参数 `clearOrderMasterDataCache()`，再清除当前用户状态并跳转登录页；
  - 登出失败时保持当前会话缓存，不提前改变现有登录态。

### 阶段 2：页面容错与模板骨架屏升级

- [x] **步骤 2.1**：修改 `web/src/components/ui/order-template/OrderFormTemplate.tsx`
  - 将 `loading` 态从突兀的大卡片 Spin 升级为符合系统规范的分节骨架屏（业务卡片骨架 + 运输卡片骨架），保留 `loadingTip`。
- [x] **步骤 2.2**：修改 `web/src/pages/orders/new.tsx`
  - 当数据加载失败（`error`）时渲染带有“重新加载”操作的错误结果卡片，严防展示空表单；
  - 优先按 `GENERAL` 业务码匹配默认货物类别。

### 阶段 3：针对性单元测试与验证

- [x] **步骤 3.1**：新增 `web/src/utils/order-options-cache.test.ts` 并更新 `web/src/pages/orders/orders.test.ts`
  - 覆盖测试：
    1. sea 模式不请求机场，air 模式不请求港口，默认双拉取；
    2. 同组织并发调用去重，只产生 1 次网络请求；
    3. 同组织顺序调用命中缓存；
    4. 请求失败自动移出缓存，下次调用重新发起；
    5. 不同组织隔离不共享；
    6. `clearOrderMasterDataCache(orgId)` 定向失效，`clearOrderMasterDataCache()` 完整清理；
    7. 优先基于 `GENERAL` 码选中默认普货。
- [x] **步骤 3.2**：新增或更新 Hook 与资源加载测试
  - `use-order-create-options.test.ts`：覆盖错误状态、重试、缺少组织，以及 A 请求晚于 B 返回时不得覆盖 B 状态；
  - `use-order-detail-data.test.ts`：覆盖组织参数与组织切换过期响应保护；
  - `list-resources.test.ts`：覆盖按类别加载、列表与新建页缓存复用及组织切换竞态。
- [x] **步骤 3.3**：新增模板与页面行为测试
  - `OrderFormTemplate.test.tsx`：覆盖分节骨架、`loadingTip`、加载期间不挂载可提交表单；
  - 新建页测试：覆盖关键数据错误状态、“重新加载”操作、重试前缓存失效与 `GENERAL` 默认值；
  - `AvatarDropdown.test.tsx`：覆盖登出成功后全量清理订单会话缓存，登出失败时不清理。
- [x] **步骤 3.4**：运行全量单元测试 `pnpm --dir web test`。

### 阶段 4：质量门禁与网络请求验收

- [x] **步骤 4.1**：类型检查与代码检查
  - 运行 `pnpm --dir web tsc --noEmit`；
  - 运行 `pnpm --dir web biome:lint`。
- [x] **步骤 4.2**：浏览器端真实网络请求验证
  - 在现有 `web/tests/e2e/` 中增加订单加载 Playwright 用例，访问 `/orders/sea-export/new`，断言不发出 airports 请求；同一组织二次进入时，确认主数据、港口、币种及人员候选项的新增请求数均为 0；
  - 执行 `pnpm --dir web test:e2e`。

### 阶段 5：收尾复核与回滚准备

- [x] **步骤 5.1**：确认普通组织切换只依赖组织键隔离和过期响应保护，不全量清空其他组织缓存；登出成功、测试重置或明确全局刷新才调用无参数清理。
- [x] **步骤 5.2**：以 `order-options-cache.ts` 与 `common.ts` 的缓存接入为独立回滚点；若缓存出现回归，可回退调用方到直接请求而不改变后端契约或数据。
- [x] **步骤 5.3**：执行 `git diff --check`，复核只修改订单模块、通用订单模板及对应测试，没有手改生成代码。
