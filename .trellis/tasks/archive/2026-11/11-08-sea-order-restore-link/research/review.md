# 独立 Review 结果

## 结论

独立review实际完成并通过，未发现剩余阻塞项。主会话仍须等待完整门禁通过，并对用户已授权现场订单先保存预检结果、再显式apply；本review没有执行开发库写入。

## 已发现并复核修复的问题

1. 恢复辅助函数未预载Organization，却用orderToBiz访问Organization.Name，会在恢复正例产生nil panic。已由实施代理改为仅构造运输校验所需字段的biz.Order；DIRECT/HOUSE真实恢复正例经独立复跑通过。
2. 其他活动成员仅因CLOSED而阻断恢复，偏离现有「完结不冻结业务内容」口径。已向主会话报告并获明确裁定：其他成员ACTIVE且未审核锁即可，本票仍要求OPEN；实施代理已修正并补CLOSED未锁成员正例，独立复跑通过。
3. 新测试的锁夹具初次只设LockedAt违反正式锁约束、后续锁版本基线比较错误，共享成员缺必填运输枚举、CLOSED缺完整结案事实。已由实施代理按正式约束修正，没有移除约束或弱化断言；最终完整新套件通过。
4. 补强显式repair负例：角色撤权、同刻多个匹配ENDED关系、恢复后最新终止事件已变化均拒绝；默认预检无业务行/版本/审计变化，apply仅一次新增审计且不伪造状态转移。
5. 普通恢复多个匹配退关关系采用既有409结构冲突，避免把歧义暴露为Ent500。新流程恢复事件保存新Link引用，repair拒绝把这种已有新流程引用的事件当旧异常恢复补链。

## 核对边界

- 完整成员Order按UUID排序先锁定，后MBL、排序Link、TE/HBL；主单锁后重验成员集合和原关系，不补锁新成员。
- 校验同公司、唯一真实退关关系、原MBL/TE/单证结构、DIRECT独占/全员HOUSE、审核锁与运输字段一致性。新Link和恢复订单/生命周期/审计同事务，故障回滚不留部分ACTIVE关系；旧ENDED不改。
- 取消退关/AE原流程保持独立。恢复不改MBL/TE/HBL内容或实体版本，前端原MBL版本与新Link初始版本证据一致。
- 专用修复命令默认只预检，真实启用用户/成员/角色read与transition授权锁内核验；显式目标版本及ENDED关系，严格最新事件链和结束时间，禁止任取历史、自动扫库或ACTIVE→ACTIVE兼容。
- 前端仅精确订单版本+1、原保留MBL身份版本/结构、新Link1及共享运输字段一致才识别本人恢复；保持旧草稿、提交基线、外部更新和迟到身份保护。恢复确认明确原主单/原航程语义。
- 本期未修改Proto/Schema/生成物或扩张权限；独立review未触碰已有布局任务，未执行Git写操作。

## 独立验证

- 6个前端修改文件Biome通过，无修复或规则豁免。
- pnpm --dir web tsc最终通过。
- 前端Vitest：use-order-change-notice、order-detail-transitions、detail-restore-refresh共3文件51项通过，7.05秒。停止仅本review旧详情定位错误编译进程后，针对最终稳定源码重跑；未改产品或测试以掩盖错误。
- 专用PostgreSQL正式迁移隔离Schema：go test ./internal/data -run '^TestSeaOrderRestore' -count=1 -v最终全PASS，56.972秒，无SKIP。覆盖DIRECT/HOUSE、全部阻断、并发恢复、repair授权/证据/预检/单次apply、CLOSED成员及动作投影。
- repair-sea-restore-link命令包go test与go vet通过。
- gofmt -l无输出，git diff --check通过。
- 完整check:fast、现场预检/apply结果由主会话另行记录，不将未执行检查视作通过。
