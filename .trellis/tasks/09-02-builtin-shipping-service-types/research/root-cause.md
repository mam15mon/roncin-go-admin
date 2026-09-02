# 根因调查：海运服务类型误报缺失

## 复现现象

进入海运出口订单并点击“新建订单”，页面提示：

```text
缺少海运服务类型主数据：订舱（BOOKING）
```

## 后端初始化证据

- `server/internal/biz/masterdata.go` 的 `DefaultOrderOptions()` 内置 19 个
  `service_type`，第一项为 `BOOKING / 订舱`，来源为 `system`。
- `server/cmd/migrate/main.go` 在正式迁移 post-step 调用
  `SyncDefaultOrderOptions`，为所有已有组织幂等补齐缺项。
- `server/cmd/bootstrap-admin/main.go` 和
  `server/internal/data/admin_organization.go` 在新建首个组织或管理端新建组织时调用
  `CreateDefaultOrderOptions`。
- `server/internal/data/default_order_options_sync_integration_test.go` 已验证首次补齐、
  缺项重建、重复幂等及不覆盖管理员修改。

## 当前开发库只读核对

2026-09-02 对 `.env.local` 指向的开发库只读查询：

- 组织数量：3。
- 每个组织均有 19 个 `service_type`。
- 每个组织均有且仅有 1 个 `BOOKING`。
- `BOOKING` 的 `source=system`、`enabled=true`。

因此用户不需要手工 seed，当前报错不能由数据库缺项解释。

## 前端契约根因

数据流：

```text
MasterDataItem.kind（Proto 枚举）
  → HTTP/OpenAPI 数字 8
  → 生成类型 API.MasterDataItem.kind?: number
  → fetchOrderMasterData
  → isMasterDataKind(8, "MASTER_DATA_KIND_SERVICE_TYPE")
  → false
  → serviceTypeOptions=[]
  → requireSeaServiceTypeOptions 报 BOOKING 缺失
```

直接证据：

- `web/src/services/roncin/typings.d.ts` 声明 `MasterDataItem.kind?: number`。
- `web/src/enums.generated.ts` 声明
  `MasterDataKind.MASTER_DATA_KIND_SERVICE_TYPE === 8`。
- `web/src/pages/orders/common.ts` 的 `MASTER_DATA_KINDS` 却保存字符串枚举名，
  `isMasterDataKind` 只执行严格相等比较。
- `web/src/pages/orders/orders.test.ts` 使用字符串输入断言成功，与真实生成类型不一致，
  因而形成假保护。

## 最小修复边界

- 订单公共代码直接消费生成的 `MasterDataKind` 数字常量。
- 将测试输入改为真实数字枚举，并覆盖候选构建与缺项失败。
- 后端初始化、数据库记录和生成文件均无需修改。
