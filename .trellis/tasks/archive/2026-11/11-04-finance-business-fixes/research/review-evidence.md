# 前期审查证据

基线 main@f80b6726；九项问题及定位见父任务 PRD。R2 使用正式迁移的隔离 PostgreSQL 实际调用账单 Cancel 后证明当前理论提成为40、支付预览仍为30；R3/R4/R5/R6 通过临时定向测试证明，其他项目完成调用链核对。审查阶段未修改产品代码。

## 本次验证环境
新建一次性 PostgreSQL 实例，测试仅通过 RONCIN_INTEGRATION_DATABASE_SOURCE 指向专用验收库，各测试创建独立 Schema 并按正式 SQL 迁移初始化。数据库地址与进程信息只保存于仓库外临时环境文件。不得连接开发业务库、调用运行期 Schema.Create 或把 SKIP 算作通过。

## 跨组边界
- bill 组扩展 order_fee_review.go 的 lockBillFeeOrders / lockExistingBillOrders 获取提成订单闭包；commission 组不改这两个函数。
- rate 组负责 cashflow 页面 access 选择；不改变服务端权限码。
- 父任务负责规范、整体门禁、独立 review 与提交。
