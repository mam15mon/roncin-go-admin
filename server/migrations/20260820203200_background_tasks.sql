-- 后台任务表；结构与 ent/schema/background_task.go 保持一致。
CREATE TABLE "background_tasks" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "kind" character varying NOT NULL,
  "idempotency_key" character varying(128) NOT NULL,
  "status" character varying NOT NULL DEFAULT 'PENDING',
  "attempts" bigint NOT NULL DEFAULT 0,
  "max_attempts" bigint NOT NULL DEFAULT 3,
  "next_run_at" timestamp with time zone NOT NULL,
  "lease_token" character varying(128) NULL,
  "lease_expires_at" timestamp with time zone NULL,
  "last_error" character varying(2000) NULL,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "background_tasks_kind_check"
    CHECK ("kind" IN ('MASTER_DATA_IMPORT', 'UNLOCODE_IMPORT', 'ORDER_REMINDER', 'INTEGRATION')),
  CONSTRAINT "background_tasks_status_check"
    CHECK ("status" IN ('PENDING', 'RUNNING', 'SUCCEEDED', 'FAILED', 'DEAD_LETTER')),
  CONSTRAINT "background_tasks_organizations_background_tasks"
    FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE INDEX "backgroundtask_updated_at"
  ON "background_tasks" ("updated_at");

CREATE UNIQUE INDEX "backgroundtask_organization_id_kind_idempotency_key"
  ON "background_tasks" ("organization_id", "kind", "idempotency_key");

CREATE INDEX "backgroundtask_status_next_run_at"
  ON "background_tasks" ("status", "next_run_at");

CREATE INDEX "backgroundtask_status_lease_expires_at"
  ON "background_tasks" ("status", "lease_expires_at");

CREATE INDEX "backgroundtask_organization_id_created_at"
  ON "background_tasks" ("organization_id", "created_at");
