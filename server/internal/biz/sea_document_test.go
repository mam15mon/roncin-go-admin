package biz

import (
	"context"
	"math"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeSeaHouseNo(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  string
		wantError bool
	}{
		{
			name:      "简单英文字符转大写",
			input:     "hbl-123456",
			expected:  "HBL-123456",
			wantError: false,
		},
		{
			name:      "去前后空格并转大写",
			input:     "   hbl-abc-789   ",
			expected:  "HBL-ABC-789",
			wantError: false,
		},
		{
			name:      "保留内部空格、标点、前导零、年份",
			input:     "  COSU 000123 / 2026.B  ",
			expected:  "COSU 000123 / 2026.B",
			wantError: false,
		},
		{
			name:      "包含非ASCII字符（中文/特殊符号）不被意外篡改",
			input:     "  海运分单-abc-001号 / 2026  ",
			expected:  "海运分单-ABC-001号 / 2026",
			wantError: false,
		},
		{
			name:      "Unicode NFC 规范化",
			input:     "e\u0301-123", // e + combining acute accent
			expected:  "é-123",       // only ascii a-z uppercased, é preserved in NFC
			wantError: false,
		},
		{
			name:      "空字符串报错",
			input:     "   ",
			expected:  "",
			wantError: true,
		},
		{
			name:      "超过128字符报错",
			input:     strings.Repeat("A", 129),
			expected:  "",
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			normalized, err := NormalizeSeaHouseNo(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatalf("NormalizeSeaHouseNo(%q) expected error, got nil", tc.input)
				}
			} else {
				if err != nil {
					t.Fatalf("NormalizeSeaHouseNo(%q) unexpected error: %v", tc.input, err)
				}
				if normalized != tc.expected {
					t.Fatalf("NormalizeSeaHouseNo(%q) = %q, want %q", tc.input, normalized, tc.expected)
				}
			}
		})
	}
}

func TestValidateSeaBillContent(t *testing.T) {
	pkgCount := int32(10)
	negPkgCount := int32(-1)
	gw := 100.5
	negGw := -1.0
	nanVal := math.NaN()
	infVal := math.Inf(1)
	cbm := 20.5
	negCbm := -0.5

	strPtr := func(s string) *string { return &s }

	t.Run("nil content is valid", func(t *testing.T) {
		res, err := ValidateSeaBillContent(nil)
		if err != nil || res != nil {
			t.Fatalf("expected (nil, nil), got (%v, %v)", res, err)
		}
	})

	t.Run("valid content trims text fields and persists normalized copies", func(t *testing.T) {
		c := &SeaBillContent{
			ShipperText:   strPtr("  Shipper ABC  "),
			ConsigneeText: strPtr("Consignee XYZ"),
			MarksText:     strPtr("   "),
			PackageCount:  &pkgCount,
			GrossWeightKg: &gw,
			VolumeCbm:     &cbm,
		}
		res, err := ValidateSeaBillContent(c)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.ShipperText == nil || *res.ShipperText != "Shipper ABC" {
			t.Fatalf("expected ShipperText to be trimmed to 'Shipper ABC', got %v", *res.ShipperText)
		}
		if res.MarksText != nil {
			t.Fatalf("expected whitespace-only MarksText to be nil, got %v", *res.MarksText)
		}
	})

	t.Run("negative package count fails", func(t *testing.T) {
		c := &SeaBillContent{
			PackageCount: &negPkgCount,
		}
		_, err := ValidateSeaBillContent(c)
		if err == nil {
			t.Fatalf("expected error for negative package count")
		}
	})

	t.Run("negative gross weight fails", func(t *testing.T) {
		c := &SeaBillContent{
			GrossWeightKg: &negGw,
		}
		_, err := ValidateSeaBillContent(c)
		if err == nil {
			t.Fatalf("expected error for negative gross weight")
		}
	})

	t.Run("NaN gross weight fails", func(t *testing.T) {
		c := &SeaBillContent{
			GrossWeightKg: &nanVal,
		}
		_, err := ValidateSeaBillContent(c)
		if err == nil {
			t.Fatalf("expected error for NaN gross weight")
		}
	})

	t.Run("Inf gross weight fails", func(t *testing.T) {
		c := &SeaBillContent{
			GrossWeightKg: &infVal,
		}
		_, err := ValidateSeaBillContent(c)
		if err == nil {
			t.Fatalf("expected error for Inf gross weight")
		}
	})

	t.Run("negative volume fails", func(t *testing.T) {
		c := &SeaBillContent{
			VolumeCbm: &negCbm,
		}
		_, err := ValidateSeaBillContent(c)
		if err == nil {
			t.Fatalf("expected error for negative volume")
		}
	})

	t.Run("excessive text length fails", func(t *testing.T) {
		c := &SeaBillContent{
			ShipperText: strPtr(strings.Repeat("A", MaxSeaPartyTextLength+1)),
		}
		_, err := ValidateSeaBillContent(c)
		if err == nil {
			t.Fatalf("expected error for shipper text > %d", MaxSeaPartyTextLength)
		}
	})
}

