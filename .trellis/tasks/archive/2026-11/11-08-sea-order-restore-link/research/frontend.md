# 前端实现与验证

## 最小实现

- 恢复确认仅对海运出口说明返回原主单、原航程并保留退关记录。
- detail.tsx把currentDocuments（既有编辑单证基线）传给变动检查，不用恢复后的新查询替换原证据。
- acceptTransition闭包绑定操作前保留单证。本人恢复必须满足：TERMINATED→ACTIVE、SE、订单版本精确+1、原MBL ID/版本相符、HOUSE/DIRECT结构一致、新Link版本1、TE身份与版本存在，以及船公司/共同港口/中转港/船名航次/ETD/ETA与原订单保存字段一致。原TE精确身份由后端锁内验证，不增加DTO。
- 恢复不能走普通“关联指纹未变”分支而接纳无活动主单的响应。整包重读若与实际写响应指纹不同立即保留提醒；原草稿、已有外部变化和旧身份回调的保护沿用。
- 不回填表单输入、版本或提交基线；只推进检查参考点。

## 验证

- Vitest：use-order-change-notice.test.tsx、order-detail-transitions.test.tsx、detail-restore-refresh.test.tsx、detail-termination-refresh.test.tsx，4文件58项通过，13.39秒。
- pnpm --dir web tsc通过。
- 修改6个前端文件Biome通过；git diff --check通过。
- 详情恢复测试最初误把终态直接“恢复订单”按钮当作退关中下拉菜单，产生巨大DOM错误格式化；已按真实入口修正，所有调试与临时标记已移除。另修正新夹具的单证枚举使用生成常量。
- 没有执行Git写操作或业务数据库写入。
