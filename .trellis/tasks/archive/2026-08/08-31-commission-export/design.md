# 提成导出设计

## 分层职责

- Proto：定义 Export RPC、HTTP GET 路径、权限和导出 DTO。
- Service：转换筛选条件和导出响应，不生成 CSV。
- Biz：校验筛选、执行总数门禁、分批读取、构造成功审计。
- Data：复用列表谓词、稳定排序和动态 CNY 转换；保存成功导出审计。
- Access：登记独立权限并生成服务端规则与前端权限键。

## 契约

`ExportCommissionsRequest` 包含 keyword、status、commission_date_from/to，无分页。
`ExportCommissionsResponse` 返回 `CommissionExportItem` 列表。导出 DTO 使用明确的
扁平字段，不复用包含 lines/adjustments 的详情对象，避免无关数据和响应膨胀。

## 查询流程

```text
校验筛选
  → Count(同一谓词)
  → total > 10000：返回明确业务错误，不执行数据查询
  → total <= 10000：每批最多 200，按稳定排序读取
  → 组装扁平 DTO
  → 写 finance.commission.export 成功审计
  → 审计成功后返回响应
```

数据层由同一个私有 predicate 构造函数服务 List、Count 和 Export batch。批量查询
使用 offset/limit 和唯一 ID 排序；第一版不扩展事务隔离级别。

## 权限与审计

- Manifest：`system.finance.commission.export`，Requires 包含 read。
- Proto RPC：permission mode、organization scope。
- 运行访问规则生成和 `generate:permission-keys`。
- 成功审计只记录 actor、organization、规范化筛选摘要、最终行数和动作名称。
- 业务审计写入失败即接口失败，避免数据成功返回却没有审计。
- 所有失败类型继续走已有请求日志和权限中间件，不新增失败审计事务。

## 错误与验证

- 超限使用明确的提成导出上限领域错误，提示缩小筛选范围。
- 日期、状态、keyword 校验沿用列表规则。
- 查询或审计错误不截断、不返回部分数据。
- 验证 0、1、200、201、10000、10001 行边界，列表/导出谓词与排序一致。
