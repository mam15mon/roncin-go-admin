package data

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	airhouse "github.com/roncin/roncin-go-admin/server/internal/data/ent/airhousewaybill"
	airversion "github.com/roncin/roncin-go-admin/server/internal/data/ent/airhousewaybillversion"
	airline "github.com/roncin/roncin-go-admin/server/internal/data/ent/airline"
	airmaster "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybill"
	airhistory "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillhistory"
	airlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	airport "github.com/roncin/roncin-go-admin/server/internal/data/ent/airport"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	lifecycle "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
)

type airExportRepo struct{ data *Data }

func NewAirExportRepo(d *Data) biz.AirExportRepo { return &airExportRepo{data: d} }

// airMasterLockContext 的成员及关联在订单锁后、主单锁内重验，禁止补取逆序锁。
type airMasterLockContext struct {
	master  *ent.AirMasterWaybill
	members []*ent.Order
	links   []*ent.AirMasterWaybillOrderLink
}

func lockAirMasterMembers(ctx context.Context, tx *ent.Tx, org, masterID uuid.UUID, extraOrderIDs []uuid.UUID) (*airMasterLockContext, error) {
	links, err := tx.AirMasterWaybillOrderLink.Query().Where(airlink.OrganizationIDEQ(org), airlink.MasterWaybillIDEQ(masterID), airlink.StatusEQ("ACTIVE")).All(ctx)
	if err != nil {
		return nil, err
	}
	ids := append([]uuid.UUID(nil), extraOrderIDs...)
	memberIDs := make([]uuid.UUID, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.OrderID)
		memberIDs = append(memberIDs, l.OrderID)
	}
	ids = sortAndDeduplicateUUIDs(ids)
	memberIDs = sortAndDeduplicateUUIDs(memberIDs)
	orders, err := tx.Order.Query().Where(orderent.OrganizationIDEQ(org), orderent.IDIn(ids...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(orders) != len(ids) {
		return nil, biz.ErrAirWaybillConflict
	}
	master, err := tx.AirMasterWaybill.Query().Where(airmaster.IDEQ(masterID), airmaster.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrAirWaybillNotFound, nil)
	}
	lockedLinks, err := tx.AirMasterWaybillOrderLink.Query().Where(airlink.OrganizationIDEQ(org), airlink.MasterWaybillIDEQ(masterID), airlink.StatusEQ("ACTIVE")).Order(airlink.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	currentIDs := make([]uuid.UUID, 0, len(lockedLinks))
	for _, l := range lockedLinks {
		currentIDs = append(currentIDs, l.OrderID)
	}
	currentIDs = sortAndDeduplicateUUIDs(currentIDs)
	if len(memberIDs) != len(currentIDs) {
		return nil, biz.ErrAirWaybillConflict
	}
	for i, id := range memberIDs {
		if id != currentIDs[i] {
			return nil, biz.ErrAirWaybillConflict
		}
	}
	members := make([]*ent.Order, 0, len(links))
	for _, o := range orders {
		for _, id := range memberIDs {
			if o.ID == id {
				members = append(members, o)
				break
			}
		}
	}
	return &airMasterLockContext{master: master, members: members, links: lockedLinks}, nil
}
func ensureAirMembersEditable(c *airMasterLockContext) error {
	for _, o := range c.members {
		if o.BusinessType != orderent.BusinessTypeAE || o.OrganizationID != c.master.OrganizationID || o.LockedAt != nil || o.TerminationStatus != orderent.TerminationStatusACTIVE || o.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrAirWaybillBlocked
		}
	}
	return nil
}
func airMasterToBiz(m *ent.AirMasterWaybill) *biz.AirMasterWaybill {
	if m == nil {
		return nil
	}
	return &biz.AirMasterWaybill{ID: m.ID, OrganizationID: m.OrganizationID, Version: m.Version, Content: biz.AirMasterWaybillInput{MasterNo: m.MasterNo, AirlineID: m.AirlineID, OriginAirportID: m.OriginAirportID, DestinationAirportID: m.DestinationAirportID, FlightNo: m.FlightNo, ETD: m.Etd, ETA: m.Eta, ActualDepartureAt: m.ActualDepartureAt, ActualArrivalAt: m.ActualArrivalAt, DocumentCutoffAt: m.DocumentCutoffAt, CustomsCutoffAt: m.CustomsCutoffAt, Status: m.Status, DeclaredPackages: m.DeclaredPackages, DeclaredGrossWeightKg: m.DeclaredGrossWeightKg, DeclaredVolumeCbm: m.DeclaredVolumeCbm, Notes: m.Notes}, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func decorateAirMaster(ctx context.Context, c *ent.Client, m *biz.AirMasterWaybill) error {
	a, err := c.Airline.Query().Where(airline.IDEQ(m.Content.AirlineID)).Only(ctx)
	if err != nil {
		return err
	}
	m.AirlineName = orderOptionalStringValue(a.NameZh)
	if m.AirlineName == "" {
		m.AirlineName = a.NameEn
	}
	ports, err := c.Airport.Query().Where(airport.IDIn(m.Content.OriginAirportID, m.Content.DestinationAirportID)).All(ctx)
	if err != nil {
		return err
	}
	for _, p := range ports {
		name := orderOptionalStringValue(p.NameZh)
		if name == "" {
			name = p.NameEn
		}
		name = p.IataCode + " " + name
		if p.ID == m.Content.OriginAirportID {
			m.OriginAirportName = name
		}
		if p.ID == m.Content.DestinationAirportID {
			m.DestinationAirportName = name
		}
	}
	m.MemberCount, err = c.AirMasterWaybillOrderLink.Query().Where(airlink.MasterWaybillIDEQ(m.ID), airlink.StatusEQ("ACTIVE")).Count(ctx)
	return err
}
func attachAirOrders(ctx context.Context, c *ent.Client, orders []*biz.Order) error {
	ids := make([]uuid.UUID, 0)
	for _, o := range orders {
		if o.BusinessType == biz.OrderBusinessAE {
			ids = append(ids, o.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	links, err := c.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDIn(ids...)).Order(airlink.ByStartedAt()).All(ctx)
	if err != nil {
		return err
	}
	masters := map[uuid.UUID]*biz.AirMasterWaybill{}
	byOrder := map[uuid.UUID]*ent.AirMasterWaybillOrderLink{}
	for _, l := range links {
		byOrder[l.OrderID] = l
	}
	houses, err := c.AirHouseWaybill.Query().Where(airhouse.OrderIDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	byHouse := map[uuid.UUID]*ent.AirHouseWaybill{}
	for _, h := range houses {
		byHouse[h.OrderID] = h
	}
	for _, o := range orders {
		if o.BusinessType != biz.OrderBusinessAE {
			continue
		}
		l := byOrder[o.ID]
		h := byHouse[o.ID]
		if l == nil || h == nil {
			return biz.ErrAirWaybillConflict
		}
		m := masters[l.MasterWaybillID]
		if m == nil {
			row, e := c.AirMasterWaybill.Query().Where(airmaster.IDEQ(l.MasterWaybillID), airmaster.OrganizationIDEQ(o.OrganizationID)).Only(ctx)
			if e != nil {
				return e
			}
			m = airMasterToBiz(row)
			if e = decorateAirMaster(ctx, c, m); e != nil {
				return e
			}
			masters[m.ID] = m
		}
		o.AirExport = &biz.AirOrderInfo{Master: m, HouseNo: h.HouseNo, HouseStatus: h.Status, HouseVersion: h.Version, LinkID: l.ID, LinkVersion: l.Version, HouseID: h.ID, HouseContent: h.Content, Active: l.Status == "ACTIVE"}
		o.OriginLocationID = &m.Content.OriginAirportID
		o.DestinationLocationID = &m.Content.DestinationAirportID
		o.ETD = m.Content.ETD
		o.ETA = m.Content.ETA
		o.DocCutoff = m.Content.DocumentCutoffAt
		o.CustomsCutoff = m.Content.CustomsCutoffAt
	}
	return nil
}
func (r *airExportRepo) List(ctx context.Context, org uuid.UUID, o biz.SelectorListOptions) ([]*biz.AirMasterWaybill, int, error) {
	c, e := r.data.client(ctx)
	if e != nil {
		return nil, 0, e
	}
	q := c.AirMasterWaybill.Query().Where(airmaster.OrganizationIDEQ(org))
	if o.Keyword != "" {
		q.Where(airmaster.Or(airmaster.MasterNoContainsFold(strings.ReplaceAll(o.Keyword, "-", "")), airmaster.FlightNoContainsFold(o.Keyword),
			airmaster.HasAirlineWith(airline.SearchKeywordsContainsFold(o.Keyword)),
			airmaster.HasOriginAirportWith(airport.SearchKeywordsContainsFold(o.Keyword)),
			airmaster.HasDestinationAirportWith(airport.SearchKeywordsContainsFold(o.Keyword))))
	}
	total, e := q.Clone().Count(ctx)
	if e != nil {
		return nil, 0, e
	}
	rows, e := q.Order(airmaster.ByUpdatedAt()).Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).All(ctx)
	if e != nil {
		return nil, 0, e
	}
	out := make([]*biz.AirMasterWaybill, 0, len(rows))
	for _, row := range rows {
		v := airMasterToBiz(row)
		if e = decorateAirMaster(ctx, c, v); e != nil {
			return nil, 0, e
		}
		out = append(out, v)
	}
	return out, total, nil
}
func airMember(o *ent.Order, h *ent.AirHouseWaybill) *biz.AirMasterMember {
	v := &biz.AirMasterMember{OrderID: o.ID, OrderNo: o.OrderNo, Packages: o.TotalPackages, PackageUnit: o.TotalPackageUnit, GrossWeightKg: o.TotalGrossWeightKg, VolumeCbm: o.TotalVolumeCbm, FlowStatus: string(o.FlowStatus), TerminationStatus: string(o.TerminationStatus), ClosureStatus: string(o.ClosureStatus), Locked: o.LockedAt != nil, OrderVersion: o.Version}
	if h != nil {
		v.HouseNo = h.HouseNo
		v.HouseStatus = h.Status
	}
	return v
}
func loadAirMembers(ctx context.Context, c *ent.Client, org, id uuid.UUID) ([]*biz.AirMasterMember, error) {
	links, err := c.AirMasterWaybillOrderLink.Query().Where(airlink.OrganizationIDEQ(org), airlink.MasterWaybillIDEQ(id), airlink.StatusEQ("ACTIVE")).All(ctx)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.OrderID)
	}
	orders, err := c.Order.Query().Where(orderent.OrganizationIDEQ(org), orderent.BusinessTypeEQ(orderent.BusinessTypeAE), orderent.IDIn(ids...)).WithCustomer().Order(orderent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	if len(orders) != len(ids) {
		return nil, biz.ErrAirWaybillConflict
	}
	houses, err := c.AirHouseWaybill.Query().Where(airhouse.OrderIDIn(ids...), airhouse.MasterWaybillIDEQ(id)).All(ctx)
	if err != nil {
		return nil, err
	}
	hm := map[uuid.UUID]*ent.AirHouseWaybill{}
	for _, h := range houses {
		hm[h.OrderID] = h
	}
	out := make([]*biz.AirMasterMember, 0, len(orders))
	for _, o := range orders {
		m := airMember(o, hm[o.ID])
		if o.Edges.Customer != nil {
			m.CustomerName = o.Edges.Customer.LegalName
		}
		out = append(out, m)
	}
	if e := attachAirMemberWeights(ctx, c, org, out); e != nil {
		return nil, e
	}
	return out, nil
}
func airBlocked(members []*biz.AirMasterMember) []string {
	out := []string{}
	for _, m := range members {
		if m.Locked || m.TerminationStatus != "ACTIVE" || m.ClosureStatus != "OPEN" {
			out = append(out, m.OrderNo+"：锁定、终止或结案，不能修改共享资料")
		}
	}
	return out
}
func (r *airExportRepo) Get(ctx context.Context, org, id uuid.UUID, o biz.SelectorListOptions) (*biz.AirMasterDetail, error) {
	var result *biz.AirMasterDetail
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, e := lockAirMasterMembers(ctx, tx, org, id, nil); e != nil {
			return e
		}
		var e error
		result, e = readAirMasterDetail(ctx, tx.Client(), org, id, o)
		return e
	})
	return result, err
}

// readAirMasterDetail 在完整成员锁内生成聚合，避免分页/成员集合跨版本混合。
func readAirMasterDetail(ctx context.Context, c *ent.Client, org, id uuid.UUID, o biz.SelectorListOptions) (*biz.AirMasterDetail, error) {
	row, e := c.AirMasterWaybill.Query().Where(airmaster.IDEQ(id), airmaster.OrganizationIDEQ(org)).Only(ctx)
	if e != nil {
		return nil, mapEntError(e, biz.ErrAirWaybillNotFound, nil)
	}
	m := airMasterToBiz(row)
	if e = decorateAirMaster(ctx, c, m); e != nil {
		return nil, e
	}
	members, e := loadAirMembers(ctx, c, org, id)
	if e != nil {
		return nil, e
	}
	result := &biz.AirMasterDetail{Master: m, Total: len(members), Totals: biz.AirMasterTotals{PackagesByUnit: map[string]int{}, Complete: true, MemberCount: len(members)}, BlockedReasons: airBlocked(members)}
	for _, v := range members {
		if v.Packages != nil {
			result.Totals.PackagesByUnit[v.PackageUnit] += *v.Packages
		}
		if v.GrossWeightKg != nil {
			result.Totals.GrossWeightKg += *v.GrossWeightKg
		}
		if v.VolumeCbm != nil {
			result.Totals.VolumeCbm += *v.VolumeCbm
		}
	}
	start := (o.Page - 1) * o.PageSize
	if start > len(members) {
		start = len(members)
	}
	end := start + o.PageSize
	if end > len(members) {
		end = len(members)
	}
	result.Members = members[start:end]
	historyQuery := c.AirMasterWaybillHistory.Query().Where(airhistory.MasterWaybillIDEQ(id), airhistory.OrganizationIDEQ(org))
	result.HistoryTotal, e = historyQuery.Clone().Count(ctx)
	if e != nil {
		return nil, e
	}
	rows, e := historyQuery.Order(ent.Desc(airhistory.FieldVersion)).Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).All(ctx)
	if e != nil {
		return nil, e
	}
	for _, h := range rows {
		v := &biz.AirMasterHistory{ID: h.ID, Version: h.Version, Action: h.Action, Reason: h.Reason, ActorID: h.ActorID, OccurredAt: h.CreatedAt}
		if e = json.Unmarshal(h.Content, &v.Content); e != nil {
			return nil, e
		}
		result.History = append(result.History, v)
	}
	return result, nil
}
func (r *airExportRepo) Preview(ctx context.Context, org, id uuid.UUID, v uint64, input *biz.AirMasterWaybillInput) (*biz.AirMasterPreview, error) {
	c, e := r.data.client(ctx)
	if e != nil {
		return nil, e
	}
	row, e := c.AirMasterWaybill.Query().Where(airmaster.IDEQ(id), airmaster.OrganizationIDEQ(org)).Only(ctx)
	if e != nil {
		return nil, mapEntError(e, biz.ErrAirWaybillNotFound, nil)
	}
	if row.Version != v {
		return nil, biz.ErrAirWaybillConflict
	}
	members, e := loadAirMembers(ctx, c, org, id)
	if e != nil {
		return nil, e
	}
	reasons := airBlocked(members)
	if row.MasterNo != input.MasterNo {
		reasons = append(reasons, "主单身份不能通过共享资料修改")
	}
	return &biz.AirMasterPreview{InputFingerprint: biz.AirMasterFingerprint(airMasterToBiz(row), members, input), Members: members, BlockedReasons: reasons, CanExecute: len(reasons) == 0}, nil
}
func validateAirMasterReferences(ctx context.Context, tx *ent.Tx, in *biz.AirMasterWaybillInput) error {
	_, e := tx.Airline.Query().Where(airline.IDEQ(in.AirlineID), airline.EnabledEQ(true)).ForShare().Only(ctx)
	if e != nil {
		return mapEntError(e, biz.ErrAirWaybillInvalid, nil)
	}
	rows, e := tx.Airport.Query().Where(airport.IDIn(in.OriginAirportID, in.DestinationAirportID), airport.EnabledEQ(true)).Order(airport.ByID()).ForShare().All(ctx)
	if e != nil {
		return e
	}
	if len(rows) != 2 {
		return biz.ErrAirWaybillInvalid
	}
	return nil
}
func writeAirMasterHistory(ctx context.Context, tx *ent.Tx, m *ent.AirMasterWaybill, actor uuid.UUID, action, reason string) error {
	payload, e := json.Marshal(airMasterToBiz(m).Content)
	if e != nil {
		return e
	}
	_, e = tx.AirMasterWaybillHistory.Create().SetOrganizationID(m.OrganizationID).SetMasterWaybillID(m.ID).SetVersion(m.Version).SetAction(action).SetReason(reason).SetActorID(actor).SetContent(payload).Save(ctx)
	return e
}
func (r *airExportRepo) Update(ctx context.Context, org, actor, id uuid.UUID, v uint64, in *biz.AirMasterWaybillInput, fingerprint, reason string) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		locked, e := lockAirMasterMembers(ctx, tx, org, id, nil)
		if e != nil {
			return e
		}
		if locked.master.Version != v {
			return biz.ErrAirWaybillConflict
		}
		if e = ensureAirMembersEditable(locked); e != nil {
			return e
		}
		if locked.master.MasterNo != in.MasterNo {
			return biz.ErrAirWaybillInvalid
		}
		members, e := loadAirMembers(ctx, tx.Client(), org, id)
		if e != nil {
			return e
		}
		if biz.AirMasterFingerprint(airMasterToBiz(locked.master), members, in) != fingerprint {
			return biz.ErrAirWaybillConflict
		}
		if e = validateAirMasterReferences(ctx, tx, in); e != nil {
			return e
		}
		update := locked.master.Update().SetVersion(v + 1).SetAirlineID(in.AirlineID).SetOriginAirportID(in.OriginAirportID).SetDestinationAirportID(in.DestinationAirportID).SetFlightNo(in.FlightNo).SetEtd(in.ETD).SetEta(in.ETA).SetActualDepartureAt(in.ActualDepartureAt).SetActualArrivalAt(in.ActualArrivalAt).SetDocumentCutoffAt(in.DocumentCutoffAt).SetCustomsCutoffAt(in.CustomsCutoffAt).SetStatus(in.Status).SetNotes(in.Notes)
		if in.DeclaredPackages == nil {
			update.ClearDeclaredPackages()
		} else {
			update.SetDeclaredPackages(*in.DeclaredPackages)
		}
		if in.DeclaredGrossWeightKg == nil {
			update.ClearDeclaredGrossWeightKg()
		} else {
			update.SetDeclaredGrossWeightKg(*in.DeclaredGrossWeightKg)
		}
		if in.DeclaredVolumeCbm == nil {
			update.ClearDeclaredVolumeCbm()
		} else {
			update.SetDeclaredVolumeCbm(*in.DeclaredVolumeCbm)
		}
		m, e := update.Save(ctx)
		if e != nil {
			return e
		}
		if e = writeAirMasterHistory(ctx, tx, m, actor, "UPDATE", reason); e != nil {
			return e
		}
		return writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: "air.master.update", Result: "success", ResourceID: id.String(), Details: map[string]string{"reason": reason}})
	})
}
func (r *airExportRepo) Link(ctx context.Context, org, actor, masterID, orderID uuid.UUID, mv, ov uint64, house, reason string) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		locked, e := lockAirMasterMembers(ctx, tx, org, masterID, []uuid.UUID{orderID})
		if e != nil {
			return e
		}
		if locked.master.Version != mv {
			return biz.ErrAirWaybillConflict
		}
		if e = ensureAirMembersEditable(locked); e != nil {
			return e
		}
		o, e := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(org)).Only(ctx)
		if e != nil {
			return biz.ErrOrderNotFound
		}
		if o.Version != ov {
			return biz.ErrAirWaybillConflict
		}
		if o.BusinessType != orderent.BusinessTypeAE || o.LockedAt != nil || o.ClosureStatus != orderent.ClosureStatusOPEN || o.TerminationStatus != orderent.TerminationStatusTERMINATED {
			return biz.ErrAirWaybillBlocked
		}
		active, e := tx.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(orderID), airlink.StatusEQ("ACTIVE")).Exist(ctx)
		if e != nil {
			return e
		}
		if active {
			return biz.ErrAirWaybillConflict
		}
		h, e := tx.AirHouseWaybill.Query().Where(airhouse.OrderIDEQ(orderID)).ForUpdate().Only(ctx)
		if e != nil {
			return e
		}
		if h.MasterWaybillID != masterID || h.HouseNo != house {
			return biz.ErrAirWaybillInvalid
		}
		if _, e = tx.AirMasterWaybillOrderLink.Create().SetOrganizationID(org).SetMasterWaybillID(masterID).SetOrderID(orderID).Save(ctx); e != nil {
			return mapEntError(e, nil, biz.ErrAirWaybillExists)
		}
		if _, e = o.Update().SetTerminationStatus(orderent.TerminationStatusACTIVE).ClearTerminationType().ClearTerminationReason().ClearTerminatedAt().ClearTerminatedBy().SetVersion(ov + 1).Save(ctx); e != nil {
			return e
		}
		m, e := locked.master.Update().SetVersion(mv + 1).Save(ctx)
		if e != nil {
			return e
		}
		if e = writeAirMasterHistory(ctx, tx, m, actor, "RESTORE_LINK", reason); e != nil {
			return e
		}
		if _, e = tx.OrderLifecycleEvent.Create().SetOrderID(orderID).SetDimension(lifecycle.DimensionTERMINATION).SetFromStatus("TERMINATED").SetToStatus("ACTIVE").SetAction("restore_air_link").SetReason(reason).SetOperatorID(actor).Save(ctx); e != nil {
			return e
		}
		return writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{OrganizationID: &org, UserID: &actor, Action: "air.master.link", Result: "success", ResourceID: masterID.String(), Details: map[string]string{"order.id": orderID.String(), "reason": reason}})
	})
}

