# Research: 后端后24份规范逐份补查

- Query：阅读全文后24份后端规范，对照关键源码，记录实际核查深度。
- Scope：internal；只写当前research，不改现行规范、代码、配置或数据，不作Git写操作。
- Date：2026-11-10。

## Findings

已独立阅读全文14份，其中13份未发现确证现行错误、1份待修正；10份按主会话规划收敛要求明确待确认，实施阶段继续。逐文件记录见 [backend-extra-ledger.jsonl](backend-extra-ledger.jsonl)，没有将前轮报告自动当成本轮通过。

### 已确认遗漏

`.trellis/spec/server/backend/order-lock-and-document-version.md` 第5节Base仍写「恢复订单但不复活历史Link，后续显式建立新Link」，与同篇第1/3节冲突。实际 `server/internal/data/order_write_lifecycle.go:108-115` 在恢复同事务调用 `createSeaOrderRestoreLink`，实现为 `server/internal/data/sea_order_restore.go:128`。建议改为同事务创建原MBL/TE活动Link，保留旧Link不可变。

同篇有效lock grant谓词所有者旧定位为order_lock.go，实际qualifiedBusinessLockGrantPredicate位于 `server/internal/data/order_lock_qualification.go`，应修定位，不改变资格和并发规则。

## 已读文件与对照重点

- order-attachment-oss：附件Proto Prepare/Register/Access/Remove，biz上限100<<20，data/order_attachment_storage。
- order-fee-correction：审批/撤回能力、非bootstrap真实grant、逐源费待审。
- order-fee-quantity-preview：order_fee.proto:866-900，显式PLANNED/ACTUAL与确认计费重，无自动回退。
- order-fee-review：data/order_fee_review.go:514-548的逐行能力、order_lock_apply。
- order-lock-and-document-version：恢复Link和资格谓词，发现上列遗漏。
- order-milestone-retirement：20261030140000严格DROP TABLE，Manifest已无里程碑权限，删除清单保留历史含义。
- order-personnel-editor-retirement：order_personnel.proto仅List，保留人员实体与读权限。
- order-shared-enum-field：schema optional/nillable包装类别，order_write.go:300-301省略保留。
- organization-shared-masterdata：RequireGlobalMasterDataWrite，公共港口船代，sync-unlocode联合中文摘要。
- partner-casual-contract：partner字段编号与错误码12匹配，finance_bill_batch正式/散客账期分别判定。
- partner-role-blacklist：Proto签名、data/partner.go:86-98黑名单启停正交与biz角色审计。
- passkey-authentication：全部路由，五分钟挑战，用户共享锁→凭证写锁及正式迁移。
- retire-role-data-scope：正式迁移停用self再删列，能力推导与角色配置分离。
- role-workspace-ownership：admin_role.go:24/38，auth.go:155/161，前端isWorkspaceKindValue。

## 尚待独立全文核对

- `.trellis/spec/server/backend/quality-guidelines.md`
- `.trellis/spec/server/backend/sea-document-change-history.md`
- `.trellis/spec/server/backend/sea-export-document-contract.md`
- `.trellis/spec/server/backend/sea-hbl-production.md`
- `.trellis/spec/server/backend/sea-order-restore.md`
- `.trellis/spec/server/backend/sea-shared-write-consistency.md`
- `.trellis/spec/server/backend/sea-terminated-order-read.md`
- `.trellis/spec/server/backend/user-membership-batch.md`
- `.trellis/spec/server/backend/wecom-retirement.md`
- `.trellis/spec/server/backend/workbench-todos.md`

## External References

没有访问外网，依据当前仓库Proto、Schema、迁移和实现。

## Caveats / Not Found

- 已读14份只证明全文阅读与关键入口对照，不代表每条业务矩阵实跑。
- 未执行真实数据库、OSS、钉钉、浏览器、生成器或业务测试，不补造历史验收结果。
- 尚待10份不能标为本轮已核实最新，实施阶段需继续。
