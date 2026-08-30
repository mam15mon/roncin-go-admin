-- 企业图片删除任务：删除图片资源时在同一事务登记类型化任务，提交后由
-- 对象删除 Worker 领取并删除对象存储文件；结构与 ent/schema/object_storage_deletion.go
-- 及 background_task.go 保持一致。
ALTER TABLE "background_tasks"
  DROP CONSTRAINT "background_tasks_kind_check";

ALTER TABLE "background_tasks"
  ADD CONSTRAINT "background_tasks_kind_check"
    CHECK ("kind" IN ('MASTER_DATA_IMPORT', 'UNLOCODE_IMPORT', 'ORDER_REMINDER', 'INTEGRATION', 'DINGTALK_NOTIFICATION', 'OBJECT_STORAGE_DELETION'));

CREATE TABLE "object_storage_deletions" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "object_key" character varying(1024) NOT NULL,
  "background_task_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "object_storage_deletions_background_tasks_object_storage_deletion"
    FOREIGN KEY ("background_task_id") REFERENCES "background_tasks" ("id") ON DELETE NO ACTION
);

CREATE INDEX "objectstoragedeletion_updated_at"
  ON "object_storage_deletions" ("updated_at");

CREATE UNIQUE INDEX "objectstoragedeletion_background_task_id"
  ON "object_storage_deletions" ("background_task_id");
