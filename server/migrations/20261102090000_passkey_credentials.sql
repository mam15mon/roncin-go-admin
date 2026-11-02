-- 由 Ent Schema 在正式迁移前态生成，仅新增通行密钥公共凭证表。
CREATE TABLE "passkey_credentials" ("id" uuid NOT NULL, "created_at" timestamptz NOT NULL, "updated_at" timestamptz NOT NULL, "rp_id" character varying NOT NULL, "credential_id" character varying NOT NULL, "credential_json" jsonb NOT NULL, "name" character varying NOT NULL, "last_used_at" timestamptz NULL, "user_id" uuid NOT NULL, PRIMARY KEY ("id"), CONSTRAINT "passkey_credentials_users_passkey_credentials" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE NO ACTION);
CREATE INDEX "passkeycredential_updated_at" ON "passkey_credentials" ("updated_at");
CREATE UNIQUE INDEX "passkey_credentials_rp_credential_key" ON "passkey_credentials" ("rp_id", "credential_id");
CREATE INDEX "passkey_credentials_user_rp" ON "passkey_credentials" ("user_id", "rp_id");