func TestValidateSeaOrderDocumentInput(t *testing.T) {
	strDirect := SeaDocumentStructureDirect
	strHouse := SeaDocumentStructureHouse
	ver := uint64(1)

	t.Run("UpdateOrder rejects non-empty HouseBills", func(t *testing.T) {
		input := &SeaOrderDocumentInput{
			HouseBills: []*SeaHouseBillInput{
				{HouseNo: "HBL1", IssuerSource: SeaHouseBillIssuerSourceSelfOrganization},
			},
		}
		_, err := ValidateSeaOrderDocumentInput(input, false)
		if err == nil {
			t.Fatalf("expected error for HouseBills in UpdateOrder, got nil")
		}
	})

	t.Run("UpdateOrder requires expected_link_version when changing structure", func(t *testing.T) {
		input := &SeaOrderDocumentInput{
			DocumentStructure: &strDirect,
		}
		_, err := ValidateSeaOrderDocumentInput(input, false)
		if err == nil {
			t.Fatalf("expected error for missing expected_link_version in UpdateOrder, got nil")
		}
	})

	t.Run("UpdateOrder requires expected_mbl_version when changing MBL content", func(t *testing.T) {
		s := "SHIPPER"
		input := &SeaOrderDocumentInput{
			MasterBillContent: &SeaBillContent{ShipperText: &s},
		}
		_, err := ValidateSeaOrderDocumentInput(input, false)
		if err == nil {
			t.Fatalf("expected error for missing expected_mbl_version in UpdateOrder, got nil")
		}
	})

	t.Run("CreateOrder rejects HOUSE structure with 0 HBLs", func(t *testing.T) {
		input := &SeaOrderDocumentInput{
			DocumentStructure: &strHouse,
			HouseBills:        []*SeaHouseBillInput{},
		}
		_, err := ValidateSeaOrderDocumentInput(input, true)
		if err == nil {
			t.Fatalf("expected error for HOUSE with 0 HBLs, got nil")
		}
	})

	t.Run("CreateOrder rejects DIRECT structure with HBLs", func(t *testing.T) {
		input := &SeaOrderDocumentInput{
			DocumentStructure: &strDirect,
			HouseBills: []*SeaHouseBillInput{
				{HouseNo: "HBL1", IssuerSource: SeaHouseBillIssuerSourceSelfOrganization},
			},
		}
		_, err := ValidateSeaOrderDocumentInput(input, true)
		if err != ErrSeaDocumentDirectAddHBLBlocked {
			t.Fatalf("expected ErrSeaDocumentDirectAddHBLBlocked, got %v", err)
		}
	})

	t.Run("UpdateOrder valid with expected versions", func(t *testing.T) {
		s := "SHIPPER"
		input := &SeaOrderDocumentInput{
			DocumentStructure:   &strDirect,
			ExpectedLinkVersion: &ver,
			MasterBillContent:   &SeaBillContent{ShipperText: &s},
			ExpectedMblVersion:  &ver,
		}
		res, err := ValidateSeaOrderDocumentInput(input, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil {
			t.Fatalf("expected non-nil result")
		}
	})
}

type seaDocumentRepoMock struct {
	getSeaOrderDocumentsFunc       func(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderDocuments, error)
	getSummariesByOrderIDsFunc     func(ctx context.Context, organizationID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]*SeaOrderDocumentSummary, error)
	markSeaOrderDirectFunc         func(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *AuditEvent) (*SeaOrderDocuments, error)
	cancelSeaOrderDirectFunc       func(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *AuditEvent) (*SeaOrderDocuments, error)
	addSeaHouseBillFunc            func(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error)
	updateSeaHouseBillFunc         func(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error)
	removeSeaHouseBillFunc         func(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, returnToUndetermined bool, audit *AuditEvent) error
	updateSeaMasterBillContentFunc func(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedMblVersion uint64, content *SeaBillContent, audit *AuditEvent) (*SeaMasterBillDetail, error)
}

