package data

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	airhouse "github.com/roncin/roncin-go-admin/server/internal/data/ent/airhousewaybill"
	airlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

func prepareAirOrderCreate(ctx context.Context, tx *ent.Tx, org, actor uuid.UUID, input *biz.Order) (*ent.AirMasterWaybill, error) {
	if input.BusinessType != biz.OrderBusinessAE {
		return nil, nil
	}
	in := input.AirExportInput
	if in == nil || in.Master == nil {
		return nil, biz.ErrAirWaybillInvalid
	}
	var m *ent.AirMasterWaybill
	if in.MasterWaybillID != nil {
		locked, e := lockAirMasterMembers(ctx, tx, org, *in.MasterWaybillID, nil)
		if e != nil {
			return nil, e
		}
		if locked.master.Version != in.ExpectedMasterVersion {
			return nil, biz.ErrAirWaybillConflict
		}
		if e = ensureAirMembersEditable(locked); e != nil {
			return nil, e
		}
		a, _ := json.Marshal(in.Master)
		b, _ := json.Marshal(airMasterToBiz(locked.master).Content)
		if string(a) != string(b) {
			return nil, biz.ErrAirWaybillConflict
		}
		m = locked.master
		if e = validateAirMasterReferences(ctx, tx, &airMasterToBiz(m).Content); e != nil {
			return nil, e
		}
	} else {
		v := in.Master
		if e := validateAirMasterReferences(ctx, tx, v); e != nil {
			return nil, e
		}
		var e error
		m, e = tx.AirMasterWaybill.Create().SetOrganizationID(org).SetMasterNo(v.MasterNo).SetAirlineID(v.AirlineID).SetOriginAirportID(v.OriginAirportID).SetDestinationAirportID(v.DestinationAirportID).SetFlightNo(v.FlightNo).SetEtd(v.ETD).SetEta(v.ETA).SetActualDepartureAt(v.ActualDepartureAt).SetActualArrivalAt(v.ActualArrivalAt).SetDocumentCutoffAt(v.DocumentCutoffAt).SetCustomsCutoffAt(v.CustomsCutoffAt).SetStatus(v.Status).SetNillableDeclaredPackages(v.DeclaredPackages).SetNillableDeclaredGrossWeightKg(v.DeclaredGrossWeightKg).SetNillableDeclaredVolumeCbm(v.DeclaredVolumeCbm).SetNotes(v.Notes).Save(ctx)
		if e != nil {
			return nil, mapEntError(e, nil, biz.ErrAirWaybillExists)
		}
	}
	input.OriginLocationID = &m.OriginAirportID
	input.DestinationLocationID = &m.DestinationAirportID
	input.ETD = m.Etd
	input.ETA = m.Eta
	input.DocCutoff = m.DocumentCutoffAt
	input.CustomsCutoff = m.CustomsCutoffAt
	return m, nil
}
func createAirOrderDocuments(ctx context.Context, tx *ent.Tx, org, actor uuid.UUID, o *ent.Order, input *biz.Order, m *ent.AirMasterWaybill) error {
	if input.BusinessType != biz.OrderBusinessAE {
		return nil
	}
	in := input.AirExportInput
	if _, e := tx.AirMasterWaybillOrderLink.Create().SetOrganizationID(org).SetMasterWaybillID(m.ID).SetOrderID(o.ID).Save(ctx); e != nil {
		return mapEntError(e, nil, biz.ErrAirWaybillExists)
	}
	h, e := tx.AirHouseWaybill.Create().SetOrganizationID(org).SetMasterWaybillID(m.ID).SetOrderID(o.ID).SetHouseNo(in.HouseNo).SetContent(in.HouseContent).SetStatus(in.HouseStatus).Save(ctx)
	if e != nil {
		return mapEntError(e, nil, biz.ErrAirWaybillExists)
	}
	if e = writeAirHouseVersion(ctx, tx, h, actor, "创建分运单"); e != nil {
		return e
	}
	if in.MasterWaybillID != nil {
		m, e = m.Update().SetVersion(m.Version + 1).Save(ctx)
		if e != nil {
			return e
		}
	}
	return writeAirMasterHistory(ctx, tx, m, actor, "CREATE_LINK", "创建空运订单 "+o.OrderNo)
}
func updateAirOrderDocuments(ctx context.Context, tx *ent.Tx, org, actor uuid.UUID, o *ent.Order, input *biz.Order) error {
	if input.BusinessType != biz.OrderBusinessAE {
		return nil
	}
	if input.AirExportInput == nil {
		return biz.ErrAirWaybillInvalid
	}
	in := input.AirExportInput
	l, e := tx.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(o.ID), airlink.OrganizationIDEQ(org), airlink.StatusEQ("ACTIVE")).Only(ctx)
	if e != nil {
		return biz.ErrAirWaybillConflict
	}
	if in.ExpectedLinkVersion == 0 || in.ExpectedLinkVersion != l.Version {
		return biz.ErrAirWaybillConflict
	}
	if in.MasterWaybillID != nil && *in.MasterWaybillID != l.MasterWaybillID {
		return biz.ErrAirWaybillInvalid
	}
	h, e := tx.AirHouseWaybill.Query().Where(airhouse.OrderIDEQ(o.ID), airhouse.MasterWaybillIDEQ(l.MasterWaybillID)).ForUpdate().Only(ctx)
	if e != nil {
		return e
	}
	if h.HouseNo != in.HouseNo {
		return biz.ErrAirWaybillInvalid
	}
	if h.Content == in.HouseContent && h.Status == in.HouseStatus {
		return nil
	}
	if h.Status == "RELEASED" && strings.TrimSpace(in.ChangeReason) == "" {
		return biz.NewAirFlowDataError("已签发分单修改请填写改单原因")
	}
	h, e = h.Update().SetContent(in.HouseContent).SetStatus(in.HouseStatus).SetVersion(h.Version + 1).Save(ctx)
	if e != nil {
		return e
	}
	return writeAirHouseVersion(ctx, tx, h, actor, in.ChangeReason)
}
func lockAirLifecycleTarget(ctx context.Context, tx *ent.Tx, org, id uuid.UUID) (*ent.Order, *airMasterLockContext, error) {
	row, e := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(org)).Only(ctx)
	if e != nil {
		return nil, nil, mapEntError(e, biz.ErrOrderNotFound, nil)
	}
	if row.BusinessType != orderent.BusinessTypeAE {
		o, e := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		return o, nil, e
	}
	l, e := tx.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(id), airlink.StatusEQ("ACTIVE")).Only(ctx)
	if ent.IsNotFound(e) {
		o, e := tx.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		return o, nil, e
	}
	if e != nil {
		return nil, nil, e
	}
	locked, e := lockAirMasterMembers(ctx, tx, org, l.MasterWaybillID, nil)
	if e != nil {
		return nil, nil, e
	}
	for _, o := range locked.members {
		if o.ID == id {
			return o, locked, nil
		}
	}
	return nil, nil, biz.ErrAirWaybillConflict
}
func endAirOrderOnTermination(ctx context.Context, tx *ent.Tx, o *ent.Order, c *airMasterLockContext, actor uuid.UUID, at time.Time) error {
	if o.BusinessType != orderent.BusinessTypeAE {
		return nil
	}
	if c == nil {
		return biz.ErrAirWaybillConflict
	}
	for _, l := range c.links {
		if l.OrderID == o.ID {
			if _, e := l.Update().SetStatus("ENDED").SetEndedAt(at).SetEndedReason("订单终止").SetVersion(l.Version + 1).Save(ctx); e != nil {
				return e
			}
		}
	}
	m, e := c.master.Update().SetVersion(c.master.Version + 1).Save(ctx)
	if e != nil {
		return e
	}
	return writeAirMasterHistory(ctx, tx, m, actor, "END_LINK", "终止空运订单 "+o.OrderNo)
}
func validateAirFlowData(ctx context.Context, tx *ent.Tx, o *ent.Order, target biz.OrderFlowStatus, reason string) error {
	if o.BusinessType != orderent.BusinessTypeAE {
		return nil
	}
	if !biz.ValidAirFlowTransition(biz.OrderFlowStatus(o.FlowStatus), target) {
		return biz.ErrOrderStatusInvalid
	}
	l, e := tx.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(o.ID), airlink.StatusEQ("ACTIVE")).Only(ctx)
	if e != nil {
		return biz.ErrAirWaybillConflict
	}
	m, e := tx.AirMasterWaybill.Get(ctx, l.MasterWaybillID)
	if e != nil {
		return e
	}
	h, e := tx.AirHouseWaybill.Query().Where(airhouse.OrderIDEQ(o.ID)).Only(ctx)
	if e != nil {
		return e
	}
	if target == biz.OrderFlowSpaceAllocated && m.FlightNo == "" {
		return biz.NewAirFlowDataError("配舱前请填写主单航班号")
	}
	if target == biz.OrderFlowDocumentCutoff && (o.GoodsDescription == "" || o.TotalGrossWeightKg == nil || o.TotalVolumeCbm == nil) {
		return biz.NewAirFlowDataError("截单前请补全货物描述、毛重和体积")
	}
	if target == biz.OrderFlowCustomsDeclarationArranged && o.OperationNotes == "" {
		return biz.NewAirFlowDataError("报关安排前请填写本票操作记录")
	}
	if target == biz.OrderFlowDocumentReleased && (h.Status != "RELEASED" || m.Status != "RELEASED") {
		return biz.NewAirFlowDataError("签单前请将主单和本票分单登记为已签发")
	}
	cutoff := ""
	if target == biz.OrderFlowDocumentCutoff {
		cutoff = m.DocumentCutoffAt
	}
	if target == biz.OrderFlowCustomsDeclarationArranged {
		cutoff = m.CustomsCutoffAt
	}
	if cutoff != "" {
		t, e := time.Parse(time.RFC3339, cutoff)
		if e != nil {
			return e
		}
		if time.Now().After(t) && reason == "" {
			return biz.NewAirFlowDataError("已超过截止时间，请填写处理原因后继续")
		}
	}
	return nil
}

// airOrderLockDocumentSnapshot 独立记录航空单证，不填充海运外键。
func airOrderLockDocumentSnapshot(ctx context.Context, tx *ent.Tx, org, id uuid.UUID) (map[string]string, error) {
	l, e := tx.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(id), airlink.OrganizationIDEQ(org), airlink.StatusEQ("ACTIVE")).Only(ctx)
	if e != nil {
		return nil, biz.ErrAirWaybillConflict
	}
	m, e := tx.AirMasterWaybill.Get(ctx, l.MasterWaybillID)
	if e != nil {
		return nil, e
	}
	h, e := tx.AirHouseWaybill.Query().Where(airhouse.OrderIDEQ(id)).Only(ctx)
	if e != nil {
		return nil, e
	}
	masterJSON, e := json.Marshal(airMasterToBiz(m))
	if e != nil {
		return nil, e
	}
	houseJSON, e := json.Marshal(struct {
		ID                  uuid.UUID
		Version             uint64
		No, Status, Content string
	}{h.ID, h.Version, h.HouseNo, h.Status, h.Content})
	if e != nil {
		return nil, e
	}
	return map[string]string{"master": string(masterJSON), "house": string(houseJSON), "link_id": l.ID.String()}, nil
}
