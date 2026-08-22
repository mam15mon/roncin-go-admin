CREATE TABLE "partner_shipping_presets" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "preset_type" character varying NOT NULL,
  "title" character varying NOT NULL,
  "company_name" character varying NULL,
  "address" character varying NULL,
  "contact_name" character varying NULL,
  "phone" character varying NULL,
  "email" character varying NULL,
  "country_code" character varying NULL,
  "tax_identifier" character varying NULL,
  "content" character varying NULL,
  "code" character varying NULL,
  "is_default" boolean NOT NULL DEFAULT false,
  "sort_order" bigint NOT NULL DEFAULT 0,
  "remark" character varying NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  "partner_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "partner_shipping_presets_partners_shipping_presets"
    FOREIGN KEY ("partner_id") REFERENCES "partners" ("id") ON DELETE NO ACTION
);

CREATE INDEX "partnershippingpreset_updated_at"
  ON "partner_shipping_presets" ("updated_at");

CREATE INDEX "partnershippingpreset_partner_id_preset_type_enabled_sort_order"
  ON "partner_shipping_presets" ("partner_id", "preset_type", "enabled", "sort_order");

CREATE INDEX "partnershippingpreset_partner_id_preset_type_is_default"
  ON "partner_shipping_presets" ("partner_id", "preset_type", "is_default");

CREATE UNIQUE INDEX "partner_shipping_preset_default_key"
  ON "partner_shipping_presets" ("partner_id", "preset_type")
  WHERE "is_default" = true;
