CREATE TABLE "enterprise_resources" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "resource_type" character varying NOT NULL,
  "short_name" character varying NOT NULL,
  "enabled" boolean NOT NULL DEFAULT true,
  "sort_order" bigint NOT NULL DEFAULT 0,
  "search_keywords" text NOT NULL DEFAULT '',
  "organization_id" uuid NOT NULL,
  "created_by" uuid NULL,
  "updated_by" uuid NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resources_organizations_enterprise_resources" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION,
  CONSTRAINT "enterprise_resources_users_created_enterprise_resources" FOREIGN KEY ("created_by") REFERENCES "users" ("id") ON DELETE SET NULL,
  CONSTRAINT "enterprise_resources_users_updated_enterprise_resources" FOREIGN KEY ("updated_by") REFERENCES "users" ("id") ON DELETE SET NULL
);

CREATE INDEX "enterpriseresource_updated_at" ON "enterprise_resources" ("updated_at");
CREATE INDEX "enterpriseresource_organization_id_resource_type_enabled_sort_order" ON "enterprise_resources" ("organization_id", "resource_type", "enabled", "sort_order");
CREATE INDEX "enterpriseresource_organization_id_updated_at" ON "enterprise_resources" ("organization_id", "updated_at");
CREATE INDEX "enterprise_resources_search_keywords_trgm" ON "enterprise_resources" USING gin ("search_keywords" gin_trgm_ops);

CREATE TABLE "enterprise_tag_groups" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "name" character varying NOT NULL,
  "normalized_name" character varying NOT NULL,
  "color" character varying NULL,
  "sort_order" bigint NOT NULL DEFAULT 0,
  "organization_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_tag_groups_organizations_enterprise_tag_groups" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterprisetaggroup_updated_at" ON "enterprise_tag_groups" ("updated_at");
CREATE UNIQUE INDEX "enterprisetaggroup_organization_id_normalized_name" ON "enterprise_tag_groups" ("organization_id", "normalized_name");
CREATE INDEX "enterprisetaggroup_organization_id_sort_order" ON "enterprise_tag_groups" ("organization_id", "sort_order");

CREATE TABLE "enterprise_resource_addresses" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "contact_name" character varying NULL,
  "contact_phone" character varying NULL,
  "country_code" character varying NOT NULL,
  "province_code" character varying NULL,
  "city_code" character varying NULL,
  "district_code" character varying NULL,
  "address_detail" character varying NOT NULL,
  "remark" character varying NULL,
  "resource_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resource_addresses_enterprise_resources_address" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterpriseresourceaddress_updated_at" ON "enterprise_resource_addresses" ("updated_at");
CREATE UNIQUE INDEX "enterpriseresourceaddress_resource_id" ON "enterprise_resource_addresses" ("resource_id");

CREATE TABLE "enterprise_resource_remarks" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "remark_type" character varying NOT NULL,
  "content" character varying NOT NULL,
  "resource_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resource_remarks_enterprise_resources_remark" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterpriseresourceremark_updated_at" ON "enterprise_resource_remarks" ("updated_at");
CREATE UNIQUE INDEX "enterpriseresourceremark_resource_id" ON "enterprise_resource_remarks" ("resource_id");
CREATE INDEX "enterpriseresourceremark_remark_type" ON "enterprise_resource_remarks" ("remark_type");

CREATE TABLE "enterprise_resource_images" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "file_name" character varying NOT NULL,
  "mime_type" character varying NOT NULL,
  "file_size" bigint NOT NULL,
  "object_key" character varying NOT NULL,
  "checksum" character varying NOT NULL,
  "width" bigint NULL,
  "height" bigint NULL,
  "resource_id" uuid NOT NULL,
  "uploaded_by" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resource_images_enterprise_resources_image" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION,
  CONSTRAINT "enterprise_resource_images_users_uploaded_enterprise_resource_images" FOREIGN KEY ("uploaded_by") REFERENCES "users" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterpriseresourceimage_updated_at" ON "enterprise_resource_images" ("updated_at");
CREATE UNIQUE INDEX "enterpriseresourceimage_resource_id" ON "enterprise_resource_images" ("resource_id");
CREATE UNIQUE INDEX "enterpriseresourceimage_object_key" ON "enterprise_resource_images" ("object_key");

CREATE TABLE "enterprise_resource_parties" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL,
  "resource_type" character varying NOT NULL,
  "company_name" character varying NOT NULL,
  "business_code" character varying NULL,
  "normalized_business_code" character varying NULL,
  "address" character varying NULL,
  "country_code" character varying NOT NULL DEFAULT 'CN',
  "contact_name" character varying NULL,
  "contact_phone" character varying NULL,
  "email" character varying NULL,
  "tax_identifier" character varying NULL,
  "aeo_code" character varying NULL,
  "custom_display" boolean NOT NULL DEFAULT false,
  "display_content" character varying NULL,
  "remark" character varying NULL,
  "resource_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resource_parties_enterprise_resources_party" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterpriseresourceparty_updated_at" ON "enterprise_resource_parties" ("updated_at");
