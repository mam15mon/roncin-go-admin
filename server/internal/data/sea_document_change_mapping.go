package data

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seadocumentvoideventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentvoidevent"
)

func masterVersionToBiz(v *ent.SeaMasterBillVersion, orderID uuid.UUID) *biz.SeaDocumentVersion {
	if v == nil {
		return nil
	}
	result := &biz.SeaDocumentVersion{ID: v.ID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: v.MasterBillID, OrderID: orderID, MasterBillID: v.MasterBillID, VersionNo: v.VersionNo, SourceEntityVersion: v.SourceEntityVersion, DocumentNo: v.MasterNo, NormalizedDocumentNo: v.NormalizedMasterNo, Status: string(v.Status), Source: string(v.Source), Reason: v.Reason, ShippingLineID: &v.ShippingLineID, Content: versionContent(v.ShipperText, v.ConsigneeText, v.NotifyPartyText, v.SecondNotifyPartyText, v.MarksText, v.GoodsDescriptionText, v.PackageCount, v.PackageUnit, v.GrossWeightKg, v.VolumeCbm, v.FreightTerms, v.TransportTerms, v.BillForm, v.ReleaseType, v.Clauses, v.ForeignAgentText), CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, Confirmation: externalConfirmationFromVersion(v.ConfirmedByParty, v.ConfirmedAt, v.ConfirmationNote, v.ConfirmationAttachmentID)}
	if line := v.Edges.ShippingLine; line != nil {
		result.ShippingLineName = formatShippingLineName(line.NameZh, line.NameEn, line.ScacCode)
	}
	return result
}
func houseVersionToBiz(v *ent.SeaHouseBillVersion) *biz.SeaDocumentVersion {
	if v == nil {
		return nil
	}
	return &biz.SeaDocumentVersion{ID: v.ID, DocumentType: biz.SeaDocumentTypeHouseBill, DocumentID: v.HouseBillID, OrderID: v.OrderID, MasterBillID: v.MasterBillID, VersionNo: v.VersionNo, SourceEntityVersion: v.SourceEntityVersion, DocumentNo: v.HouseNo, NormalizedDocumentNo: v.NormalizedHouseNo, Status: string(v.Status), Source: string(v.Source), Reason: v.Reason, IssuerPartnerID: v.IssuerPartnerID, IssuerOrganizationID: v.IssuerOrganizationID, IssuerSource: biz.SeaHouseBillIssuerSource(v.IssuerSource), Note: v.Note, Content: versionContent(v.ShipperText, v.ConsigneeText, v.NotifyPartyText, v.SecondNotifyPartyText, v.MarksText, v.GoodsDescriptionText, v.PackageCount, v.PackageUnit, v.GrossWeightKg, v.VolumeCbm, v.FreightTerms, v.TransportTerms, v.BillForm, v.ReleaseType, v.Clauses, v.ForeignAgentText), CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt, Confirmation: externalConfirmationFromVersion(v.ConfirmedByParty, v.ConfirmedAt, v.ConfirmationNote, v.ConfirmationAttachmentID)}
}

