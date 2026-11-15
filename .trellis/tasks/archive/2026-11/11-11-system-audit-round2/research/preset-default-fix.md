# B7 禁用默认预设修复与验收

2026-11-11 用户在动态确认后明确要求修复，沿用当前审计任务实施，工作分支
`codex/preset-default-validation`，基线 main `67b25ad0`。

## 行为变化

创建和更新共用的 `normalizePartnerShippingPreset` 对 `IsDefault && !Enabled` 返回已有
`ErrPartnerShippingPresetInvalidArgument`（HTTP 400），请求在仓储调用之前结束，不能清除
原默认。拒绝矛盾请求，不静默纠正；正常启用默认及禁用非默认仍允许。

本次只修改 Biz 共用校验及针对性验证，不改 Proto、Schema、迁移、前端或独立文本 SQL 问题。
不清理已有数据。业务库未被写入，正式迁移集成验证使用单独 PostgreSQL 临时集群和随机 Schema。

## 验收

- 实现后补 Biz Party/Text × Create/Update 四种非法组合，断言原错误并且未调用仓储。
- Service→Biz→Data→正式迁移 PostgreSQL 验证两条原触发路径拒绝，响应无成功结果，原默认、
  目标资源及全部关联快照不变，成功审计不增加；对照正常启用默认创建/更新及禁用非默认合法。
- 定向命令：`go -C server test ./internal/biz ./internal/service ./internal/data -run TestPartnerShippingPreset -count=1 -v`，通过仓库外专用数据库包装器执行。Biz 三个顶层测试及 Party/Text×两入口验证通过；data 五场景真实 PASS，无 SKIP，1.452s。service 该筛选无匹配测试，只作编译；实际 Service 转换已在 data 集成测试执行。
- Go vet（biz/data）、gofmt、git diff --check 通过。独立 trellis-check review 通过，无遗留问题；`pnpm run check:server` 完整通过，包含架构、Proto、Go 测试、vet 与漏洞扫描。普通门禁不含隔离库，真实数据库结果以上述定向 PASS 为准。
- 当前不运行前端测试；未改前端、契约或构建入口，不重复前端全量检查或构建。

修复前动态复现保持在 [动态确认报告](preset-default-dynamic-confirmation.md)，不改写其历史结果。

## 提交与交付

产品及规范提交 `7e36d4e0`，独立测试集群已停止，隔离 Schema 清理后计数为 0。通过全部校验后按已授权流程快进合并 main、安全删除本次分支并普通 push；不归档持续审计任务。
