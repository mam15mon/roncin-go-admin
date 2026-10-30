# 地点引用误校验证据

## 用户场景

用户更正为刷新订单后不改任何内容直接保存也失败，并提供实际更新 JSON。起运港与目的港使用同一 ID；无需假设额外编辑或版本冲突。

## 确认因果链

1. `server/internal/data/order_sync.go:76` 调用 `nonNilUUIDs` 收集地点引用。
2. `server/internal/data/order_convert.go:352` 只去掉 nil，保留重复 UUID。
3. `validatePortIDs` 用 `WHERE id IN (...) AND enabled = true` 查询数量，重复 ID 只会匹配一条记录。
4. `order_sync.go:112` 与原始引用数量比较，重复引用被误判缺失。
5. 通过本地 `.env.local` 注入连接、强制 `default_transaction_read_only=on` 运行最小统计，结果：请求地点引用 2、唯一 ID 1、启用港口匹配 1、订单存在。未输出连接配置，未修改数据库。

## 相邻路径

- `validateAirportIDs` 用相同计数结构，必须同步验证；这两条校验产品调用仅来自 `validateOrderReferences`。
- `nonNilUUIDs` 的产品调用仅地点收集，可在该边界按集合处理，也可在具体校验入口去重；选择最小、语义明确的修复，不对 cargoCategory 等业务禁止重复的列表放宽规则。
- 现有 `shared_location_order_integration_test.go` 读路径测试允许重复中转 ID，但写校验只传单个 ID，未覆盖该缺陷。
- 仓库未查到起运港/目的港必须不同的规则；本次不可新增这类业务限制掩盖程序错误。
- 后续完整空运流程验证发现既有 `biz/air_export.go:222` 的 `NormalizeAirMasterInput`
  明确要求起运机场与目的机场不同；保留此规则。机场集合校验的四字段同 ID 用例直接验证
  validator；AE 完整保存使用合法不同起降机场并重复其他地点字段，不能误放宽业务限制。

## 验证要求

- 禁止 TDD：先实现最小修复，再补有针对性的回归。
- 单测验证引用收集/唯一集合且不改原始字段；真实 PostgreSQL 正式迁移集成验证重复 ID、不同 ID、混入缺失/停用 ID及无修改更新。
- PostgreSQL 集成使用专用 `RONCIN_INTEGRATION_DATABASE_SOURCE` 和隔离 Schema；不得把正常开发数据库当作可重置库，不运行 Schema.Create。
- 如果缺少专用测试环境，建立单独可删除测试数据库或记录阻塞，不跳过后却声称验证通过。
- 错误提示限港口/机场的真实失效场景，沿用错误 reason；不能把所有 ORDER_INVALID_ARGUMENT 一律改成港口错误。