func externalConfirmationFromVersion(party *string, confirmedAt *time.Time, note *string, attachmentID *uuid.UUID) *biz.SeaExternalConfirmation {
	if party == nil || confirmedAt == nil || note == nil {
		return nil
	}
	return &biz.SeaExternalConfirmation{ConfirmedByParty: *party, ConfirmedAt: *confirmedAt, ConfirmationNote: *note, ConfirmationAttachmentID: attachmentID}
}
func versionContent(shipper, consignee, notify, secondNotify, marks, goods *string, packages *int, unit *string, weight, volume *float64, freight, transport, form, release, clauses, foreignAgent *string) *biz.SeaBillContent {
	var count *int32
	if packages != nil {
		v := int32(*packages)
		count = &v
	}
	return &biz.SeaBillContent{ShipperText: shipper, ConsigneeText: consignee, NotifyPartyText: notify, SecondNotifyPartyText: secondNotify, MarksText: marks, GoodsDescriptionText: goods, PackageCount: count, PackageUnit: unit, GrossWeightKg: weight, VolumeCbm: volume, FreightTerms: freight, TransportTerms: transport, BillForm: form, ReleaseType: release, Clauses: clauses, ForeignAgentText: foreignAgent}
}
func documentStringPointer(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}
func amendmentEventFromVersion(v *biz.SeaDocumentVersion) *biz.SeaDocumentEvent {
	id, no := v.DocumentID, v.DocumentNo
	return &biz.SeaDocumentEvent{ID: v.ID, EventType: biz.SeaDocumentEventTypeAmendment, DocumentType: v.DocumentType, DocumentID: &id, DocumentNo: &no, ResultVersionID: &v.ID, Reason: derefString(v.Reason), CreatedBy: v.CreatedBy, CreatedAt: v.CreatedAt}
}
func voidEventToBiz(v *ent.SeaDocumentVoidEvent) *biz.SeaDocumentEvent {
	if v == nil {
		return nil
	}
	t := biz.SeaDocumentTypeHouseBill
	docID := v.HouseBillID
	prev := v.PreviousHouseBillVersionID
	result := v.HouseBillVersionID
	if v.DocumentType == seadocumentvoideventent.DocumentTypeMASTER {
		t = biz.SeaDocumentTypeMasterBill
		docID = v.MasterBillID
		prev = v.PreviousMasterBillVersionID
		result = v.MasterBillVersionID
	}
	var documentNo *string
	if v.Edges.MasterBillVersion != nil {
		documentNo = &v.Edges.MasterBillVersion.MasterNo
	}
	if v.Edges.HouseBillVersion != nil {
		documentNo = &v.Edges.HouseBillVersion.HouseNo
	}
	return &biz.SeaDocumentEvent{ID: v.ID, EventType: biz.SeaDocumentEventTypeVoid, DocumentType: t, DocumentID: docID, DocumentNo: documentNo, PreviousVersionID: prev, ResultVersionID: result, Reason: v.Reason, ImpactSummary: v.ImpactSummary, CreatedBy: &v.CreatedBy, CreatedAt: v.CreatedAt, Confirmation: externalConfirmationFromVoidEvent(v)}
}

func modeChangeEventToBiz(v *ent.SeaDocumentModeChangeEvent) *biz.SeaDocumentEvent {
	if v == nil {
		return nil
	}
	previous := biz.SeaDocumentStructure(v.PreviousMode)
	target := biz.SeaDocumentStructure(v.TargetMode)
	return &biz.SeaDocumentEvent{
		ID: v.ID, EventType: biz.SeaDocumentEventTypeModeChange, DocumentType: biz.SeaDocumentTypeHouseBill,
		Reason: v.Reason, ImpactSummary: v.ImpactSummary, PreviousMode: &previous, TargetMode: &target,
		CreatedBy: &v.CreatedBy, CreatedAt: v.CreatedAt,
		Confirmation: &biz.SeaExternalConfirmation{ConfirmedByParty: v.ConfirmedByParty, ConfirmedAt: v.ConfirmedAt, ConfirmationNote: v.ConfirmationNote, ConfirmationAttachmentID: v.ConfirmationAttachmentID},
	}
}

func externalConfirmationFromVoidEvent(v *ent.SeaDocumentVoidEvent) *biz.SeaExternalConfirmation {
	if v == nil {
		return nil
	}
	return &biz.SeaExternalConfirmation{ConfirmedByParty: v.ConfirmedByParty, ConfirmedAt: v.ConfirmedAt, ConfirmationNote: v.ConfirmationNote, ConfirmationAttachmentID: v.ConfirmationAttachmentID}
}

