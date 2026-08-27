CREATE TABLE "finance_fee_ledger_preferences"(
  "id" uuid PRIMARY KEY,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL REFERENCES "organizations"("id"),
  "user_id" uuid NOT NULL REFERENCES "users"("id"),
  "columns" jsonb NOT NULL,
  "page_size" integer NOT NULL,
  "sort_field" varchar(64),
  "sort_direction" varchar(4),
  "row_colors" jsonb NOT NULL,
  "version" bigint NOT NULL DEFAULT 1,
  CONSTRAINT "finance_fee_ledger_preference_page_size_check" CHECK("page_size" IN(40,60,100)),
  CONSTRAINT "finance_fee_ledger_preference_sort_direction_check" CHECK("sort_direction" IS NULL OR "sort_direction" IN('ASC','DESC')),
  CONSTRAINT "finance_fee_ledger_preference_sort_pair_check" CHECK(("sort_field" IS NULL) = ("sort_direction" IS NULL))
);

CREATE UNIQUE INDEX "financefeeledgerpreference_org_user" ON "finance_fee_ledger_preferences"("organization_id","user_id");
CREATE INDEX "financefeeledgerpreference_user_id" ON "finance_fee_ledger_preferences"("user_id");
