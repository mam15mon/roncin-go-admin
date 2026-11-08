# 后端实施与验证

## 实施边界

- 新增sea_order_restore.go：按完整成员Order排序锁→MBL→排序Link→原TE/HBL，锁后复核成员集合、原ENDED身份与版本，检查主单有效、原航程运输字段一致及单证结构。
- 普通SE恢复在统一事务创建新ACTIVE Link、订单version+1、清退关字段、生命周期事件和审计；历史Link及MBL/TE/HBL正文与版本不变。事件reference_id记录新Link，审计包含新旧Link。DIRECT独占、HOUSE批次约束不放宽。
- SE本票审核锁阻断恢复；其他成员须ACTIVE且未锁，已完结但未审核锁的成员按既有共享资料规则允许。本票仍须OPEN。锁定终态不投影可恢复动作，TERMINATING取消退关不误禁。
- 一次性cmd/repair-sea-restore-link默认仅预检业务事实，不写订单/审计/关系。显式apply要求真实同公司read/transition授权并锁授权父行，核对旧恢复事件无新流程reference、紧邻完成退关事件与唯一ENDED时间配对、无后续运输关联；不制造ACTIVE→ACTIVE事件。修复审计含事件和原/新Link引用。
- 未修改开发数据库、未执行apply、未执行Git写操作。

## 验证结果

- 专用PostgreSQL（仅注入RONCIN_INTEGRATION_DATABASE_SOURCE，正式迁移隔离Schema）完整命令：
  `go -C server test ./internal/data ./cmd/repair-sea-restore-link -run 'TestSeaOrderRestore|TestSeaDocumentTerminationPostgres|TestSeaDocumentTerminatedHistoryPostgres|TestOrderLifecycleGatePostgres|TestRepairInputRequiresExplicitTarget' -count=1 -v`
  最新源码全部PASS，无SKIP。data65.030秒、CLI0.051秒；日志/tmp/sea-restore-backend-tests.log。
- 新DIRECT/HOUSE真实退关→恢复→重复单证重读、原MBL/TE身份版本保持、旧Link不变、新Link初版、正常恢复审计与事件引用；历史版本列表读取成功。
- 阻断：MBL作废、HBL作废、TE资料冲突、本票/共享成员审核锁、本次关系缺失/歧义、DIRECT主单被占用、审计故障整体回滚。
- 并发恢复恰好一成功；旧版本拒绝；共享CLOSED未锁成员正例通过。
- 修复命令：无权、撤权、当前版本错误、同刻关系歧义、最新事件链改变、事件时间不匹配均拒绝；默认预检无业务写，apply仅一次、恢复详情可读、独立审计、生命周期事件数不增加。
- 旧生命周期和第二轮退关测试同步为旧ENDED不变+新ACTIVE建立；第二轮新Link并发版本设7，结束后必须读取新版本8而非旧版本2，保留防借旧关系保护。
- 空运相邻真实库：`TestAirExportTerminationRestoreAndConcurrentIdentityPostgres` PASS 4.274秒，日志/tmp/sea-restore-air-regression.log。
- CLI vet通过，gofmt及git diff --check通过。

## 首轮问题与修正

独立review捕获完整orderToBiz转换依赖Organization预载造成nil panic，已改为仅构造运输比较字段，真实恢复正例通过。新增测试先后遇到未配齐审核锁/成员必填/完结一致性字段的夹具错误，均按正式约束补齐，不删除约束或改业务规则。最终全套在完整夹具下通过。

## 现场预检入口

在仓库根目录用现有环境加载方式执行：

`node --env-file-if-exists=.env.local scripts/run-with-env.mjs go -C server run ./cmd/repair-sea-restore-link --organization-id 01a0c1b1-a30b-7747-8fae-2dd785dfdac3 --order-id 01a1bbd3-18fe-73e5-b185-9c6b5ae7f788 --expected-version 7 --ended-link-id 01a1bbd3-1906-7d3c-b37f-212871a37440 --actor-id 01a13400-0da3-76a9-a83b-509e8fcf41af --reason '修复用户授权测试订单恢复后缺失原运输关联'`

缺省不写入。主会话须在独立review与预检证据通过后，按既有用户授权才加`--apply`；当前版本若变化须停止重核，不自动刷新期望版本。
