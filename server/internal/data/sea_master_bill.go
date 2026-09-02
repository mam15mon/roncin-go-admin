package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

type seaMasterBillRepo struct {
	data *Data
}

func NewSeaMasterBillRepo(data *Data) biz.SeaMasterBillRepo {
	return &seaMasterBillRepo{data: data}
}

func (r *seaMasterBillRepo) MatchCandidate(ctx context.Context, organizationID, issuerPartnerID uuid.UUID, normalizedMasterNo string, voyage *biz.SeaTransportExecution) (*biz.SeaMasterBillMatchResult, error) {
	return r.matchCandidateInternal(ctx, organizationID, issuerPartnerID, normalizedMasterNo, voyage)
}

func (r *seaMasterBillRepo) matchCandidateInternal(ctx context.Context, organizationID, issuerPartnerID uuid.UUID, normalizedMasterNo string, voyage *biz.SeaTransportExecution) (*biz.SeaMasterBillMatchResult, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	mbl, err := client.SeaMasterBill.Query().
		Where(
			seamasterbill.OrganizationIDEQ(organizationID),
			seamasterbill.IssuerPartnerIDEQ(issuerPartnerID),
			seamasterbill.NormalizedMasterNoEQ(normalizedMasterNo),
		).
		WithTransportExecution().
		WithOrderLinks(func(q *ent.SeaMasterBillOrderLinkQuery) {
			q.Where(seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE)).
				WithOrder(func(oq *ent.OrderQuery) {
					oq.Where(orderent.OrganizationIDEQ(organizationID))
				})
		}).
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			return &biz.SeaMasterBillMatchResult{Matched: false}, nil
		}
		return nil, err
	}

	te := mbl.Edges.TransportExecution
	if te == nil {
		return &biz.SeaMasterBillMatchResult{Matched: false}, nil
	}

	candidateTE := &biz.SeaTransportExecution{
		ID:                te.ID,
		OrganizationID:    te.OrganizationID,
		TransitLocationID: te.TransitLocationID,
		VesselName:        te.VesselName,
		VoyageNo:          te.VoyageNo,
		ETD:               te.Etd,
		ETA:               te.Eta,
		Version:           te.Version,
		CreatedAt:         te.CreatedAt,
		UpdatedAt:         te.UpdatedAt,
	}
	if te.CarrierID != nil {
		candidateTE.CarrierID = *te.CarrierID
	}
	if te.OriginLocationID != nil {
		candidateTE.OriginLocationID = *te.OriginLocationID
	}
	if te.DischargeLocationID != nil {
		candidateTE.DischargeLocationID = *te.DischargeLocationID
	}

	// 填补港口名称与承运人名称
	if err := r.populateLocationAndPartnerNames(ctx, client, organizationID, candidateTE); err != nil {
		return nil, err
	}

	var members []*biz.SeaMasterBillMemberSummary
	for _, link := range mbl.Edges.OrderLinks {
		if link.Edges.Order != nil {
			members = append(members, &biz.SeaMasterBillMemberSummary{
				OrderID:             link.Edges.Order.ID,
				OrderNo:             link.Edges.Order.OrderNo,
				CustomerReferenceNo: link.Edges.Order.CustomerReferenceNo,
			})
		}
	}

	issuerName, err := r.getPartnerName(ctx, client, organizationID, mbl.IssuerPartnerID)
	if err != nil {
		return nil, err
	}

	candidate := &biz.SeaMasterBillCandidate{
		ID:                 mbl.ID,
		Version:            mbl.Version,
		MasterNo:           mbl.MasterNo,
		IssuerPartnerID:    mbl.IssuerPartnerID,
		IssuerPartnerName:  issuerName,
		TransportExecution: candidateTE,
		MemberCount:        len(members),
		Members:            members,
	}

	var conflicts []*biz.SeaVoyageConflict
	if voyage != nil {
		conflicts = biz.CheckSeaVoyageConflicts(candidateTE, voyage)
	}

	return &biz.SeaMasterBillMatchResult{
		Matched:   true,
		Candidate: candidate,
		Conflicts: conflicts,
	}, nil
}

func (r *seaMasterBillRepo) GetSummaryByOrderID(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.SeaMasterBillSummary, error) {
	summaries, err := r.GetSummariesByOrderIDs(ctx, organizationID, []uuid.UUID{orderID})
	if err != nil {
		return nil, err
	}
	return summaries[orderID], nil
}

