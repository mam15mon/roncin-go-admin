ALTER TABLE "users"
  ALTER COLUMN "username" DROP NOT NULL;

DROP INDEX "user_username";

CREATE UNIQUE INDEX "user_username"
  ON "users" ("username")
  WHERE "username" IS NOT NULL AND "username" <> '';