CREATE UNIQUE INDEX "enterpriseresourceparty_resource_id" ON "enterprise_resource_parties" ("resource_id");
CREATE UNIQUE INDEX "enterpriseresourceparty_organization_id_resource_type_normalized_business_code" ON "enterprise_resource_parties" ("organization_id", "resource_type", "normalized_business_code") WHERE normalized_business_code IS NOT NULL AND normalized_business_code <> '';

CREATE TABLE "enterprise_resource_shipping_texts" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "content" character varying NULL,
  "code" character varying NULL,
  "remark" character varying NULL,
  "resource_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resource_shipping_texts_enterprise_resources_shipping_text" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterpriseresourceshippingtext_updated_at" ON "enterprise_resource_shipping_texts" ("updated_at");
CREATE UNIQUE INDEX "enterpriseresourceshippingtext_resource_id" ON "enterprise_resource_shipping_texts" ("resource_id");

CREATE TABLE "enterprise_resource_partners" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "resource_type" character varying NOT NULL,
  "is_default" boolean NOT NULL DEFAULT false,
  "resource_id" uuid NOT NULL,
  "partner_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resource_partners_enterprise_resources_partner_links" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION,
  CONSTRAINT "enterprise_resource_partners_partners_enterprise_resource_links" FOREIGN KEY ("partner_id") REFERENCES "partners" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterpriseresourcepartner_updated_at" ON "enterprise_resource_partners" ("updated_at");
CREATE UNIQUE INDEX "enterpriseresourcepartner_resource_id_partner_id" ON "enterprise_resource_partners" ("resource_id", "partner_id");
CREATE INDEX "enterpriseresourcepartner_partner_id_resource_type" ON "enterprise_resource_partners" ("partner_id", "resource_type");
CREATE UNIQUE INDEX "enterprise_resource_partner_default_key" ON "enterprise_resource_partners" ("partner_id", "resource_type") WHERE "is_default" = true;

CREATE TABLE "enterprise_resource_assignees" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "resource_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resource_assignees_enterprise_resources_assignees" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION,
  CONSTRAINT "enterprise_resource_assignees_users_enterprise_resource_assignments" FOREIGN KEY ("user_id") REFERENCES "users" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterpriseresourceassignee_updated_at" ON "enterprise_resource_assignees" ("updated_at");
CREATE UNIQUE INDEX "enterpriseresourceassignee_resource_id_user_id" ON "enterprise_resource_assignees" ("resource_id", "user_id");
CREATE INDEX "enterpriseresourceassignee_user_id" ON "enterprise_resource_assignees" ("user_id");

CREATE TABLE "enterprise_resource_address_types" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "address_type" character varying NOT NULL,
  "resource_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_resource_address_types_enterprise_resources_address_types" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterpriseresourceaddresstype_updated_at" ON "enterprise_resource_address_types" ("updated_at");
CREATE UNIQUE INDEX "enterpriseresourceaddresstype_resource_id_address_type" ON "enterprise_resource_address_types" ("resource_id", "address_type");
CREATE INDEX "enterpriseresourceaddresstype_address_type" ON "enterprise_resource_address_types" ("address_type");

CREATE TABLE "enterprise_tags" (
  "id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  "organization_id" uuid NOT NULL,
  "normalized_name" character varying NOT NULL,
  "sort_order" bigint NOT NULL DEFAULT 0,
  "resource_id" uuid NOT NULL,
  "group_id" uuid NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "enterprise_tags_enterprise_resources_tag" FOREIGN KEY ("resource_id") REFERENCES "enterprise_resources" ("id") ON DELETE NO ACTION,
  CONSTRAINT "enterprise_tags_enterprise_tag_groups_tags" FOREIGN KEY ("group_id") REFERENCES "enterprise_tag_groups" ("id") ON DELETE NO ACTION
);

CREATE INDEX "enterprisetag_updated_at" ON "enterprise_tags" ("updated_at");
CREATE UNIQUE INDEX "enterprisetag_resource_id" ON "enterprise_tags" ("resource_id");
CREATE UNIQUE INDEX "enterprisetag_group_id_normalized_name" ON "enterprise_tags" ("group_id", "normalized_name");
CREATE INDEX "enterprisetag_organization_id_group_id_sort_order" ON "enterprise_tags" ("organization_id", "group_id", "sort_order");