func (m *seaDocumentRepoMock) GetSeaOrderDocuments(ctx context.Context, organizationID, orderID uuid.UUID) (*SeaOrderDocuments, error) {
	if m.getSeaOrderDocumentsFunc != nil {
		return m.getSeaOrderDocumentsFunc(ctx, organizationID, orderID)
	}
	return &SeaOrderDocuments{}, nil
}

func (m *seaDocumentRepoMock) GetSummariesByOrderIDs(ctx context.Context, organizationID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]*SeaOrderDocumentSummary, error) {
	if m.getSummariesByOrderIDsFunc != nil {
		return m.getSummariesByOrderIDsFunc(ctx, organizationID, orderIDs)
	}
	return make(map[uuid.UUID]*SeaOrderDocumentSummary), nil
}

func (m *seaDocumentRepoMock) MarkSeaOrderDirect(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *AuditEvent) (*SeaOrderDocuments, error) {
	if m.markSeaOrderDirectFunc != nil {
		return m.markSeaOrderDirectFunc(ctx, organizationID, actorID, orderID, expectedLinkVersion, audit)
	}
	return &SeaOrderDocuments{}, nil
}

func (m *seaDocumentRepoMock) CancelSeaOrderDirect(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, audit *AuditEvent) (*SeaOrderDocuments, error) {
	if m.cancelSeaOrderDirectFunc != nil {
		return m.cancelSeaOrderDirectFunc(ctx, organizationID, actorID, orderID, expectedLinkVersion, audit)
	}
	return &SeaOrderDocuments{}, nil
}

func (m *seaDocumentRepoMock) AddSeaHouseBill(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error) {
	if m.addSeaHouseBillFunc != nil {
		return m.addSeaHouseBillFunc(ctx, organizationID, actorID, orderID, expectedLinkVersion, input, audit)
	}
	return &SeaHouseBill{}, nil
}

func (m *seaDocumentRepoMock) UpdateSeaHouseBill(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error) {
	if m.updateSeaHouseBillFunc != nil {
		return m.updateSeaHouseBillFunc(ctx, organizationID, actorID, orderID, houseBillID, expectedVersion, expectedLinkVersion, input, audit)
	}
	return &SeaHouseBill{}, nil
}

func (m *seaDocumentRepoMock) RemoveSeaHouseBill(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, returnToUndetermined bool, audit *AuditEvent) error {
	if m.removeSeaHouseBillFunc != nil {
		return m.removeSeaHouseBillFunc(ctx, organizationID, actorID, orderID, houseBillID, expectedVersion, expectedLinkVersion, returnToUndetermined, audit)
	}
	return nil
}

func (m *seaDocumentRepoMock) UpdateSeaMasterBillContent(ctx context.Context, organizationID, actorID, orderID uuid.UUID, expectedMblVersion uint64, content *SeaBillContent, audit *AuditEvent) (*SeaMasterBillDetail, error) {
	if m.updateSeaMasterBillContentFunc != nil {
		return m.updateSeaMasterBillContentFunc(ctx, organizationID, actorID, orderID, expectedMblVersion, content, audit)
	}
	return &SeaMasterBillDetail{}, nil
}

