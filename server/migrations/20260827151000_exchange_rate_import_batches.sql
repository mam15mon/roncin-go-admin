-- 汇率 Excel 两阶段导入批次。原始文件不保存，只保存 SHA-256 和规范化预检结果。
CREATE TABLE "exchange_rate_import_batches" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL,
  "owner_organization_id" uuid NOT NULL,
  "created_by" uuid NOT NULL,
  "file_name" varchar(255) NOT NULL,
  "file_checksum" varchar(64) NOT NULL,
  "template_version" bigint NOT NULL,
  "status" varchar NOT NULL,
  "preview_token_hash" varchar(64) NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "idempotency_key" varchar(128) NULL,
  "total_count" bigint NOT NULL,
  "valid_count" bigint NOT NULL,
  "invalid_count" bigint NOT NULL,
  "imported_count" bigint NOT NULL DEFAULT 0,
  "rows" jsonb NOT NULL,
  "imported_at" timestamptz NULL,
  "imported_by" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "exchange_rate_import_batches_status_check"
    CHECK ("status" IN ('PREVIEW_READY', 'PREVIEW_INVALID', 'IMPORTED')),
  CONSTRAINT "exchange_rate_import_batches_counts_check"
    CHECK (
      "template_version" > 0
      AND "total_count" >= 0
      AND "valid_count" >= 0
      AND "invalid_count" >= 0
      AND "imported_count" >= 0
      AND "total_count" = "valid_count" + "invalid_count"
      AND "imported_count" <= "valid_count"
    ),
  CONSTRAINT "exchange_rate_import_batches_result_check"
    CHECK (
      ("status" <> 'IMPORTED' AND "idempotency_key" IS NULL AND "imported_at" IS NULL AND "imported_by" IS NULL AND "imported_count" = 0)
      OR
      ("status" = 'IMPORTED' AND "idempotency_key" IS NOT NULL AND "imported_at" IS NOT NULL AND "imported_by" IS NOT NULL AND "imported_count" = "valid_count" AND "invalid_count" = 0)
    ),
  CONSTRAINT "exchange_rate_import_batches_organization_fk"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "exchange_rate_import_batches_owner_organization_fk"
    FOREIGN KEY ("owner_organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "exchange_rate_import_batches_created_by_fk"
    FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE NO ACTION,
  CONSTRAINT "exchange_rate_import_batches_imported_by_fk"
    FOREIGN KEY ("imported_by") REFERENCES "users" ("id") ON DELETE NO ACTION
);

CREATE UNIQUE INDEX "exchange_rate_import_preview_token"
  ON "exchange_rate_import_batches" ("preview_token_hash");
CREATE INDEX "exchange_rate_import_org_id"
  ON "exchange_rate_import_batches" ("organization_id", "id");
CREATE UNIQUE INDEX "exchange_rate_import_idempotency"
  ON "exchange_rate_import_batches" ("organization_id", "idempotency_key")
  WHERE "idempotency_key" IS NOT NULL;
CREATE INDEX "exchange_rate_import_created_at"
  ON "exchange_rate_import_batches" ("organization_id", "created_at");
CREATE INDEX "exchange_rate_import_batches_updated_at"
  ON "exchange_rate_import_batches" ("updated_at");
