DROP INDEX IF EXISTS "orderpersonnel_order_id_role_user_id";
DROP INDEX IF EXISTS "orderpersonnel_order_single_role_unique";

CREATE UNIQUE INDEX "orderpersonnel_order_id_role"
  ON "order_personnels" ("order_id", "role");

DELETE FROM "role_permissions"
WHERE "permission_id" IN (
  SELECT "id"
  FROM "permissions"
  WHERE "key" ~ '^business[.]order[.](se|si|ae|ai)[.]shipping_document[.](read|create|update|transition|delete)$'
);

DELETE FROM "permissions"
WHERE "key" ~ '^business[.]order[.](se|si|ae|ai)[.]shipping_document[.](read|create|update|transition|delete)$';
