CREATE UNIQUE INDEX "verification_allocation_pair_unique"
  ON "finance_verification_allocations"("verification_id", "cashflow_id", "bill_id");
