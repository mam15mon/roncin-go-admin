-- 钉钉邀请模型演进：
--   1. ding_talk_invitations 增加 token（128-bit 随机不可枚举凭证）与唯一索引；
--   2. ding_talk_invitations 增加 kind（TARGETED 定向单人 vs GENERIC 扩招通用码）；
--   3. mobile 与 role_id 设为可空（通用码无手机号，预设角色可选）；
--   4. 活跃部分唯一索引调整为仅针对 TARGETED 且手机号非空记录。

ALTER TABLE "ding_talk_invitations"
  ADD COLUMN "token" character varying(64);

UPDATE "ding_talk_invitations"
  SET "token" = md5(id::text || random()::text)
  WHERE "token" IS NULL;

ALTER TABLE "ding_talk_invitations"
  ALTER COLUMN "token" SET NOT NULL;

CREATE UNIQUE INDEX "dingtalkinvitation_token"
  ON "ding_talk_invitations" ("token");

ALTER TABLE "ding_talk_invitations"
  ADD COLUMN "kind" character varying NOT NULL DEFAULT 'TARGETED';

ALTER TABLE "ding_talk_invitations"
  ADD CONSTRAINT "ding_talk_invitations_kind_check"
    CHECK ("kind" IN ('TARGETED', 'GENERIC'));

ALTER TABLE "ding_talk_invitations"
  ALTER COLUMN "mobile" DROP NOT NULL;

ALTER TABLE "ding_talk_invitations"
  ALTER COLUMN "role_id" DROP NOT NULL;

DROP INDEX IF EXISTS "dingtalkinvitation_organization_id_mobile";

CREATE UNIQUE INDEX "dingtalkinvitation_organization_id_mobile"
  ON "ding_talk_invitations" ("organization_id", "mobile")
  WHERE "status" = 'PENDING' AND "kind" = 'TARGETED' AND "mobile" IS NOT NULL AND "mobile" != '';
