# 技术方案设计：优化新建订单页面主数据加载体验

## 1. 架构定位与改动边界

```text
web/src/pages/orders/
  ├── common.ts                   # 消费共享缓存，统一按需加载并派生 fetchOrderMasterData 结果
  ├── use-order-create-options.ts # 传递 orgId 与 category，导出 error/retry，丢弃过期响应
  ├── use-order-detail-data.ts   # 传递 orgId 与 category，复用缓存并扩展过期响应保护
  ├── list-resources.ts          # 复用组织级缓存、按需加载并丢弃过期响应
  ├── new.tsx                    # 错误状态阻断渲染与重试入口，GENERAL 业务码默认值对齐
  └── orders.test.ts             # 增加针对性缓存隔离、按需加载、失败释放单测
web/src/components/ui/order-template/
  └── OrderFormTemplate.tsx       # 分节骨架屏占位优化
web/src/components/RightContent/
  └── AvatarDropdown.tsx          # 退出登录成功后清理订单会话缓存
web/src/utils/
  └── order-options-cache.ts      # 组织隔离的 Promise 缓存、定向失效与全量清理入口
```

## 2. 核心改动点设计

### 2.1 组织隔离的模块级 Promise 缓存与失败自动释放 (`order-options-cache.ts`)

- **数据结构与作用域**：
  ```ts
  const masterOptionsCache = new Map<string, Promise<API.MasterDataItem[]>>();
  const portsCache = new Map<string, Promise<API.Port[]>>();
  const airportsCache = new Map<string, Promise<API.Airport[]>>();
  const personnelOptionsCache = new Map<string, Promise<API.OrderPersonnelOption[]>>();
  ```
- **单例获取、并发共享与失败清理模板**：
  ```ts
  export function getMasterDataOptions(organizationId: string): Promise<API.MasterDataItem[]> {
    if (!organizationId) {
      return Promise.reject(new Error('缺少当前组织，无法加载订单主数据'));
    }
    let req = masterOptionsCache.get(organizationId);
    if (!req) {
      req = masterDataServiceListOptions()
        .then(unwrapList)
        .catch((err) => {
          masterOptionsCache.delete(organizationId);
          throw err;
        });
      masterOptionsCache.set(organizationId, req);
    }
    return req;
  }
  ```
- **人员候选项缓存**：
  键名为 `${organizationId}:${businessType}`：
  ```ts
  export function getOrderPersonnelOptions(
    organizationId: string,
    businessType: number,
  ): Promise<API.OrderPersonnelOption[]> {
    if (!organizationId) {
      return Promise.reject(new Error('缺少当前组织，无法加载订单人员选项'));
    }
    const key = `${organizationId}:${businessType}`;
    let req = personnelOptionsCache.get(key);
    if (!req) {
      req = orderServiceListPersonnelOptions({ businessType, page: 1, pageSize: 200 })
        .then(unwrapList)
        .catch((err) => {
          personnelOptionsCache.delete(key);
          throw err;
        });
      personnelOptionsCache.set(key, req);
    }
    return req;
  }
  ```
- **按组织失效与全量清理入口**：
  ```ts
  export function clearOrderMasterDataCache(organizationId?: string) {
    if (!organizationId) {
      masterOptionsCache.clear();
      portsCache.clear();
      airportsCache.clear();
      personnelOptionsCache.clear();
      return;
    }

    masterOptionsCache.delete(organizationId);
    portsCache.delete(organizationId);
    airportsCache.delete(organizationId);
    for (const key of personnelOptionsCache.keys()) {
      if (key.startsWith(`${organizationId}:`)) {
        personnelOptionsCache.delete(key);
      }
    }
  }
  ```
- **生命周期约定**：
  - 普通组织切换不清空其他组织缓存；首次访问组织 B 时因 key 不同自然发起 B 的请求，切回组织 A 时可复用 A 的缓存；
  - 新建页主动重试前调用 `clearOrderMasterDataCache(organizationId)`，使成功响应后的业务校验错误也能重新获取；
  - 退出登录成功后、清理当前用户状态和跳转登录页之前调用无参数清理，防止模块缓存跨登录会话残留；无参数清理也用于测试或明确的全局刷新；
  - 币种来自全局参考数据，继续复用 `getCurrencies()` 的全局 Promise 缓存，不参与组织级清理。
- **`common.ts` 按需加载重构**：缓存模块只负责获取原始数据；订单候选项派生继续留在页面公共逻辑中：
  ```ts
  export async function fetchOrderMasterData(
    organizationId: string,
    category?: 'sea' | 'air',
  ) {
    const shouldLoadPorts = !category || category === 'sea';
    const shouldLoadAirports = !category || category === 'air';

    const [masterOptions, ports, airports, currencies] = await Promise.all([
      getMasterDataOptions(organizationId),
      shouldLoadPorts ? getCachedPorts(organizationId) : Promise.resolve([]),
      shouldLoadAirports ? getCachedAirports(organizationId) : Promise.resolve([]),
      getCurrencies(),
    ]);
    ...
  }
  ```

### 2.2 订单列表、新建与详情全链路接入

- **`list-resources.ts`**：
  - 将原本无条件的 3 个全量接口调用改为调用 `getMasterDataOptions(organizationId)` 与按 `config?.category` 按需拉取港口或机场，使列表页与新建页之间无缝复用缓存；
  - 将 `organizationId` 纳入 Effect 依赖，使用请求序号或 Effect cleanup 丢弃组织切换后晚到的旧响应。
