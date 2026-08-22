CREATE TABLE "ports" (
    "id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "un_locode" varchar(5) NOT NULL,
    "name_zh" varchar(200) NOT NULL,
    "name_en" varchar(200) NOT NULL,
    "country_code" varchar(2) NOT NULL,
    "transport_modes" jsonb NOT NULL,
    "source" varchar(100) NOT NULL DEFAULT 'manual',
    "sort_order" bigint NOT NULL DEFAULT 100,
    "enabled" boolean NOT NULL DEFAULT true,
    "organization_id" uuid NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "ports_organizations_ports" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "ports_un_locode_check" CHECK ("un_locode" ~ '^[A-Z]{2}[A-Z0-9]{3}$'),
    CONSTRAINT "ports_country_code_check" CHECK ("country_code" ~ '^[A-Z]{2}$' AND left("un_locode", 2) = "country_code")
);
CREATE UNIQUE INDEX "port_organization_id_un_locode" ON "ports" ("organization_id", "un_locode");
CREATE INDEX "port_organization_id_enabled_sort_order" ON "ports" ("organization_id", "enabled", "sort_order");
CREATE INDEX "port_updated_at" ON "ports" ("updated_at");

CREATE TABLE "airports" (
    "id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "iata_code" varchar(3) NOT NULL,
    "icao_code" varchar(4) NULL,
    "name_zh" varchar(200) NOT NULL,
    "name_en" varchar(200) NOT NULL,
    "city_name_zh" varchar(100) NOT NULL,
    "city_name_en" varchar(100) NULL,
    "country_code" varchar(2) NOT NULL,
    "source" varchar(100) NOT NULL DEFAULT 'manual',
    "sort_order" bigint NOT NULL DEFAULT 100,
    "enabled" boolean NOT NULL DEFAULT true,
    "organization_id" uuid NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "airports_organizations_airports" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "airports_iata_code_check" CHECK ("iata_code" ~ '^[A-Z]{3}$'),
    CONSTRAINT "airports_icao_code_check" CHECK ("icao_code" IS NULL OR "icao_code" ~ '^[A-Z0-9]{4}$'),
    CONSTRAINT "airports_country_code_check" CHECK ("country_code" ~ '^[A-Z]{2}$')
);
CREATE UNIQUE INDEX "airport_organization_id_iata_code" ON "airports" ("organization_id", "iata_code");
CREATE UNIQUE INDEX "airport_organization_id_icao_code" ON "airports" ("organization_id", "icao_code");
CREATE INDEX "airport_organization_id_enabled_sort_order" ON "airports" ("organization_id", "enabled", "sort_order");
CREATE INDEX "airport_updated_at" ON "airports" ("updated_at");

CREATE TABLE "airlines" (
    "id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "iata_code" varchar(2) NOT NULL,
    "icao_code" varchar(3) NULL,
    "awb_prefix" varchar(3) NOT NULL,
    "name_zh" varchar(200) NOT NULL,
    "name_en" varchar(200) NOT NULL,
    "country_code" varchar(2) NOT NULL,
    "cargo_only" boolean NOT NULL DEFAULT false,
    "source" varchar(100) NOT NULL DEFAULT 'manual',
    "sort_order" bigint NOT NULL DEFAULT 100,
    "enabled" boolean NOT NULL DEFAULT true,
    "organization_id" uuid NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "airlines_organizations_airlines" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "airlines_iata_code_check" CHECK ("iata_code" ~ '^[A-Z0-9]{2}$'),
    CONSTRAINT "airlines_icao_code_check" CHECK ("icao_code" IS NULL OR "icao_code" ~ '^[A-Z0-9]{3}$'),
    CONSTRAINT "airlines_awb_prefix_check" CHECK ("awb_prefix" ~ '^[0-9]{3}$'),
    CONSTRAINT "airlines_country_code_check" CHECK ("country_code" ~ '^[A-Z]{2}$')
);
CREATE UNIQUE INDEX "airline_organization_id_iata_code" ON "airlines" ("organization_id", "iata_code");
CREATE UNIQUE INDEX "airline_organization_id_icao_code" ON "airlines" ("organization_id", "icao_code");
CREATE UNIQUE INDEX "airline_organization_id_awb_prefix" ON "airlines" ("organization_id", "awb_prefix");
CREATE INDEX "airline_organization_id_enabled_sort_order" ON "airlines" ("organization_id", "enabled", "sort_order");
CREATE INDEX "airline_updated_at" ON "airlines" ("updated_at");

