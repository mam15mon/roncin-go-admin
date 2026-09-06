# 技术方案：修复订单缓存复用与组织切换隔离

## 1. 边界与数据流

本次只修改订单前端 Hook 及其测试，不变更 API 契约。

```text
当前组织/业务配置
  ├─ 首批缓存数据 ──> Hook 中的有效 locationOptions
  │                     └─ 空关键字搜索直接返回
  ├─ 非空关键字 ──> 服务端联想请求 ──> 校验请求组织仍为当前组织 ──> 返回或丢弃
  └─ 加载失败 ──> 记录失败请求身份 ──> 与当前页面身份匹配后才展示
```

## 2. 空关键字地点搜索

- 在新建和详情数据 Hook 中处理地点搜索，因为这里同时拥有当前组织、加载完成的地点候选项和活动组织引用。
- 关键字使用 `trim()` 判断：空值或仅空白时返回当前有效 `locationOptions`；非空时保持原值传给服务端，避免无关改变搜索语义。
- 空关键字返回值必须使用按 `loadedOrganizationId`（详情还包括 `loadedOrderId`）校验后的有效数据，不能直接读取未分区的底层 state。
- 非空远程请求发起前捕获组织 ID，完成后比较 `activeOrgIdRef.current`；组织已变化则返回 `[]`。

## 3. 列表远程搜索隔离

- 为列表 Hook 内的搜索统一使用“捕获组织 → 请求 → 校验组织 → 更新/返回”的结构。
- 当前组织缺失时直接返回 `[]`。
- 客户、港口、地点、承运人、人员全部覆盖；现有人员搜索的返回保护作为统一行为基准。
- 客户映射和港口列表只在组织仍匹配时更新；过期响应既不更新 state，也不向调用方返回候选项。
- 不新增全局缓存或通用状态库，避免把页面局部的活动组织引用扩展成跨模块状态。

## 4. 错误身份

- 新建 Hook 把错误保存为带请求键的结构，键包含 `organizationId`、`category` 和 `businessType`。
- 详情 Hook 的错误键包含 `organizationId` 与 `orderId`。
- 渲染派生值只暴露与当前键完全一致的错误；旧错误可以暂时留在底层 state，但在页面身份变化的同一次 render 中立即失效。
- “缺少当前组织”继续作为由当前用户状态同步派生的独立错误，不依赖异步失败 state。

## 5. 兼容性与回滚

- 有关键字的远程联想、候选项结构和调用方接口保持不变。
- 空关键字从远程查询改为返回已加载首批数据，这是消除重复请求所需的行为修正。
- 若出现回归，可按 Hook 文件独立回滚；不涉及生成物或服务端部署顺序。

## 6. 预计修改文件

- `web/src/pages/orders/use-order-create-options.ts`：地点空查询复用、远程结果组织校验、错误身份隔离。
- `web/src/pages/orders/use-order-create-options.test.ts`：新建 Hook 边界测试。
- `web/src/pages/orders/use-order-detail-data.ts`：地点查询和详情错误身份隔离。
- `web/src/pages/orders/use-order-detail-data.test.ts`：详情组织失败切换测试。
- `web/src/pages/orders/list-resources.ts`：所有列表候选搜索的返回值组织隔离。
- `web/src/pages/orders/list-resources.test.ts`：搜索 Promise 与空组织测试。
- `web/tests/e2e/order-create-loading.e2e.ts`：原则上保留严格断言；仅在需要提升等待稳定性且不改变验收语义时调整。

## 7. 明确不做

- 不改订单模板字段数量或 UI 布局。
- 不给关键字远程查询增加新的会话缓存。
- 不处理本次 Review 之外的订单列表行数据、标签或其他模块组织切换问题。
