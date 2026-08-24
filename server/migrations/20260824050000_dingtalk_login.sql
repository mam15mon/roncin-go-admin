ALTER TABLE "users"
  ADD COLUMN "dingtalk_unionid" character varying(128) NULL,
  ADD COLUMN "dingtalk_name" character varying(100) NULL;

CREATE UNIQUE INDEX "user_dingtalk_unionid" ON "users" ("dingtalk_unionid");
