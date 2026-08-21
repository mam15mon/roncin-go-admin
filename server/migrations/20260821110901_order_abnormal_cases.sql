-- 订单异常标记表；结构与 ent/schema/order_abnormal_case.go 保持一致。
CREATE TABLE "order_abnormal_cases" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "abnormal_case_id" uuid NOT NULL,
  "status" character varying NOT NULL DEFAULT 'ACTIVE',
  "marked_at" timestamp with time zone NOT NULL,
  "marked_by" uuid NOT NULL,
  "resolved_at" timestamp with time zone NULL,
  "resolved_by" uuid NULL,
  "order_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "order_abnormal_cases_status_check"
    CHECK ("status" IN ('ACTIVE', 'RESOLVED')),
  CONSTRAINT "order_abnormal_cases_orders_abnormal_cases"
    FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION
);

CREATE INDEX "orderabnormalcase_updated_at"
  ON "order_abnormal_cases" ("updated_at");

CREATE UNIQUE INDEX "orderabnormalcase_order_id_abnormal_case_id"
  ON "order_abnormal_cases" ("order_id", "abnormal_case_id");

CREATE INDEX "orderabnormalcase_order_id_status_marked_at"
  ON "order_abnormal_cases" ("order_id", "status", "marked_at");