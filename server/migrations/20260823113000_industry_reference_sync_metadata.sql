ALTER TABLE "ports"
    ALTER COLUMN "name_zh" DROP NOT NULL,
    ADD COLUMN "source_version" varchar(100) NULL,
    ADD COLUMN "source_hash" varchar(64) NULL;

ALTER TABLE "airports"
    ALTER COLUMN "name_zh" DROP NOT NULL,
    ALTER COLUMN "city_name_zh" DROP NOT NULL,
    ADD COLUMN "source_version" varchar(100) NULL,
    ADD COLUMN "source_hash" varchar(64) NULL;
