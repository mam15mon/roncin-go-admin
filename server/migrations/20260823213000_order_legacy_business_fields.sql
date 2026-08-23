ALTER TABLE "orders"
    ADD COLUMN "internal_reference_no" character varying NULL,
    ADD COLUMN "shipping_agent_id" uuid NULL,
    ADD COLUMN "insurance_premium" character varying NULL,
    ADD COLUMN "insurance_currency" character varying NULL,
    ADD COLUMN "un_number" character varying NULL,
    ADD COLUMN "hazard_class" character varying NULL,
    ADD COLUMN "factory_name" character varying NULL,
    ADD COLUMN "cargo_ready_at" character varying NULL,
    ADD COLUMN "loading_terms" character varying NULL;
