ALTER TABLE "airlines"
    ALTER COLUMN "awb_prefix" DROP NOT NULL,
    ALTER COLUMN "name_zh" DROP NOT NULL,
    ADD COLUMN "source_version" varchar(100) NULL,
    ADD COLUMN "source_hash" varchar(64) NULL;
