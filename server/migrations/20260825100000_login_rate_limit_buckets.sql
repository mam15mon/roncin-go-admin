CREATE TABLE "login_rate_limit_buckets" (
  "id" uuid NOT NULL,
  "created_at" timestamp with time zone NOT NULL,
  "updated_at" timestamp with time zone NOT NULL,
  "key_hash" character varying(64) NOT NULL,
  "window_started_at" timestamp with time zone NOT NULL,
  "attempts" bigint NOT NULL,
  PRIMARY KEY ("id")
);

CREATE INDEX "loginratelimitbucket_updated_at"
  ON "login_rate_limit_buckets" ("updated_at");

CREATE UNIQUE INDEX "loginratelimitbucket_key_hash"
  ON "login_rate_limit_buckets" ("key_hash");
