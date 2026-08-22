ALTER TABLE "master_data_items"
    ADD COLUMN "attributes" jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX "masterdataitem_organization_id_kind_transport_mode"
    ON "master_data_items" ("organization_id", "kind", "transport_mode");
