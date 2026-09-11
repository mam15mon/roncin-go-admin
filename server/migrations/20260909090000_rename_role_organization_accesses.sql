ALTER TABLE "role_order_organization_accesses" RENAME TO "role_organization_accesses";

DO $$
DECLARE
  v_fk_role text;
  v_fk_org text;
BEGIN
  IF to_regclass('role_organization_accesses') IS NULL THEN
    RETURN;
  END IF;

  SELECT conname INTO v_fk_role
  FROM pg_constraint
  WHERE conrelid = 'role_organization_accesses'::regclass
    AND contype = 'f'
    AND conkey = ARRAY[(SELECT attnum FROM pg_attribute WHERE attrelid = 'role_organization_accesses'::regclass AND attname = 'role_id')];

  IF v_fk_role IS NOT NULL AND v_fk_role <> 'role_organization_accesses_roles_organization_accesses' THEN
    EXECUTE format('ALTER TABLE "role_organization_accesses" RENAME CONSTRAINT %I TO %I',
      v_fk_role, 'role_organization_accesses_roles_organization_accesses');
  END IF;

  SELECT conname INTO v_fk_org
  FROM pg_constraint
  WHERE conrelid = 'role_organization_accesses'::regclass
    AND contype = 'f'
    AND conkey = ARRAY[(SELECT attnum FROM pg_attribute WHERE attrelid = 'role_organization_accesses'::regclass AND attname = 'organization_id')];

  IF v_fk_org IS NOT NULL AND v_fk_org NOT LIKE 'role_organization_accesses_organizations_role_organization_acc%' THEN
    EXECUTE format('ALTER TABLE "role_organization_accesses" RENAME CONSTRAINT %I TO %I',
      v_fk_org, 'role_organization_accesses_organizations_role_organization_accesses');
  END IF;
END $$;

ALTER INDEX IF EXISTS "role_order_organization_accesses_pkey"
  RENAME TO "role_organization_accesses_pkey";

ALTER INDEX IF EXISTS "roleorderorganizationaccess_updated_at"
  RENAME TO "roleorganizationaccess_updated_at";

ALTER INDEX IF EXISTS "roleorderorganizationaccess_role_id_organization_id"
  RENAME TO "roleorganizationaccess_role_id_organization_id";

ALTER INDEX IF EXISTS "roleorderorganizationaccess_organization_id"
  RENAME TO "roleorganizationaccess_organization_id";

