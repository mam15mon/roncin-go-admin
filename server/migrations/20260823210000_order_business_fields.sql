ALTER TABLE "orders"
    ADD COLUMN "customer_reference_no" character varying NULL,
    ADD COLUMN "foreign_agent_id" uuid NULL,
    ADD COLUMN "contract_no" character varying NULL,
    ADD COLUMN "cargo_value" character varying NULL,
    ADD COLUMN "cargo_currency" character varying NULL;