func TestSeaDocumentUsecase_ValidationRules(t *testing.T) {
	orgID := uuid.New()
	actorID := uuid.New()
	orderID := uuid.New()
	validAudit := &AuditEvent{
		OrganizationID: &orgID,
		UserID:         &actorID,
		Result:         "success",
	}

	t.Run("AddHouseBill rejects empty HouseNo", func(t *testing.T) {
		repo := &seaDocumentRepoMock{}
		uc := NewSeaDocumentUsecase(repo)
		_, err := uc.AddSeaHouseBill(context.Background(), orgID, actorID, orderID, 1, &SeaHouseBillInput{
			HouseNo:      "   ",
			IssuerSource: SeaHouseBillIssuerSourceSelfOrganization,
		}, validAudit)
		if err == nil {
			t.Fatalf("expected error for empty house no, got nil")
		}
	})

	t.Run("AddHouseBill rejects other partner without ID", func(t *testing.T) {
		repo := &seaDocumentRepoMock{}
		uc := NewSeaDocumentUsecase(repo)
		_, err := uc.AddSeaHouseBill(context.Background(), orgID, actorID, orderID, 1, &SeaHouseBillInput{
			HouseNo:      "HBL001",
			IssuerSource: SeaHouseBillIssuerSourceOtherPartner,
		}, validAudit)
		if err == nil {
			t.Fatalf("expected error for missing other partner ID, got nil")
		}
	})

	for _, source := range []SeaHouseBillIssuerSource{
		SeaHouseBillIssuerSourceSelfOrganization,
		SeaHouseBillIssuerSourceCustomerPartner,
	} {
		t.Run("本公司或委托单位来源拒绝夹带其他合作伙伴ID_"+string(source), func(t *testing.T) {
			partnerID := uuid.New()
			_, err := ValidateSeaHouseBillInput(&SeaHouseBillInput{
				HouseNo:         "HBL001",
				IssuerSource:    source,
				IssuerPartnerID: &partnerID,
			})
			if err == nil {
				t.Fatal("SELF/CUSTOMER 来源夹带 issuer_partner_id 应明确拒绝")
			}
		})
	}

	t.Run("UpdateSeaHouseBill validates and delegates", func(t *testing.T) {
		var called bool
		repo := &seaDocumentRepoMock{
			updateSeaHouseBillFunc: func(ctx context.Context, organizationID, actorID, orderID, houseBillID uuid.UUID, expectedVersion, expectedLinkVersion uint64, input *SeaHouseBillInput, audit *AuditEvent) (*SeaHouseBill, error) {
				called = true
				return &SeaHouseBill{HouseNo: input.HouseNo}, nil
			},
		}
		uc := NewSeaDocumentUsecase(repo)
		hbID := uuid.New()
		res, err := uc.UpdateSeaHouseBill(context.Background(), orgID, actorID, orderID, hbID, 1, 1, &SeaHouseBillInput{
			HouseNo:      "  cosu-998877  ",
			IssuerSource: SeaHouseBillIssuerSourceSelfOrganization,
		}, validAudit)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !called || res == nil {
			t.Fatalf("expected delegate call")
		}
	})

	t.Run("Usecase rejects nil or mismatched audit", func(t *testing.T) {
		repo := &seaDocumentRepoMock{}
		uc := NewSeaDocumentUsecase(repo)
		_, err := uc.MarkSeaOrderDirect(context.Background(), orgID, actorID, orderID, 1, nil)
		if err != ErrSeaDocumentInvalidArgument {
			t.Fatalf("expected ErrSeaDocumentInvalidArgument for nil audit, got %v", err)
		}

		diffOrg := uuid.New()
		mismatchedAudit := &AuditEvent{
			OrganizationID: &diffOrg,
			UserID:         &actorID,
			Result:         "success",
		}
		_, err = uc.MarkSeaOrderDirect(context.Background(), orgID, actorID, orderID, 1, mismatchedAudit)
		if err != ErrSeaDocumentInvalidArgument {
			t.Fatalf("expected ErrSeaDocumentInvalidArgument for mismatched audit org, got %v", err)
		}
	})

	t.Run("Usecase rejects nil UUID parameters", func(t *testing.T) {
		repo := &seaDocumentRepoMock{}
		uc := NewSeaDocumentUsecase(repo)
		_, err := uc.GetSeaOrderDocuments(context.Background(), uuid.Nil, orderID)
		if err == nil {
			t.Fatalf("expected error for nil orgID")
		}
		_, err = uc.GetSeaOrderDocuments(context.Background(), orgID, uuid.Nil)
		if err == nil {
			t.Fatalf("expected error for nil orderID")
		}
		_, err = uc.MarkSeaOrderDirect(context.Background(), orgID, uuid.Nil, orderID, 1, validAudit)
		if err == nil {
			t.Fatalf("expected error for nil actorID")
		}
	})
}