CREATE TABLE "shipping_lines" (
    "id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "scac_code" varchar(4) NOT NULL,
    "name_zh" varchar(200) NOT NULL,
    "name_en" varchar(200) NOT NULL,
    "country_code" varchar(2) NOT NULL,
    "tracking_url" varchar(500) NULL,
    "alliance" varchar(100) NULL,
    "source" varchar(100) NOT NULL DEFAULT 'manual',
    "sort_order" bigint NOT NULL DEFAULT 100,
    "enabled" boolean NOT NULL DEFAULT true,
    "organization_id" uuid NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "shipping_lines_organizations_shipping_lines" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id"),
    CONSTRAINT "shipping_lines_scac_code_check" CHECK ("scac_code" ~ '^[A-Z]{2,4}$'),
    CONSTRAINT "shipping_lines_country_code_check" CHECK ("country_code" ~ '^[A-Z]{2}$')
);
CREATE UNIQUE INDEX "shippingline_organization_id_scac_code" ON "shipping_lines" ("organization_id", "scac_code");
CREATE INDEX "shippingline_organization_id_enabled_sort_order" ON "shipping_lines" ("organization_id", "enabled", "sort_order");
CREATE INDEX "shippingline_updated_at" ON "shipping_lines" ("updated_at");

CREATE TABLE "shipping_line_container_prefixes" (
    "id" uuid NOT NULL,
    "created_at" timestamptz NOT NULL,
    "updated_at" timestamptz NOT NULL,
    "prefix" varchar(4) NOT NULL,
    "organization_id" uuid NOT NULL,
    "shipping_line_id" uuid NOT NULL,
    PRIMARY KEY ("id"),
    CONSTRAINT "shipping_line_container_prefixes_shipping_lines_container_prefixes" FOREIGN KEY ("shipping_line_id") REFERENCES "shipping_lines" ("id"),
    CONSTRAINT "shipping_line_container_prefixes_prefix_check" CHECK ("prefix" ~ '^[A-Z]{3}[UJZ]$')
);
CREATE UNIQUE INDEX "shippinglinecontainerprefix_organization_id_prefix" ON "shipping_line_container_prefixes" ("organization_id", "prefix");
CREATE UNIQUE INDEX "shippinglinecontainerprefix_shipping_line_id_prefix" ON "shipping_line_container_prefixes" ("shipping_line_id", "prefix");
CREATE INDEX "shippinglinecontainerprefix_updated_at" ON "shipping_line_container_prefixes" ("updated_at");

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM "master_data_items"
        WHERE "kind" = 'port'
          AND (
              "code" !~ '^[A-Z]{2}[A-Z0-9]{3}$'
              OR NULLIF(trim("name_en"), '') IS NULL
              OR COALESCE("attributes" ->> 'country_code', '') !~ '^[A-Z]{2}$'
              OR left("code", 2) <> "attributes" ->> 'country_code'
              OR NOT COALESCE("attributes" -> 'modes', '[]'::jsonb) ? 'PORT'
              OR COALESCE("attributes" -> 'modes', '[]'::jsonb) ? 'AIRPORT'
          )
    ) THEN
        RAISE EXCEPTION '旧港口主数据不符合新模型约束，请先修正标准码、中英文名、国家码和运输类型';
    END IF;

    IF EXISTS (
        SELECT 1 FROM "master_data_items"
        WHERE "kind" = 'airport'
          AND (
              "code" !~ '^[A-Z]{3}$'
              OR NULLIF(trim("name_en"), '') IS NULL
              OR NULLIF(trim("attributes" ->> 'city_name'), '') IS NULL
              OR COALESCE("attributes" ->> 'country_code', '') !~ '^[A-Z]{2}$'
              OR (NULLIF(trim("attributes" ->> 'icao_code'), '') IS NOT NULL AND "attributes" ->> 'icao_code' !~ '^[A-Z0-9]{4}$')
          )
    ) THEN
        RAISE EXCEPTION '旧机场主数据不符合新模型约束，请先修正 IATA、ICAO、中英文名、城市和国家码';
    END IF;

    IF EXISTS (
        SELECT 1 FROM "master_data_items"
        WHERE "kind" = 'carrier'
          AND (
              NULLIF(trim("name_en"), '') IS NULL
              OR COALESCE("attributes" ->> 'country_code', '') !~ '^[A-Z]{2}$'
              OR "transport_mode" NOT IN ('AIR', 'SEA')
              OR ("transport_mode" = 'AIR' AND ("code" !~ '^[A-Z0-9]{2}$' OR COALESCE("attributes" ->> 'awb_prefix', '') !~ '^[0-9]{3}$'))
              OR ("transport_mode" = 'SEA' AND (COALESCE("attributes" ->> 'scac_code', '') !~ '^[A-Z]{2,4}$' OR ("code" <> "attributes" ->> 'scac_code' AND "code" !~ '^[A-Z]{3}[UJZ]$')))
          )
    ) THEN
        RAISE EXCEPTION '旧承运人主数据不符合新模型约束，请先修正运输方式及行业标准码';
    END IF;
