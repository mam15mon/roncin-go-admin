-- 总部和公司明确维护本币；部门和团队按组织层级继承，不重复存储。
ALTER TABLE "organizations"
  ADD COLUMN "base_currency" character varying(3) NULL;

UPDATE "organizations"
SET "base_currency" = 'CNY'
WHERE "kind" IN ('headquarters', 'company');

ALTER TABLE "organizations"
  ADD CONSTRAINT "organizations_base_currency_by_kind"
    CHECK (
      ("kind" IN ('headquarters', 'company') AND "base_currency" IS NOT NULL)
      OR
      ("kind" IN ('department', 'team') AND "base_currency" IS NULL)
    ),
  ADD CONSTRAINT "organizations_currencies_base_currency"
    FOREIGN KEY ("base_currency") REFERENCES "currencies" ("code") ON DELETE NO ACTION;

CREATE INDEX "organization_base_currency" ON "organizations" ("base_currency");
