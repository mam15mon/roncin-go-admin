ALTER TABLE "users"
  ADD COLUMN "dingtalk_userid" character varying(64) NULL;

CREATE UNIQUE INDEX "user_dingtalk_userid"
  ON "users" ("dingtalk_userid");

ALTER TABLE "background_tasks"
  DROP CONSTRAINT "background_tasks_kind_check";

ALTER TABLE "background_tasks"
  ADD CONSTRAINT "background_tasks_kind_check"
    CHECK ("kind" IN ('MASTER_DATA_IMPORT', 'UNLOCODE_IMPORT', 'ORDER_REMINDER', 'INTEGRATION', 'DINGTALK_NOTIFICATION'));

CREATE TABLE "notification_deliveries" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "channel" character varying NOT NULL,
  "template" character varying NOT NULL,
  "resource_type" character varying(64) NOT NULL,
  "resource_id" uuid NOT NULL,
  "reference_code" character varying(64) NOT NULL,
  "parameter" character varying(64) NULL,
  "external_message_id" character varying(256) NULL,
  "background_task_id" uuid NOT NULL,
  "recipient_user_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "notification_deliveries_channel_check"
    CHECK ("channel" IN ('DINGTALK')),
  CONSTRAINT "notification_deliveries_template_check"
    CHECK ("template" IN ('ORDER_PERSONNEL_ASSIGNED')),
  CONSTRAINT "notification_deliveries_background_tasks_notification_delivery"
    FOREIGN KEY ("background_task_id") REFERENCES "background_tasks" ("id") ON DELETE NO ACTION,
  CONSTRAINT "notification_deliveries_users_notification_deliveries"
    FOREIGN KEY ("recipient_user_id") REFERENCES "users" ("id") ON DELETE NO ACTION
);

CREATE INDEX "notificationdelivery_updated_at"
  ON "notification_deliveries" ("updated_at");

CREATE UNIQUE INDEX "notificationdelivery_background_task_id"
  ON "notification_deliveries" ("background_task_id");

CREATE INDEX "notificationdelivery_recipient_user_id_created_at"
  ON "notification_deliveries" ("recipient_user_id", "created_at");

CREATE INDEX "notificationdelivery_resource_type_resource_id"
  ON "notification_deliveries" ("resource_type", "resource_id");
