-- 订单状态从数据库模板切换为代码内 SE 状态机，并拆分终止与结案维度。
ALTER TABLE "orders" RENAME COLUMN "status" TO "flow_status";

ALTER TABLE "orders"
  ADD COLUMN "termination_status" varchar(24) NOT NULL DEFAULT 'ACTIVE',
  ADD COLUMN "termination_type" varchar(32),
  ADD COLUMN "termination_reason" varchar(500),
  ADD COLUMN "terminated_at" timestamptz,
  ADD COLUMN "terminated_by" uuid,
  ADD COLUMN "closure_status" varchar(16) NOT NULL DEFAULT 'OPEN',
  ADD COLUMN "closure_reason" varchar(500),
  ADD COLUMN "closed_at" timestamptz,
  ADD COLUMN "closed_by" uuid,
  ADD COLUMN "version" bigint NOT NULL DEFAULT 1;

ALTER TABLE "orders"
  ADD CONSTRAINT "orders_flow_status_check"
    CHECK ("flow_status" IN ('DRAFT', 'BOOKED', 'SPACE_ALLOCATED', 'TRUCKING_ARRANGED', 'DOCUMENT_CUTOFF', 'CUSTOMS_DECLARATION_ARRANGED', 'DOCUMENT_RELEASED')),
  ADD CONSTRAINT "orders_termination_status_check"
    CHECK ("termination_status" IN ('ACTIVE', 'TERMINATING', 'TERMINATED')),
  ADD CONSTRAINT "orders_termination_type_check"
    CHECK ("termination_type" IS NULL OR "termination_type" IN ('CUSTOMER_CANCEL', 'CARRIER_CANCEL', 'CUSTOMS_RETURN', 'OPERATION_CANCEL', 'OTHER')),
  ADD CONSTRAINT "orders_termination_consistency_check"
    CHECK (
      ("termination_status" = 'ACTIVE' AND "termination_type" IS NULL AND "termination_reason" IS NULL AND "terminated_at" IS NULL AND "terminated_by" IS NULL)
      OR
      ("termination_status" = 'TERMINATING' AND "termination_type" IS NOT NULL AND "termination_reason" IS NOT NULL AND "terminated_at" IS NULL AND "terminated_by" IS NULL)
      OR
      ("termination_status" = 'TERMINATED' AND "termination_type" IS NOT NULL AND "termination_reason" IS NOT NULL AND "terminated_at" IS NOT NULL AND "terminated_by" IS NOT NULL)
    ),
  ADD CONSTRAINT "orders_closure_status_check"
    CHECK ("closure_status" IN ('OPEN', 'CLOSED')),
  ADD CONSTRAINT "orders_closure_consistency_check"
    CHECK (
      ("closure_status" = 'OPEN' AND "closure_reason" IS NULL AND "closed_at" IS NULL AND "closed_by" IS NULL)
      OR
      ("closure_status" = 'CLOSED' AND "closure_reason" IS NOT NULL AND "closed_at" IS NOT NULL AND "closed_by" IS NOT NULL)
    ),
  ADD CONSTRAINT "orders_version_positive_check" CHECK ("version" > 0);

CREATE TABLE "order_lifecycle_events" (
  "id" uuid PRIMARY KEY,
  "order_id" uuid NOT NULL REFERENCES "orders" ("id"),
  "dimension" varchar(16) NOT NULL,
  "from_status" varchar(64),
  "to_status" varchar(64) NOT NULL,
  "action" varchar(64) NOT NULL,
  "reason" varchar(500),
  "operator_id" uuid,
  "changed_at" timestamptz NOT NULL,
  CONSTRAINT "order_lifecycle_events_dimension_check" CHECK ("dimension" IN ('FLOW', 'TERMINATION', 'CLOSURE'))
);

INSERT INTO "order_lifecycle_events" (
  "id", "order_id", "dimension", "from_status", "to_status", "action", "reason", "operator_id", "changed_at"
)
SELECT "id", "order_id", 'FLOW', "from_status", "to_status", COALESCE(NULLIF("action", ''), 'transition'), NULLIF("reason", ''), "operator_id", "changed_at"
FROM "order_status_logs";

DROP TABLE "order_status_logs";

ALTER TABLE "orders" DROP CONSTRAINT "orders_status_templates_orders";
ALTER TABLE "orders" DROP COLUMN "status_template_id";
DROP TABLE "status_template_items";
DROP TABLE "status_templates";

DROP INDEX IF EXISTS "order_organization_id_status";
CREATE INDEX "order_organization_id_flow_status" ON "orders" ("organization_id", "flow_status");
CREATE INDEX "order_organization_id_termination_status" ON "orders" ("organization_id", "termination_status");
CREATE INDEX "order_organization_id_closure_status" ON "orders" ("organization_id", "closure_status");
CREATE INDEX "orderlifecycleevent_order_dimension_changed_at" ON "order_lifecycle_events" ("order_id", "dimension", "changed_at");

DELETE FROM "role_permissions"
WHERE "permission_id" IN (
  SELECT "id" FROM "permissions" WHERE "key" LIKE 'system.master_data.status_template.%'
);
DELETE FROM "permissions" WHERE "key" LIKE 'system.master_data.status_template.%';