END $$;

INSERT INTO "ports" ("id", "created_at", "updated_at", "un_locode", "name_zh", "name_en", "country_code", "transport_modes", "source", "sort_order", "enabled", "organization_id")
SELECT "id", "created_at", "updated_at", "code", "name", "name_en", "attributes" ->> 'country_code',
       (SELECT jsonb_agg(CASE value WHEN 'PORT' THEN 'SEA' ELSE value END) FROM jsonb_array_elements_text("attributes" -> 'modes')),
       "source", "sort_order", "enabled", "organization_id"
FROM "master_data_items" WHERE "kind" = 'port';

INSERT INTO "airports" ("id", "created_at", "updated_at", "iata_code", "icao_code", "name_zh", "name_en", "city_name_zh", "country_code", "source", "sort_order", "enabled", "organization_id")
SELECT "id", "created_at", "updated_at", "code", NULLIF(trim("attributes" ->> 'icao_code'), ''), "name", "name_en", "attributes" ->> 'city_name', "attributes" ->> 'country_code', "source", "sort_order", "enabled", "organization_id"
FROM "master_data_items" WHERE "kind" = 'airport';

INSERT INTO "airlines" ("id", "created_at", "updated_at", "iata_code", "awb_prefix", "name_zh", "name_en", "country_code", "cargo_only", "source", "sort_order", "enabled", "organization_id")
SELECT "id", "created_at", "updated_at", "code", "attributes" ->> 'awb_prefix', "name", "name_en", "attributes" ->> 'country_code', COALESCE(("attributes" ->> 'is_cargo_only')::boolean, false), "source", "sort_order", "enabled", "organization_id"
FROM "master_data_items" WHERE "kind" = 'carrier' AND "transport_mode" = 'AIR';

INSERT INTO "shipping_lines" ("id", "created_at", "updated_at", "scac_code", "name_zh", "name_en", "country_code", "tracking_url", "alliance", "source", "sort_order", "enabled", "organization_id")
SELECT "id", "created_at", "updated_at", "attributes" ->> 'scac_code', "name", "name_en", "attributes" ->> 'country_code', NULLIF(trim("attributes" ->> 'tracking_url'), ''), NULLIF(trim("attributes" ->> 'alliance'), ''), "source", "sort_order", "enabled", "organization_id"
FROM "master_data_items" WHERE "kind" = 'carrier' AND "transport_mode" = 'SEA';

INSERT INTO "shipping_line_container_prefixes" ("id", "created_at", "updated_at", "prefix", "organization_id", "shipping_line_id")
SELECT gen_random_uuid(), NOW(), NOW(), "code", "organization_id", "id"
FROM "master_data_items"
WHERE "kind" = 'carrier' AND "transport_mode" = 'SEA' AND "code" <> "attributes" ->> 'scac_code';

DELETE FROM "master_data_items" WHERE "kind" IN ('port', 'airport', 'carrier');

ALTER TABLE "master_data_items" DROP CONSTRAINT IF EXISTS "master_data_items_kind_check";
ALTER TABLE "master_data_items" ADD CONSTRAINT "master_data_items_kind_check"
    CHECK ("kind" IN ('currency', 'country', 'region', 'container_spec', 'service_type', 'cargo_category', 'abnormal_case'));
ALTER TABLE "master_data_items" DROP COLUMN "transport_mode";
