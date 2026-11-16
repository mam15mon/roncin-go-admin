-- 进口本票固定服务与不可变操作历史；订单版本是唯一并发基线。
CREATE TABLE "order_import_handlings" (
 "id" uuid NOT NULL,
 "state" jsonb NOT NULL,
 "order_id" uuid NOT NULL,
 PRIMARY KEY ("id"),
 CONSTRAINT "order_import_handlings_orders_order" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION
);
CREATE UNIQUE INDEX "orderimporthandling_order_id" ON "order_import_handlings" ("order_id");
CREATE TABLE "order_import_handling_events" (
 "id" uuid NOT NULL,
 "actor_id" uuid NOT NULL,
 "occurred_at" timestamptz NOT NULL,
 "reason" varchar(500) NOT NULL DEFAULT '',
 "before" jsonb NOT NULL,
 "after" jsonb NOT NULL,
 "order_id" uuid NOT NULL,
 PRIMARY KEY ("id"),
 CONSTRAINT "order_import_handling_events_users_actor" FOREIGN KEY ("actor_id") REFERENCES "users" ("id") ON DELETE NO ACTION,
 CONSTRAINT "order_import_handling_events_orders_order" FOREIGN KEY ("order_id") REFERENCES "orders" ("id") ON DELETE NO ACTION
);
CREATE INDEX "orderimporthandlingevent_order_id_occurred_at" ON "order_import_handling_events" ("order_id", "occurred_at");

ALTER TABLE "orders" DROP CONSTRAINT "orders_flow_status_check";
ALTER TABLE "orders" ADD CONSTRAINT "orders_flow_status_check" CHECK (flow_status IN ('DRAFT', 'BOOKED', 'SPACE_ALLOCATED', 'TRUCKING_ARRANGED', 'DOCUMENT_CUTOFF', 'CUSTOMS_DECLARATION_ARRANGED', 'DOCUMENT_RELEASED', 'IMPORT_NOT_APPLICABLE', 'IMPORT_NOT_STARTED', 'IMPORT_IN_PROGRESS', 'IMPORT_COMPLETED'));
-- 既有进口尚无承接记录，不从出口状态猜测完成事实。
INSERT INTO "order_import_handlings" (id, order_id, state)
 SELECT id, id, jsonb_build_object('steps', '[{"kind":"ARRIVAL","undertaken":false,"referenceNo":"","notes":"","attachmentIds":[]},{"kind":"EXCHANGE","undertaken":false,"referenceNo":"","notes":"","attachmentIds":[]},{"kind":"CUSTOMS","undertaken":false,"referenceNo":"","notes":"","attachmentIds":[]},{"kind":"PICKUP","undertaken":false,"referenceNo":"","notes":"","attachmentIds":[]},{"kind":"DELIVERY","undertaken":false,"referenceNo":"","notes":"","attachmentIds":[]}]'::jsonb)
 FROM orders WHERE business_type IN ('SI', 'AI');
UPDATE orders SET flow_status = 'IMPORT_NOT_APPLICABLE' WHERE business_type IN ('SI', 'AI');
ALTER TABLE "orders" ADD CONSTRAINT "orders_import_flow_applicability_check" CHECK ((business_type IN ('SI', 'AI') AND flow_status IN ('IMPORT_NOT_APPLICABLE', 'IMPORT_NOT_STARTED', 'IMPORT_IN_PROGRESS', 'IMPORT_COMPLETED')) OR (business_type NOT IN ('SI', 'AI') AND flow_status NOT IN ('IMPORT_NOT_APPLICABLE', 'IMPORT_NOT_STARTED', 'IMPORT_IN_PROGRESS', 'IMPORT_COMPLETED')));

-- SI必须与SE同样固化真实海运主单/航程引用；未知旧快照明确停止迁移。
ALTER TABLE "order_lock_records" DROP CONSTRAINT "order_lock_records_business_type_document_refs_check";
ALTER TABLE "order_lock_records" ADD CONSTRAINT "order_lock_records_business_type_document_refs_check" CHECK ((business_type IN ('SE', 'SI') AND master_bill_id IS NOT NULL AND master_bill_version_id IS NOT NULL AND transport_execution_id IS NOT NULL AND transport_execution_version_id IS NOT NULL) OR (business_type IN ('AE', 'AI', 'LAND', 'RAIL') AND master_bill_id IS NULL AND master_bill_version_id IS NULL AND transport_execution_id IS NULL AND transport_execution_version_id IS NULL));
