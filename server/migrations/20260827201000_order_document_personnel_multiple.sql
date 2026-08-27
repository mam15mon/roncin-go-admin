DROP INDEX IF EXISTS "orderpersonnel_order_id_role";

CREATE UNIQUE INDEX "orderpersonnel_order_id_role_user_id"
  ON "order_personnels" ("order_id", "role", "user_id");

CREATE UNIQUE INDEX "orderpersonnel_order_single_role_unique"
  ON "order_personnels" ("order_id", "role")
  WHERE "role" <> 'DOCUMENT';
