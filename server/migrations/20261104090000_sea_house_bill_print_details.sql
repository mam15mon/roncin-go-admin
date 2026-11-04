-- modify "sea_house_bill_versions" table
ALTER TABLE "sea_house_bill_versions" ADD COLUMN "print_details" jsonb NULL;
-- modify "sea_house_bills" table
ALTER TABLE "sea_house_bills" ADD COLUMN "print_details" jsonb NULL;
