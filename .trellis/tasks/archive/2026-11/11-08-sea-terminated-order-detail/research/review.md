# 独立 Review 结果

## 结论

Review 通过，未发现剩余阻塞项。全量门禁由主会话另行验收；本记录只陈述独立检查及定向验证。

## 已发现并复核修复的问题

1. 本人状态操作已写库、轮询先于状态刷新观察到新版本时，旧 `state.changed` 保护会永久保留伪外部变动提醒。独立临时测试已复现失败；实施代理增加首次差异快照与其他差异证据，只允许明确匹配本人响应的误报清除。真实外部变更持续保留。
2. 已终止订单首开后本人关账/反关账，表单保留 MBL 与 GetOrder 无活动摘要不一致；原 acceptTransition 比较仍按完整指纹，已统一终态 id/version 口径，补充更晚外部变更回归。
3. 旧订单状态成功回调缺少详情级身份守卫，可能对新订单发起无关刷新或弹旧警告；已复用显式刷新 token 在回调发起与完成处检查，补充切换前与在途迟到用例。
4. 原海运单证规范矩阵只允许 ACTIVE，与终态读取契约冲突；主会话已同步旧矩阵并链接专项规范。

## 其他检查

- 后端 TERMINATED 只按本次 terminated_at、固定退关原因与 ENDED 状态选择关联；组织隔离和既有写门禁未放松，活动统计不计终态。
- MBL 历史入口仅在终态禁用并说明原因；HBL 历史及事件仓储支持保留关系，正常只读订单仍可访问 MBL 历史。
- 未手改生成物、未改公开 API/数据库 Schema、未覆盖既有未跟踪布局任务。
- 本次读取不是退关时刻的共享 MBL 冻结快照，已在规范和设计记录；主单历史接口终态访问扩展不在本期范围。

## 独立验证

- Biome：第一次 9 个修改文件、最终 4 个变更重点文件均通过，无修复或规则豁免。
- `pnpm --dir web tsc`：通过。
- `pnpm --dir web exec vitest run src/pages/orders/use-order-change-notice.test.tsx src/pages/orders/detail-termination-refresh.test.tsx src/pages/orders/templates/components/sea/SeaDocumentSection.termination.test.tsx`：3 文件 38 项通过，总耗时 13.37 秒。
- `git diff --check`：通过。
- 复现竞态的临时测试及配置已删除。