// 供重量确认复用；成员集合始终按订单主键排序。
func airMemberOrderIDs(locked *airMasterLockContext) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(locked.members))
	for _, m := range locked.members {
		ids = append(ids, m.ID)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

// LinkCandidates 只返回原主单内已终止且无活动关系的可恢复票，不能成为改配旁路。
func (r *airExportRepo) LinkCandidates(ctx context.Context, org, masterID uuid.UUID, o biz.SelectorListOptions) ([]*biz.AirMasterMember, int, error) {
	c, e := r.data.client(ctx)
	if e != nil {
		return nil, 0, e
	}
	exists, e := c.AirMasterWaybill.Query().Where(airmaster.IDEQ(masterID), airmaster.OrganizationIDEQ(org)).Exist(ctx)
	if e != nil {
		return nil, 0, e
	}
	if !exists {
		return nil, 0, biz.ErrAirWaybillNotFound
	}
	houses, e := c.AirHouseWaybill.Query().Where(airhouse.OrganizationIDEQ(org), airhouse.MasterWaybillIDEQ(masterID)).All(ctx)
	if e != nil {
		return nil, 0, e
	}
	ids := make([]uuid.UUID, 0, len(houses))
	hm := map[uuid.UUID]*ent.AirHouseWaybill{}
	for _, h := range houses {
		ids = append(ids, h.OrderID)
		hm[h.OrderID] = h
	}
	active, e := c.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDIn(ids...), airlink.StatusEQ("ACTIVE")).All(ctx)
	if e != nil {
		return nil, 0, e
	}
	activeIDs := make([]uuid.UUID, 0, len(active))
	for _, l := range active {
		activeIDs = append(activeIDs, l.OrderID)
	}
	q := c.Order.Query().Where(orderent.OrganizationIDEQ(org), orderent.BusinessTypeEQ(orderent.BusinessTypeAE), orderent.IDIn(ids...), orderent.TerminationStatusEQ(orderent.TerminationStatusTERMINATED), orderent.ClosureStatusEQ(orderent.ClosureStatusOPEN), orderent.LockedAtIsNil())
	if len(activeIDs) > 0 {
		q.Where(orderent.IDNotIn(activeIDs...))
	}
	if o.Keyword != "" {
		q.Where(orderent.OrderNoContainsFold(o.Keyword))
	}
	total, e := q.Clone().Count(ctx)
	if e != nil {
		return nil, 0, e
	}
	rows, e := q.Order(orderent.ByID()).Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).All(ctx)
	if e != nil {
		return nil, 0, e
	}
	out := make([]*biz.AirMasterMember, 0, len(rows))
	for _, row := range rows {
		out = append(out, airMember(row, hm[row.ID]))
	}
	return out, total, nil
}

