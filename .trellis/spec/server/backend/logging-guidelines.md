# 日志与审计规范

## 运行日志

- 结构化 JSON 输出（平台能力见 `internal/platform/`）。
- 记录足以定位问题的上下文（组织 ID、业务单号、错误原因），但不倾倒报文。

## 审计日志

- 业务审计独立存储，与运行日志分开。
- 敏感操作（确认、支付、取消、导出等）记录操作人、组织、关键参数与结果。
- 范例：提成导出审计记录操作人、组织、筛选条件、导出行数与成功/失败，但不
  保存完整导出内容。

### 提成导出审计边界

- Service 先验证身份、导出权限及当前公司范围；上游拒绝不伪造操作者或组织。
  用例取得有效身份和组织集合后，筛选无效、计数失败、超限、批读失败、请求
  取消及成功各收口为一次 `finance.commission.export` 审计。
- 成功 `row_count` 为实际导出行数，失败为 `0`；阶段和读取行数只能用固定
  类别与整数描述，不保存原始错误、SQL 或部分导出内容。
- 关键字只记录是否存在和字符数；状态只记录合法枚举，日期只记录合法规范值，
  非法输入用标记描述，不能原样写入审计。
- 普通只读导出使用 `context.WithoutCancel` 保留请求元数据，再增加独立的
  5 秒超时同步保存审计；不重试、不投递后台队列。该方法保留 context value，
  **不代表移除事务身份**。
- `SaveExportAudit` 明确拒绝活动或已关闭的共享事务上下文；普通上下文仍经
  `Data.client(ctx)` 取得客户端，以单条 INSERT 自动提交。不支持共享事务内
  导出，不得声称审计在业务事务内写入后还能独立于回滚保留。
- Count 前、批次之间和全部读取后检查取消；审计失败不交付数据，双失败的
  错误安全规则见[错误处理](./error-handling.md)。审计写入成功不等于网络交付
  已被客户端确认。

回归入口：`TestCommissionExportFailureAudits`、
`TestCommissionExportCancellationStillAudits`、
`TestCommissionRepoSaveExportAuditRejectsSharedTransaction` 及
`TestCommissionRepoSaveExportAuditAfterCancellationKeepsMetadata`；权限与当前公司
拒绝由 `TestExportCommissionsRejectsIdentityPermissionAndCompanyBeforeUsecase` 验证。

## 禁止记录的内容

密码、Cookie、令牌、完整敏感请求/响应报文。提成等涉及员工收入的数据，导出
比普通查询更敏感，必须有独立权限与审计。

账号登录失败审计只保留域分隔 account_hash，不记录原始输入；手机号等价格式共用失败限流桶。成功审计不记录完整手机号，不能沿用原 username 原文路径。
密码找回的验证码与 challengeToken 原文不得进入日志、审计、URL、后台任务或通用
通知正文存储，规则见 [手机号登录与密码找回](./mobile-login-password-reset.md)。
