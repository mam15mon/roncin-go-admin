ALTER TABLE "notification_deliveries"
  DROP CONSTRAINT IF EXISTS "notification_deliveries_template_check";

ALTER TABLE "notification_deliveries"
  ADD CONSTRAINT "notification_deliveries_template_check"
    CHECK ("template" IN ('ORDER_PERSONNEL_ASSIGNED', 'USER_AUTHORIZED'));
