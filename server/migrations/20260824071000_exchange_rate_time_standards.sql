-- 汇率时间标准改为按汇率类型统一配置，不再重复保存在每条币种汇率中。
CREATE TABLE "exchange_rate_time_standards" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "organization_id" uuid NOT NULL,
  "rate_type" character varying NOT NULL,
  "time_standard" character varying NOT NULL,
  "sort_order" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "exchange_rate_time_standards_rate_type_check"
    CHECK ("rate_type" IN ('BASE_CURRENCY', 'INVOICE', 'SETTLEMENT', 'WRITE_OFF', 'BILL')),
  CONSTRAINT "exchange_rate_time_standards_time_standard_check"
    CHECK ("time_standard" IN ('ETD_ETA_TRAIN_DATE', 'BUSINESS_TIME', 'BARGE_ETD', 'EXPENSE_TIME', 'ORDER_CREATED_AT', 'BILL_CREATED_AT', 'WRITE_OFF_TIME')),
  CONSTRAINT "exchange_rate_time_standards_sort_order_nonnegative" CHECK ("sort_order" >= 0),
  CONSTRAINT "exchange_rate_time_standards_organizations_time_standards"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "exchange_rate_time_standard_unique"
  ON "exchange_rate_time_standards" ("organization_id", "rate_type", "time_standard");
CREATE UNIQUE INDEX "exchange_rate_time_standard_sort_unique"
  ON "exchange_rate_time_standards" ("organization_id", "rate_type", "sort_order");
CREATE INDEX "exchange_rate_time_standard_updated_at"
  ON "exchange_rate_time_standards" ("updated_at");

WITH headquarters AS (
  SELECT "id"
  FROM "organizations"
  WHERE "kind" = 'headquarters' AND "parent_id" IS NULL
), defaults("rate_type", "time_standard", "sort_order") AS (
  VALUES
    ('BASE_CURRENCY', 'ETD_ETA_TRAIN_DATE', 0),
    ('BASE_CURRENCY', 'BUSINESS_TIME', 1),
    ('BASE_CURRENCY', 'BARGE_ETD', 2),
    ('BASE_CURRENCY', 'ORDER_CREATED_AT', 3),
    ('INVOICE', 'BILL_CREATED_AT', 0),
    ('SETTLEMENT', 'ETD_ETA_TRAIN_DATE', 0),
    ('SETTLEMENT', 'BUSINESS_TIME', 1),
    ('SETTLEMENT', 'BARGE_ETD', 2),
    ('SETTLEMENT', 'EXPENSE_TIME', 3),
    ('SETTLEMENT', 'ORDER_CREATED_AT', 4),
    ('WRITE_OFF', 'WRITE_OFF_TIME', 0),
    ('BILL', 'BILL_CREATED_AT', 0)
)
INSERT INTO "exchange_rate_time_standards" (
  "id", "created_at", "updated_at", "organization_id", "rate_type", "time_standard", "sort_order"
)
SELECT
  md5(headquarters."id"::text || ':' || defaults."rate_type" || ':' || defaults."time_standard")::uuid,
  CURRENT_TIMESTAMP,
  CURRENT_TIMESTAMP,
  headquarters."id",
  defaults."rate_type",
  defaults."time_standard",
  defaults."sort_order"
FROM headquarters CROSS JOIN defaults;

DROP INDEX "exchange_rate_setting_unique_effective_from";
DROP INDEX "exchange_rate_setting_active_lookup";
ALTER TABLE "exchange_rate_settings" DROP COLUMN "time_standard";
CREATE UNIQUE INDEX "exchange_rate_setting_unique_effective_from"
  ON "exchange_rate_settings" ("organization_id", "rate_type", "from_currency", "to_currency", "effective_from");
CREATE INDEX "exchange_rate_setting_active_lookup"
  ON "exchange_rate_settings" ("organization_id", "rate_type", "from_currency", "to_currency", "is_active");