func writeAirHouseVersion(ctx context.Context, tx *ent.Tx, h *ent.AirHouseWaybill, actor uuid.UUID, reason string) error {
	_, e := tx.AirHouseWaybillVersion.Create().SetOrganizationID(h.OrganizationID).SetOrderID(h.OrderID).SetHouseWaybillID(h.ID).SetVersion(h.Version).SetHouseNo(h.HouseNo).SetStatus(h.Status).SetContent(h.Content).SetActorID(actor).SetReason(reason).Save(ctx)
	return e
}
func (r *airExportRepo) HouseVersions(ctx context.Context, org, id uuid.UUID, o biz.SelectorListOptions) ([]*biz.AirHouseVersion, int, error) {
	c, e := r.data.client(ctx)
	if e != nil {
		return nil, 0, e
	}
	exists, e := c.Order.Query().Where(orderent.IDEQ(id), orderent.OrganizationIDEQ(org), orderent.BusinessTypeEQ(orderent.BusinessTypeAE)).Exist(ctx)
	if e != nil {
		return nil, 0, e
	}
	if !exists {
		return nil, 0, biz.ErrOrderNotFound
	}
	q := c.AirHouseWaybillVersion.Query().Where(airversion.OrganizationIDEQ(org), airversion.OrderIDEQ(id))
	total, e := q.Clone().Count(ctx)
	if e != nil {
		return nil, 0, e
	}
	rows, e := q.Order(ent.Desc(airversion.FieldVersion)).Offset((o.Page - 1) * o.PageSize).Limit(o.PageSize).All(ctx)
	if e != nil {
		return nil, 0, e
	}
	out := make([]*biz.AirHouseVersion, 0, len(rows))
	for _, v := range rows {
		out = append(out, &biz.AirHouseVersion{ID: v.ID, Version: v.Version, HouseNo: v.HouseNo, Status: v.Status, Content: v.Content, ActorID: v.ActorID, Reason: v.Reason, OccurredAt: v.CreatedAt})
	}
	return out, total, nil
}
