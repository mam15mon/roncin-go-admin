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
				"sea_document_void_events_document_type_check": "((document_type = 'MASTER' AND master_bill_id IS NOT NULL AND house_bill_id IS NULL) OR (document_type = 'HOUSE' AND house_bill_id IS NOT NULL AND master_bill_id IS NULL))",
			},
		},
		{
			tableName: "sea_house_bill_versions",
			table:     SeaHouseBillVersionsTable,
			expectedChecks: map[string]string{
				"sea_house_bill_versions_issuer_check": "((issuer_source = 'SELF_ORGANIZATION' AND issuer_organization_id IS NOT NULL AND issuer_partner_id IS NULL) OR (issuer_source IN ('CUSTOMER_PARTNER', 'OTHER_PARTNER') AND issuer_organization_id IS NULL AND issuer_partner_id IS NOT NULL))",
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
