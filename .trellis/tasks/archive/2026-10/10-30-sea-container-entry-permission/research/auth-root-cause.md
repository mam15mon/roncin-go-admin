# 共享箱双查询权限错误调查

## 确认的源码根因

`server/internal/server/http.go` 原先先注册 `OrderService`，其中含 `GET /api/v1/orders/{id}`，随后才注册 `SeaSharedContainerService`。后者两个集合静态路径 `/api/v1/orders/sea-shared-containers` 与 `/api/v1/orders/sea-shared-container-candidates` 被先注册的动态订单详情路径匹配。

请求因此被标为 `OrderService/GetOrder` operation，`id` 为 `sea-shared-containers` 或 `sea-shared-container-candidates`。`Authorization → requestOrder` 对它执行 UUID 解析失败并返回 `ErrPermissionDenied`（403），即便操作员具备同公司正确 `business.order.se.container.read` 权限也误拒绝。直接中间件 DTO 测试跳过真实 HTTP 分发，因此原测试无法发现问题。

## 最薄修复

新增 `registerOrderAndSharedContainerHTTPServices`，先注册共享箱 service 再注册通用订单 service；生产使用该注册函数，回归测试复用相同顺序，防止测试与线上顺序脱节。不修改 proto、权限码、角色授权、公司范围、Schema 或生成代码。

## HTTP 权限与绑定矩阵

新增 `server/internal/server/shared_container_route_test.go`，经过真实 Kratos HTTP 路由、生成的 query 绑定、Authorization、RequiredFieldsValidator、Service 与 Usecase，仅仓储边界替身：

- 两个集合 GET：同公司 container.read 成功；camelCase 与 snake_case 均准确绑定 orderId、transportExecutionId、page/pageSize、keyword。
- 仅普通订单读、仅创建、其他品类、其他公司、错误锚点与缺锚点继续拒绝，不进入共享箱仓储。
- 同时保留旧注册顺序的复现分支，合法用户仍得到 403 且捕获 GetOrder operation；修复后必须命中对应共享箱 operation。
- 合法 UUID 订单详情继续分发至 GetOrder，避免修复静态路径后损害详情路由。

## 航程上下文边界

真正的锚点订单与航程关系由 data.ensureSharedAnchorExecution 校验（活动 Link 的 transportExecutionID 必须相同）。HTTP 矩阵证明请求航程 ID 正确传递，不以仓储替身伪称验证该数据关系。现有 `TestSeaSharedContainerAnchorExecutionBinding` 覆盖列表、候选、写入与错误执行拒绝，必须使用专用 PostgreSQL 正式迁移环境运行；未配置时跳过不能算通过。

## 证据限制

未读取用户截图对应 Cookie、权限快照、订单公司及运行二进制版本；不能断言截图来自哪次部署。已确认当前源码具备与双403完全相符的确定性路由缺陷，旧序复现与修复矩阵结果在下方更新。不通过修改用户权限掩盖问题。

## 验证结果

- 新增 HTTP 回归通过：18 个鉴权/绑定子场景（两个集合各 9 场景）+ 1 个正常订单详情分发场景。
- `go -C server test -p 8 ./internal/server -count=1` 通过。
- `go -C server vet ./internal/server` 通过。
- `git diff --check` 通过。
- `go -C server test -p 8 ./internal/data -run ^TestSeaSharedContainerAnchorExecutionBinding$ -v -count=1` 已执行，因未配置专用 `RONCIN_INTEGRATION_DATABASE_SOURCE` 而 SKIP；不计入数据绑定验证通过。
