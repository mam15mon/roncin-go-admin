-- 手机号来源快照保留，新增非唯一检索投影；重复号码由认证入口拒绝。
ALTER TABLE "users" ADD COLUMN "dingtalk_mobile_normalized" varchar(11);
WITH "normalized" AS (
 SELECT "id", regexp_replace(regexp_replace("dingtalk_mobile",
   U&'[\0009-\000D\0020\0085\00A0\1680\2000-\200A\2028\2029\202F\205F\3000-]', '', 'g'),
   '^(\+86|0086)', '') AS "mobile" FROM "users"
)
UPDATE "users" SET "dingtalk_mobile_normalized" = "normalized"."mobile"
FROM "normalized" WHERE "users"."id" = "normalized"."id" AND "normalized"."mobile" ~ '^1[3-9][0-9]{9}$';
CREATE INDEX "user_dingtalk_mobile_normalized" ON "users" ("dingtalk_mobile_normalized");

CREATE TABLE "password_reset_challenges" (
 "id" uuid NOT NULL PRIMARY KEY,
 "created_at" timestamptz NOT NULL,
 "updated_at" timestamptz NOT NULL,
 "mobile_key_hash" varchar(64) NOT NULL,
 "token_hash" varchar(64) NOT NULL,
 "code_hash" varchar(64) NOT NULL DEFAULT '',
 "user_id" uuid,
 "dingtalk_userid" varchar(64) NOT NULL DEFAULT '',
 "dingtalk_unionid" varchar(128) NOT NULL DEFAULT '',
 "status" varchar NOT NULL,
 "expires_at" timestamptz NOT NULL,
 "resend_available_at" timestamptz NOT NULL,
 "failed_attempts" bigint NOT NULL DEFAULT 0,
 CONSTRAINT "password_reset_challenges_status_check" CHECK ("status" IN ('PENDING', 'READY', 'FAILED', 'CONSUMED')),
 CONSTRAINT "password_reset_challenges_failed_attempts_check" CHECK ("failed_attempts" BETWEEN 0 AND 5)
);
CREATE UNIQUE INDEX "passwordresetchallenge_mobile_key_hash" ON "password_reset_challenges" ("mobile_key_hash");
CREATE UNIQUE INDEX "passwordresetchallenge_token_hash" ON "password_reset_challenges" ("token_hash");
CREATE INDEX "passwordresetchallenge_user_id" ON "password_reset_challenges" ("user_id");
CREATE INDEX "passwordresetchallenge_updated_at" ON "password_reset_challenges" ("updated_at");
