package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
)

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func derefUUID(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func computeMBLContentHash(mbl *ent.SeaMasterBill) string {
	h := sha256.New()
	fmt.Fprintf(h, "no:%s|norm:%s|shipping_line:%s|", mbl.MasterNo, mbl.NormalizedMasterNo, mbl.ShippingLineID)
	fmt.Fprintf(h, "shipper:%s|consignee:%s|notify:%s|notify2:%s|marks:%s|goods:%s|",
		derefStr(mbl.ShipperText), derefStr(mbl.ConsigneeText), derefStr(mbl.NotifyPartyText),
		derefStr(mbl.SecondNotifyPartyText), derefStr(mbl.MarksText), derefStr(mbl.GoodsDescriptionText))
	fmt.Fprintf(h, "pkgs:%d|pkgunit:%s|gw:%.4f|vol:%.4f|freight:%s|trans_terms:%s|bill_form:%s|release:%s|clauses:%s|foreignAgent:%s|",
		derefInt(mbl.PackageCount), derefStr(mbl.PackageUnit), derefFloat(mbl.GrossWeightKg), derefFloat(mbl.VolumeCbm),
		derefStr(mbl.FreightTerms), derefStr(mbl.TransportTerms), derefStr(mbl.BillForm), derefStr(mbl.ReleaseType), derefStr(mbl.Clauses), derefStr(mbl.ForeignAgentText))
	return hex.EncodeToString(h.Sum(nil))
}

func computeTransportExecutionContentHash(exec *ent.SeaTransportExecution) string {
	h := sha256.New()
	fmt.Fprintf(h, "shipping_line:%s|", exec.ShippingLineID)
	var etdStr, etaStr string
	if exec.Etd != nil {
		etdStr = exec.Etd.Format(time.RFC3339)
	}
	if exec.Eta != nil {
		etaStr = exec.Eta.Format(time.RFC3339)
	}
	fmt.Fprintf(h, "vessel:%s|voyage:%s|etd:%s|eta:%s|origin:%s|discharge:%s|transit:%s|",
		exec.VesselName, exec.VoyageNo, etdStr, etaStr,
		derefUUID(exec.OriginLocationID), derefUUID(exec.DischargeLocationID), derefUUID(exec.TransitLocationID))
	return hex.EncodeToString(h.Sum(nil))
}

func computeHBLContentHash(hbl *ent.SeaHouseBill) string {
	h := sha256.New()
	fmt.Fprintf(h, "no:%s|norm:%s|source:%s|issuer_org:%s|issuer_partner:%s|note:%s|",
		hbl.HouseNo, hbl.NormalizedHouseNo, hbl.IssuerSource, derefUUID(hbl.IssuerOrganizationID), derefUUID(hbl.IssuerPartnerID), derefStr(hbl.Note))
	fmt.Fprintf(h, "shipper:%s|consignee:%s|notify:%s|notify2:%s|marks:%s|goods:%s|",
		derefStr(hbl.ShipperText), derefStr(hbl.ConsigneeText), derefStr(hbl.NotifyPartyText),
		derefStr(hbl.SecondNotifyPartyText), derefStr(hbl.MarksText), derefStr(hbl.GoodsDescriptionText))
	fmt.Fprintf(h, "pkgs:%d|pkgunit:%s|gw:%.4f|vol:%.4f|freight:%s|trans_terms:%s|bill_form:%s|release:%s|clauses:%s|foreignAgent:%s|",
		derefInt(hbl.PackageCount), derefStr(hbl.PackageUnit), derefFloat(hbl.GrossWeightKg), derefFloat(hbl.VolumeCbm),
		derefStr(hbl.FreightTerms), derefStr(hbl.TransportTerms), derefStr(hbl.BillForm), derefStr(hbl.ReleaseType), derefStr(hbl.Clauses), derefStr(hbl.ForeignAgentText))
	return hex.EncodeToString(h.Sum(nil))
}

func computeLockOrderFingerprint(organizationID, orderID uuid.UUID, expectedVersion uint64, callerID uuid.UUID) string {
	h := sha256.New()
	fmt.Fprintf(h, "org:%s|order:%s|ver:%d|caller:%s", organizationID, orderID, expectedVersion, callerID)
	return hex.EncodeToString(h.Sum(nil))
}

// computeAutoLockFingerprint 为自动锁定记录生成稳定请求指纹：同一触发单据对同一
// 订单的重复检查可被幂等约束收敛，不与人工幂等键空间重叠。
func computeAutoLockFingerprint(organizationID, orderID, resourceID uuid.UUID, triggerType string) string {
	h := sha256.New()
	fmt.Fprintf(h, "org:%s|order:%s|trigger:%s|resource:%s|auto:1", organizationID, orderID, triggerType, resourceID)
	return hex.EncodeToString(h.Sum(nil))
}

func computeRequestFingerprint(organizationID, orderID uuid.UUID, expectedVersion uint64, callerID uuid.UUID, reason *string) string {
	h := sha256.New()
	r := ""
	if reason != nil {
		r = *reason
	}
	fmt.Fprintf(h, "org:%s|order:%s|ver:%d|caller:%s|reason:%s", organizationID, orderID, expectedVersion, callerID, r)
	return hex.EncodeToString(h.Sum(nil))
}

func findOrderLockRecordByIdempotencyKey(ctx context.Context, client *ent.Client, organizationID uuid.UUID, idempotencyKey string) (*ent.OrderLockRecord, error) {
	record, err := client.OrderLockRecord.Query().
		Where(
			orderlockrecordent.OrganizationIDEQ(organizationID),
			orderlockrecordent.IdempotencyKeyEQ(idempotencyKey),
		).
		First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return record, err
}

func lockRecordMatchesRequest(record *ent.OrderLockRecord, orderID, callerID uuid.UUID, fingerprint string) bool {
	return record != nil && record.OrderID == orderID && record.LockedBy != nil && *record.LockedBy == callerID && record.RequestFingerprint == fingerprint
}

func findOrderUnlockRequestByIdempotencyKey(ctx context.Context, client *ent.Client, organizationID uuid.UUID, idempotencyKey string) (*ent.OrderUnlockRequest, error) {
	request, err := client.OrderUnlockRequest.Query().
		Where(
			orderunlockrequestent.OrganizationIDEQ(organizationID),
			orderunlockrequestent.IdempotencyKeyEQ(idempotencyKey),
		).
		First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return request, err
}

func unlockRequestMatchesRequest(request *ent.OrderUnlockRequest, orderID, callerID uuid.UUID, fingerprint string) bool {
	return request != nil && request.OrderID == orderID && request.RequestedBy == callerID && request.RequestFingerprint == fingerprint
}

func safeWriteAudit(ctx context.Context, client *ent.AuditLogClient, audit *biz.AuditEvent, organizationID, userID uuid.UUID) error {
	if audit == nil {
		return nil
	}
	if audit.Result == "" {
		audit.Result = "success"
	}
	if audit.OrganizationID == nil {
		audit.OrganizationID = &organizationID
	}
	if audit.UserID == nil {
		audit.UserID = &userID
	}
	return writeAudit(ctx, client, audit)
}