- **`use-order-create-options.ts`**：
  - 从 `useModel('@@initialState')` 取得 `currentUser?.currentOrganization?.id`；
  - 导出 `{ loading, error, retry, ... }`；
  - 当前组织缺失时不调用组织级接口，设置明确错误；
  - 发生错误时设置 `error`；`retry()` 先调用 `clearOrderMasterDataCache(orgId)`，再重新执行加载；
  - 获取人员选项走 `getOrderPersonnelOptions(orgId, config.businessType)`；
  - 使用递增请求序号并记录当前组织 ID，仅允许最新请求更新 React 状态。
- **`use-order-detail-data.ts`**：
  - 从当前用户状态取得组织 ID 并传给 `fetchOrderMasterData`；
  - 在现有请求序号保护中增加当前组织 ID 校验，防止同一订单路径下切换组织时旧响应写回。
- **`AvatarDropdown.tsx`**：
  - 从共享工具模块导入清理入口，避免全局布局组件依赖 `pages/orders`；
  - `authServiceLogout` 成功后调用无参数 `clearOrderMasterDataCache()`；
  - 随后保持现有顺序清除 `currentUser` 并跳转登录页；登出接口失败时不提前清理当前有效会话缓存。
- **`new.tsx`**：
  - 增加错误分支拦截：当 `error` 存在且无基础数据时，展示 `<Result status="warning" title="主数据加载失败" extra={<Button onClick={retry}>重新加载</Button>} />`，坚决不渲染可提交的空表单；
  - 默认货物类别匹配：
    ```ts
    const defaultCargoCategoryId =
      cargoCategoryOptions.find((item) => item.code === 'GENERAL')?.value ??
      cargoCategoryOptions.find((item) => item.label === '普货')?.value;
    ```

### 2.3 分节骨架屏占位优化 (`OrderFormTemplate.tsx`)

- 在 `loading=true` 时：
  - 顶部正常渲染 `header`（吸顶栏）；
  - 主体区域渲染两张纯白高密度 `SectionCard` 骨架：
    - SectionCard 1 标题为“业务基本信息”，内部包含带有微圆角的 `Skeleton active paragraph={{ rows: 3 }}`；
    - SectionCard 2 标题为“运输与订舱信息”，内部包含 `Skeleton active paragraph={{ rows: 4 }}`；
    - 在右上角或卡片顶部伴随优雅的 `loadingTip` 轻量文本与微型 Spin 提示；
  - 保持整体视觉节奏与真实表单高度一致，彻底告别单调居中大卡片。

### 2.4 组织切换竞态与错误恢复数据流

```text
组织 A 发起请求（requestId=1）
        │
        ├── 用户切换到组织 B
        │       └── B 发起请求（requestId=2）
        │               └── 仅 requestId=2 且 organizationId=B 可写入状态
        │
        └── A 晚到（requestId=1）→ 判定过期并丢弃，不更新页面
```

- 缓存 Map 负责隔离和复用不同组织的请求结果；请求序号负责阻止旧异步回调污染当前页面，两者缺一不可；
- API Promise reject 时由缓存函数自动删除对应 key；
- API 成功但必需业务码校验失败时，页面进入错误状态，用户重试会主动失效当前组织缓存后重新获取；
- 缺少当前组织时不把空数组当成成功数据，不渲染可提交表单。

## 3. 测试与验证策略

1. **缓存与候选项单元测试**：
   - `order-options-cache.test.ts`：覆盖同组织并发与顺序调用去重、请求失败释放、不同组织隔离、按组织定向失效和无参数全量清理；
   - `orders.test.ts`：覆盖 sea 模式不调用机场、air 模式不调用港口、未传 category 时双加载，以及 GENERAL 稳定业务码默认值。
2. **Hook 与页面组件测试**：
   - `use-order-create-options.test.ts`：覆盖 loading/error/retry、缺少组织不发请求，以及 A 请求晚于 B 返回时不得覆盖 B 状态；
   - `use-order-detail-data.test.ts`：覆盖组织参数接入及组织切换后的过期响应丢弃；
   - `list-resources.test.ts`：覆盖按类别加载、与新建页共享缓存及组织切换竞态；
   - `OrderFormTemplate.test.tsx`：覆盖分节骨架、`loadingTip`、加载期间无可提交表单及加载完成后的正常渲染；
   - 新建页测试：覆盖主数据错误状态、“重新加载”操作、重试前当前组织缓存失效及 `GENERAL` 默认值；
   - `AvatarDropdown.test.tsx`：覆盖退出登录成功后清理全部订单会话缓存，并确认登出失败时不提前清理。
3. **全量套件验证**：
   - `pnpm --dir web test`
   - `pnpm --dir web tsc --noEmit`
   - `pnpm --dir web biome:lint`
4. **真实交互与网络请求验证**：
   - 在现有 `web/tests/e2e/` Playwright 基础设施中增加订单加载验收用例，无头浏览器访问海运新建页并监听网络请求，确认无 airports 请求；同一组织连续进入两次，确认第二次主数据、港口、币种及人员候选项的新增请求数均为 0；通过 `pnpm --dir web test:e2e` 执行。