func diffHouseVersionToInput(base *biz.SeaDocumentVersion, input *biz.SeaHouseBillInput, issuerOrgID, issuerPartnerID *uuid.UUID) []*biz.SeaDocumentFieldDifference {
	result := []*biz.SeaDocumentFieldDifference{}
	addDiff(&result, "house_no", "HBL 号", base.DocumentNo, input.HouseNo)
	addDiff(&result, "issuer_source", "签发主体来源", string(base.IssuerSource), string(input.IssuerSource))
	beforeIssuerID := uuidValue(base.IssuerPartnerID)
	if base.IssuerOrganizationID != nil {
		beforeIssuerID = base.IssuerOrganizationID.String()
	}
	afterIssuerID := uuidValue(issuerPartnerID)
	if issuerOrgID != nil {
		afterIssuerID = issuerOrgID.String()
	}
	addDiff(&result, "issuer_identity_id", "签发主体", beforeIssuerID, afterIssuerID)
	addDiff(&result, "note", "备注", documentStringValue(base.Note), documentStringValue(input.Note))
	return append(result, diffContent(base.Content, input.Content)...)
}
func diffContent(before, after *biz.SeaBillContent) []*biz.SeaDocumentFieldDifference {
	if before == nil {
		before = &biz.SeaBillContent{}
	}
	if after == nil {
		after = &biz.SeaBillContent{}
	}
	result := []*biz.SeaDocumentFieldDifference{}
	pairs := []struct{ key, label, before, after string }{{"shipper_text", "发货人", documentStringValue(before.ShipperText), documentStringValue(after.ShipperText)}, {"consignee_text", "收货人", documentStringValue(before.ConsigneeText), documentStringValue(after.ConsigneeText)}, {"notify_party_text", "通知人", documentStringValue(before.NotifyPartyText), documentStringValue(after.NotifyPartyText)}, {"second_notify_party_text", "第二通知人", documentStringValue(before.SecondNotifyPartyText), documentStringValue(after.SecondNotifyPartyText)}, {"marks_text", "唛头", documentStringValue(before.MarksText), documentStringValue(after.MarksText)}, {"goods_description_text", "货描", documentStringValue(before.GoodsDescriptionText), documentStringValue(after.GoodsDescriptionText)}, {"package_count", "件数", int32Value(before.PackageCount), int32Value(after.PackageCount)}, {"package_unit", "包装单位", documentStringValue(before.PackageUnit), documentStringValue(after.PackageUnit)}, {"gross_weight_kg", "毛重", floatValue(before.GrossWeightKg), floatValue(after.GrossWeightKg)}, {"volume_cbm", "体积", floatValue(before.VolumeCbm), floatValue(after.VolumeCbm)}, {"freight_terms", "运费条款", documentStringValue(before.FreightTerms), documentStringValue(after.FreightTerms)}, {"transport_terms", "运输条款", documentStringValue(before.TransportTerms), documentStringValue(after.TransportTerms)}, {"bill_form", "提单形式", documentStringValue(before.BillForm), documentStringValue(after.BillForm)}, {"release_type", "放单方式", documentStringValue(before.ReleaseType), documentStringValue(after.ReleaseType)}, {"clauses", "特别条款", documentStringValue(before.Clauses), documentStringValue(after.Clauses)}, {"foreign_agent_text", "外国代理", documentStringValue(before.ForeignAgentText), documentStringValue(after.ForeignAgentText)}}
	for _, p := range pairs {
		addDiff(&result, p.key, p.label, p.before, p.after)
	}
	return result
}
func addDiff(result *[]*biz.SeaDocumentFieldDifference, key, label, before, after string) {
	if before != after {
		*result = append(*result, &biz.SeaDocumentFieldDifference{Field: key, Label: label, BeforeValue: before, AfterValue: after})
	}
}
func documentStringValue(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func uuidValue(v *uuid.UUID) string {
	if v == nil {
		return ""
	}
	return v.String()
}
func int32Value(v *int32) string {
	if v == nil {
		return ""
	}
	return strconv.FormatInt(int64(*v), 10)
}
func floatValue(v *float64) string {
	if v == nil {
		return ""
	}
	return strconv.FormatFloat(*v, 'f', -1, 64)
}
func changeFingerprint(v any) string {
	raw, _ := json.Marshal(v)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
func impactSummary(items []*biz.SeaDocumentDownstreamImpact) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, item.FactType+":"+item.ReferenceNo)
	}
	return strings.Join(parts, "；")
}
func impactError(base *kratoserrors.Error, items []*biz.SeaDocumentDownstreamImpact) error {
	metadata := map[string]string{"blocked_count": strconv.Itoa(len(items))}
	if len(items) > 0 {
		metadata["fact_type"] = items[0].FactType
		metadata["reference_id"] = items[0].ReferenceID
		metadata["reference_no"] = items[0].ReferenceNo
	}
	return biz.MetadataError(base, metadata)
}
