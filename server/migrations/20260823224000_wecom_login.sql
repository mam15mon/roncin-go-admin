ALTER TABLE "users"
    ALTER COLUMN "password_hash" DROP NOT NULL,
    ADD COLUMN "wecom_userid" character varying(64) NULL,
    ADD COLUMN "wecom_name" character varying(100) NULL;

CREATE UNIQUE INDEX "users_wecom_userid_key"
    ON "users" ("wecom_userid");
