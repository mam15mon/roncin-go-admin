package migrate

import (
	"testing"

	"entgo.io/ent/dialect/sql/schema"
)

func TestGeneratedMigrateTables_CheckConstraints(t *testing.T) {
	cases := []struct {
		tableName      string
		table          *schema.Table
		expectedChecks map[string]string
	}{
		{
			tableName: "order_attachment_assets",
			table:     OrderAttachmentAssetsTable,
			expectedChecks: map[string]string{
				"order_attachment_assets_file_size_check": "file_size > 0",
			},
		},
		{
			tableName: "partner_accounts",
			table:     PartnerAccountsTable,
			expectedChecks: map[string]string{
				"partner_accounts_default_usage_check": "((NOT is_default_receivable OR (enabled AND usage IN ('RECEIVABLE', 'BOTH'))) AND (NOT is_default_payable OR (enabled AND usage IN ('PAYABLE', 'BOTH'))))",
			},
		},
		{
			tableName: "ding_talk_approval_dispatches",
			table:     DingTalkApprovalDispatchesTable,
			expectedChecks: map[string]string{
				"ding_talk_approval_dispatches_dispatch_status_check": "dispatch_status IN ('PENDING', 'SENDING', 'DISPATCHED', 'FAILED', 'UNKNOWN')",
			},
		},
		{
			tableName: "ding_talk_approval_inbox_events",
			table:     DingTalkApprovalInboxEventsTable,
			expectedChecks: map[string]string{
				"ding_talk_approval_inbox_events_status_check":   "status IN ('RECEIVED', 'PROCESSING', 'PROCESSED', 'IGNORED', 'FAILED')",
				"ding_talk_approval_inbox_events_attempts_check": "attempts >= 0",
			},
		},
		{
			tableName: "order_lifecycle_events",
			table:     OrderLifecycleEventsTable,
			expectedChecks: map[string]string{
				"order_lifecycle_events_dimension_check": "dimension IN ('FLOW', 'TERMINATION', 'CLOSURE', 'ORIGIN')",
			},
		},
		{
			tableName: "sea_order_split_results",
			table:     SeaOrderSplitResultsTable,
			expectedChecks: map[string]string{
				"sea_order_split_results_result_role_check": "result_role IN ('ORIGINAL', 'CREATED')",
			},
		},
		{
			tableName: "sea_order_reassignment_events",
			table:     SeaOrderReassignmentEventsTable,
			expectedChecks: map[string]string{
				"sea_order_reassignment_events_responsibility_type_check": "responsibility_type IN ('CARRIER', 'CUSTOMER', 'CUSTOMS', 'OWN_COMPANY', 'FORCE_MAJEURE', 'OTHER')",
			},
		},
		{
			tableName: "sea_document_void_events",
			table:     SeaDocumentVoidEventsTable,
			expectedChecks: map[string]string{
				"sea_document_void_events_document_type_check": "((document_type = 'MASTER' AND master_bill_id IS NOT NULL AND master_bill_version_id IS NOT NULL AND previous_master_bill_version_id IS NOT NULL AND house_bill_id IS NULL AND house_bill_version_id IS NULL AND previous_house_bill_version_id IS NULL) OR (document_type = 'HOUSE' AND house_bill_id IS NOT NULL AND house_bill_version_id IS NOT NULL AND previous_house_bill_version_id IS NOT NULL AND master_bill_id IS NULL AND master_bill_version_id IS NULL AND previous_master_bill_version_id IS NULL))",
				"sea_document_void_events_status_check":        "voided_status = 'VOIDED'",
			},
		},
		{
			tableName: "sea_house_bill_versions",
			table:     SeaHouseBillVersionsTable,
			expectedChecks: map[string]string{
				"sea_house_bill_versions_issuer_check": "((issuer_source = 'SELF_ORGANIZATION' AND issuer_organization_id IS NOT NULL AND issuer_partner_id IS NULL) OR (issuer_source IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER') AND issuer_organization_id IS NULL AND issuer_partner_id IS NOT NULL))",
			},
		},
		{
			tableName: "finance_nettings",
			table:     FinanceNettingsTable,
			expectedChecks: map[string]string{
				"financenetting_status_check":                     "status IN ('DRAFT', 'CONFIRMED', 'CANCELLED', 'REVERSED')",
				"financenetting_amount_positive":                  "amount > 0",
				"financenetting_base_amount_non_negative":         "base_currency_amount >= 0",
				"financenetting_payable_base_amount_non_negative": "payable_base_amount >= 0",
			},
		},
		{
			tableName: "finance_commission_adjustments",
			table:     FinanceCommissionAdjustmentsTable,
			expectedChecks: map[string]string{
				"commission_adjustment_amount_positive":         "amount > 0",
				"commission_adjustment_direction_check":         "direction IN ('INCREASE', 'DECREASE')",
				"commission_adjustment_source_supplement_check": "(source_type = 'LOCKED_FEE_SUPPLEMENT' AND source_fee_supplement_request_id IS NOT NULL) OR (source_type <> 'LOCKED_FEE_SUPPLEMENT' AND source_fee_supplement_request_id IS NULL)",
				"commission_adjustment_source_type_check":       "source_type IN ('MANUAL', 'VERIFICATION_REVERSAL', 'NETTING_REVERSAL', 'LOCKED_FEE_SUPPLEMENT')",
				"commission_adjustment_status_check":            "status IN ('DRAFT', 'CONFIRMED', 'PAID', 'CANCELLED')",
			},
		},
		{
			tableName: "finance_commission_lines",
			table:     FinanceCommissionLinesTable,
			expectedChecks: map[string]string{
				"finance_commission_lines_snapshot_consistency_check": "(snapshot_status IS NULL AND total_receivable_snapshot IS NULL AND total_payable_snapshot IS NULL AND snapshot_source IS NULL AND snapshot_backfill_version IS NULL AND snapshot_evidence_hash IS NULL AND snapshot_unavailable_reason_code IS NULL) OR (snapshot_status = 'READY' AND total_receivable_snapshot IS NOT NULL AND total_payable_snapshot IS NOT NULL AND snapshot_unavailable_reason_code IS NULL AND snapshot_source IS NOT NULL AND ((snapshot_source = 'NATIVE' AND snapshot_backfill_version IS NULL AND snapshot_evidence_hash IS NULL) OR (snapshot_source = 'MIGRATED' AND snapshot_backfill_version IS NOT NULL AND snapshot_evidence_hash IS NOT NULL))) OR (snapshot_status = 'UNAVAILABLE' AND total_receivable_snapshot IS NULL AND total_payable_snapshot IS NULL AND snapshot_source IS NULL AND snapshot_backfill_version IS NULL AND snapshot_evidence_hash IS NULL AND snapshot_unavailable_reason_code IS NOT NULL)",
			},
		},
		{
			tableName: "order_fee_supplement_requests",
			table:     OrderFeeSupplementRequestsTable,
			expectedChecks: map[string]string{
				"order_fee_supplement_requests_lock_basis_check": "(lock_basis = 'BUSINESS' AND business_lock_generation IS NOT NULL AND business_lock_generation > 0 AND financial_lock_evidence_version IS NULL AND financial_lock_evidence_hash IS NULL AND financial_lock_net_amount_snapshot IS NULL) OR (lock_basis = 'FINANCIAL' AND business_lock_generation IS NULL AND financial_lock_evidence_version IS NOT NULL AND financial_lock_evidence_hash IS NOT NULL AND financial_lock_net_amount_snapshot IS NOT NULL AND financial_lock_net_amount_snapshot > 0) OR (lock_basis = 'BOTH' AND business_lock_generation IS NOT NULL AND business_lock_generation > 0 AND financial_lock_evidence_version IS NOT NULL AND financial_lock_evidence_hash IS NOT NULL AND financial_lock_net_amount_snapshot IS NOT NULL AND financial_lock_net_amount_snapshot > 0)",
				"order_fee_supplement_requests_status_check":     "status IN ('PENDING', 'APPROVED', 'REJECTED', 'WITHDRAWN')",
			},
		},
		{
			tableName: "notification_deliveries",
			table:     NotificationDeliveriesTable,
			expectedChecks: map[string]string{
				"notification_deliveries_template_check": "template IN ('ORDER_PERSONNEL_ASSIGNED', 'USER_AUTHORIZED', 'DINGTALK_REGISTRATION_PENDING', 'DINGTALK_REGISTRATION_REJECTED', 'DINGTALK_INVITATION_ACTIVATED', 'EXCHANGE_RATE_WEEKLY_REMINDER', 'FEE_SUPPLEMENT_APPROVAL_PENDING', 'COMMISSION_DECREASE_SUGGESTED')",
			},
		},
		{
			tableName: "sea_document_mode_change_events",
			table:     SeaDocumentModeChangeEventsTable,
			expectedChecks: map[string]string{
				"sea_document_mode_change_events_mode_check": "previous_mode <> target_mode AND ((previous_mode = 'HOUSE' AND target_mode = 'DIRECT' AND previous_house_bill_id IS NOT NULL AND previous_house_bill_version_id IS NOT NULL AND target_house_bill_id IS NULL AND target_house_bill_version_id IS NULL) OR (previous_mode = 'DIRECT' AND target_mode = 'HOUSE' AND target_house_bill_id IS NOT NULL AND target_house_bill_version_id IS NOT NULL AND previous_house_bill_id IS NULL AND previous_house_bill_version_id IS NULL))",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.tableName, func(t *testing.T) {
			if tc.table.Annotation == nil {
				t.Fatalf("table %s has nil Annotation", tc.tableName)
			}
			if tc.table.Annotation.Checks == nil {
				t.Fatalf("table %s has nil Annotation.Checks", tc.tableName)
			}
			for name, wantExpr := range tc.expectedChecks {
				gotExpr, ok := tc.table.Annotation.Checks[name]
				if !ok {
					t.Errorf("table %s missing expected CHECK constraint %q", tc.tableName, name)
					continue
				}
				if gotExpr != wantExpr {
					t.Errorf("table %s CHECK constraint %q expr mismatch: got %q, want %q", tc.tableName, name, gotExpr, wantExpr)
				}
			}
		})
	}
}