func (r *seaMasterBillRepo) GetSummariesByOrderIDs(ctx context.Context, organizationID uuid.UUID, orderIDs []uuid.UUID) (map[uuid.UUID]*biz.SeaMasterBillSummary, error) {
	result := make(map[uuid.UUID]*biz.SeaMasterBillSummary, len(orderIDs))
	if len(orderIDs) == 0 {
		return result, nil
	}

	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	links, err := client.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDIn(orderIDs...),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		WithMasterBill(func(q *ent.SeaMasterBillQuery) {
			q.Where(seamasterbill.OrganizationIDEQ(organizationID)).
				WithTransportExecution()
		}).
		All(ctx)

	if err != nil {
		return nil, err
	}

	if len(links) == 0 {
		return result, nil
	}

	// 收集所有 MBL ID 查询成员数
	mblIDs := make([]uuid.UUID, 0, len(links))
	for _, link := range links {
		if link.Edges.MasterBill != nil {
			mblIDs = append(mblIDs, link.MasterBillID)
		}
	}

	memberCountMap := make(map[uuid.UUID]int)
	if len(mblIDs) > 0 {
		allLinks, err := client.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlink.OrganizationIDEQ(organizationID),
				seamasterbillorderlink.MasterBillIDIn(mblIDs...),
				seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
			).
			All(ctx)
		if err != nil {
			return nil, err
		}
		for _, l := range allLinks {
			memberCountMap[l.MasterBillID]++
		}
	}

	for _, link := range links {
		mbl := link.Edges.MasterBill
		if mbl == nil {
			continue
		}
		te := mbl.Edges.TransportExecution
		issuerName, err := r.getPartnerName(ctx, client, organizationID, mbl.IssuerPartnerID)
		if err != nil {
			return nil, err
		}
		summary := &biz.SeaMasterBillSummary{
			MasterBillID:      mbl.ID,
			MasterNo:          mbl.MasterNo,
			IssuerPartnerID:   mbl.IssuerPartnerID,
			IssuerPartnerName: issuerName,
			Status:            string(mbl.Status),
			Version:           mbl.Version,
			MemberCount:       memberCountMap[mbl.ID],
		}
		if te != nil {
			transportExecution := &biz.SeaTransportExecution{
				CarrierID:           uuid.Nil,
				OriginLocationID:    uuid.Nil,
				DischargeLocationID: uuid.Nil,
				TransitLocationID:   te.TransitLocationID,
			}
			if te.CarrierID != nil {
				transportExecution.CarrierID = *te.CarrierID
			}
			if te.OriginLocationID != nil {
				transportExecution.OriginLocationID = *te.OriginLocationID
			}
			if te.DischargeLocationID != nil {
				transportExecution.DischargeLocationID = *te.DischargeLocationID
			}
			if err := r.populateLocationAndPartnerNames(ctx, client, organizationID, transportExecution); err != nil {
				return nil, err
			}
			summary.TransportExecutionID = te.ID
			summary.CarrierID = te.CarrierID
			summary.CarrierName = transportExecution.CarrierName
			summary.OriginLocationID = te.OriginLocationID
			summary.OriginLocationName = transportExecution.OriginLocationName
			summary.DischargeLocationID = te.DischargeLocationID
			summary.DischargeLocationName = transportExecution.DischargeLocationName
			summary.TransitLocationID = te.TransitLocationID
			summary.TransitLocationName = transportExecution.TransitLocationName
			summary.VesselName = te.VesselName
			summary.VoyageNo = te.VoyageNo
			if te.Etd != nil {
				summary.ETD = te.Etd.Format("2006-01-02")
			}
			if te.Eta != nil {
				summary.ETA = te.Eta.Format("2006-01-02")
			}
		}
		result[link.OrderID] = summary
	}

	return result, nil
}

func (r *seaMasterBillRepo) getPartnerName(ctx context.Context, client *ent.Client, organizationID, partnerID uuid.UUID) (string, error) {
	if partnerID == uuid.Nil {
		return "", nil
	}
	partner, err := client.Partner.Query().Where(
		partnerent.IDEQ(partnerID),
		partnerent.OrganizationIDEQ(organizationID),
	).Only(ctx)
	if err != nil {
		return "", mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	return partner.LegalName, nil
}

func (r *seaMasterBillRepo) populateLocationAndPartnerNames(ctx context.Context, client *ent.Client, organizationID uuid.UUID, te *biz.SeaTransportExecution) error {
	if te == nil {
		return nil
	}
	if te.CarrierID != uuid.Nil {
		carrierName, err := r.getPartnerName(ctx, client, organizationID, te.CarrierID)
		if err != nil {
			return err
		}
		te.CarrierName = carrierName
	}
	locIDs := make([]uuid.UUID, 0, 3)
	seenLocationIDs := make(map[uuid.UUID]struct{}, 3)
	appendLocationID := func(id uuid.UUID) {
		if id == uuid.Nil {
			return
		}
		if _, exists := seenLocationIDs[id]; exists {
			return
		}
		seenLocationIDs[id] = struct{}{}
		locIDs = append(locIDs, id)
	}
	appendLocationID(te.OriginLocationID)
	appendLocationID(te.DischargeLocationID)
	if te.TransitLocationID != nil {
		appendLocationID(*te.TransitLocationID)
	}
	if len(locIDs) == 0 {
		return nil
	}
	ports, err := client.Port.Query().Where(portent.OrganizationIDEQ(organizationID), portent.IDIn(locIDs...)).All(ctx)
	if err != nil {
		return err
	}
	if len(ports) != len(locIDs) {
		return biz.ErrSeaMasterBillNotFound
	}
	portMap := make(map[uuid.UUID]string)
	for _, p := range ports {
		name := p.NameEn
		if p.NameZh != nil && *p.NameZh != "" {
			name = *p.NameZh + " / " + p.NameEn
		}
		if p.UnLocode != "" {
			name = name + " (" + p.UnLocode + ")"
		}
		portMap[p.ID] = name
	}
	te.OriginLocationName = portMap[te.OriginLocationID]
	te.DischargeLocationName = portMap[te.DischargeLocationID]
	if te.TransitLocationID != nil {
		te.TransitLocationName = portMap[*te.TransitLocationID]
	}
	return nil
}
