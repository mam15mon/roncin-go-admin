-- 通知模板枚举扩展：汇率周报提醒 / 费用补录待审批 / 提成冲减建议知情。
-- 20260915093000 起 Ent 枚举新增模板但未同步扩展本 CHECK，
-- 冷启动库按迁移链初始化后写入新模板会违反约束，这里一次性补齐，
-- 并为 notification_deliveries.template 补上 Ent Schema 同源 CHECK 注解。
ALTER TABLE "notification_deliveries"
  DROP CONSTRAINT "notification_deliveries_template_check";

ALTER TABLE "notification_deliveries"
  ADD CONSTRAINT "notification_deliveries_template_check"
    CHECK ("template" IN ('ORDER_PERSONNEL_ASSIGNED', 'USER_AUTHORIZED', 'DINGTALK_REGISTRATION_PENDING', 'DINGTALK_REGISTRATION_REJECTED', 'DINGTALK_INVITATION_ACTIVATED', 'EXCHANGE_RATE_WEEKLY_REMINDER', 'FEE_SUPPLEMENT_APPROVAL_PENDING', 'COMMISSION_DECREASE_SUGGESTED'));
