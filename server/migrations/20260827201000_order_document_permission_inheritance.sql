DELETE FROM "role_permissions"
WHERE "permission_id" IN (
  SELECT "id"
  FROM "permissions"
  WHERE "key" ~ '^business[.]order[.](se|si|ae|ai)[.]shipping_document[.](read|create|update|transition|delete)$'
);

DELETE FROM "permissions"
WHERE "key" ~ '^business[.]order[.](se|si|ae|ai)[.]shipping_document[.](read|create|update|transition|delete)$';
