# 空运地点筛选修复与验证

## 实现

- `orderLocationMatches` 按 AE 最新 started_at 关联主单的起运/目的机场构造地点谓词，同详情投影，包含终止后的历史关联；缺关联不回退旧订单字段。
- 主单同时校验目标订单公司；非 AE 继续使用订单地点字段。
- 列表和 count 共享同一谓词，未回写冗余订单列，未改变公共接口。

## 修改文件

- server/internal/data/order_query.go
- server/internal/data/air_order_query.go
- server/internal/data/air_order_location_integration_test.go

## 验证

`python3 /tmp/roncin-system-audit-pg/run-integration.py go -C server test -p 32 ./internal/data -run 'TestAirOrderLocationFiltersFollowSharedMasterPostgres' -count=1` 通过（约 2.7 秒）。临时独立 PostgreSQL 数据库，测试 helper 创建独立 schema，按正式 SQL 迁移初始化，无 Schema.Create。

覆盖：真实主单机场 A→B 修改、起运/目的/组合筛选、同主单两票分页及 count、真实终止状态机结束关联后的筛选、缺关联排除旧字段、海运港口过滤、公司隔离。夹具最初因加票递增主单版本、直接设终止状态未满足正式 CHECK 而失败；已改为读取当前主单版本及调用真实终止状态机，不放宽任何校验。

`gofmt` 与 `git diff --check` 通过。独立 review 与最终全量门禁由主会话继续。
