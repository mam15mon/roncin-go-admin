package data

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seaorderreassignmenteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaorderreassignmentevent"
	seaorderspliteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitevent"
	seaordersplitresultent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitresult"
)

func decodeSplitResultSnapshotSummary(raw []byte) (int32, decimal.Decimal, decimal.Decimal, error) {
	var snapshot struct {
		SchemaVersion int     `json:"schema_version"`
		PackageCount  *int32  `json:"package_count"`
		GrossWeightKg *string `json:"gross_weight_kg"`
		VolumeCbm     *string `json:"volume_cbm"`
	}
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		return 0, decimal.Zero, decimal.Zero, err
	}
	if snapshot.SchemaVersion != 1 || snapshot.PackageCount == nil || snapshot.GrossWeightKg == nil || snapshot.VolumeCbm == nil {
		return 0, decimal.Zero, decimal.Zero, fmt.Errorf("拆票结果快照缺少必需字段或版本不受支持")
	}
	grossWeight, err := decimal.NewFromString(*snapshot.GrossWeightKg)
	if err != nil {
		return 0, decimal.Zero, decimal.Zero, err
	}
	volume, err := decimal.NewFromString(*snapshot.VolumeCbm)
	if err != nil {
		return 0, decimal.Zero, decimal.Zero, err
	}
	return *snapshot.PackageCount, grossWeight, volume, nil
}

func (r *seaOrderChangeRepo) ListChangeEvents(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int32) ([]*biz.SeaOrderChangeEventSummary, int32, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}

	splitEvents, err := client.SeaOrderSplitEvent.Query().
		Where(
			seaorderspliteventent.OrganizationIDEQ(organizationID),
			seaorderspliteventent.Or(
				seaorderspliteventent.SourceOrderIDEQ(orderID),
				seaorderspliteventent.HasResultsWith(seaordersplitresultent.OrderIDEQ(orderID)),
			),
		).
		WithResults(func(query *ent.SeaOrderSplitResultQuery) {
			query.WithFinalMasterBill()
		}).
		WithCreator().
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	reassignEvents, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.OrderIDEQ(orderID),
		).
		WithPreviousMasterBill().
		WithTargetMasterBill().
		WithCreator().
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var allEvents []*biz.SeaOrderChangeEventSummary

	for _, se := range splitEvents {
		opName := ""
		if se.Edges.Creator != nil {
			opName = se.Edges.Creator.DisplayName
		}
		resItems := make([]*biz.SeaOrderSplitResultSummaryItem, 0, len(se.Edges.Results))
		for _, res := range se.Edges.Results {
			finalMasterBill, err := res.Edges.FinalMasterBillOrErr()
			if err != nil {
				return nil, 0, err
			}
			packageCount, grossWeight, volume, err := decodeSplitResultSnapshotSummary(res.ResultSnapshot)
			if err != nil {
				return nil, 0, err
			}
			resItems = append(resItems, &biz.SeaOrderSplitResultSummaryItem{
				ResultRole:    string(res.ResultRole),
				OrderID:       res.OrderID,
				OrderNo:       res.OrderNo,
				FinalMasterNo: finalMasterBill.MasterNo,
				PackageCount:  packageCount,
				GrossWeightKg: grossWeight,
				VolumeCbm:     volume,
			})
		}
		summary := &biz.SeaOrderSplitEventSummary{
			SourceOrderID: se.SourceOrderID,
			SourceOrderNo: se.SourceOrderNo,
			ResultCount:   int32(len(se.Edges.Results)),
			Results:       resItems,
		}
		note := ""
		if se.Note != nil {
			note = *se.Note
		}
		allEvents = append(allEvents, &biz.SeaOrderChangeEventSummary{
			ID:           se.ID,
			EventType:    biz.EventTypeSplit,
			CreatedAt:    se.CreatedAt,
			OperatorID:   se.CreatedBy,
			OperatorName: opName,
			NoteOrReason: note,
			SplitSummary: summary,
		})
	}

	for _, re := range reassignEvents {
		opName := ""
		if re.Edges.Creator != nil {
			opName = re.Edges.Creator.DisplayName
		}
		prevNo := ""
		if re.Edges.PreviousMasterBill != nil {
			prevNo = re.Edges.PreviousMasterBill.MasterNo
		}
		targetNo := ""
		if re.Edges.TargetMasterBill != nil {
			targetNo = re.Edges.TargetMasterBill.MasterNo
		}
		respName := ""
		if re.ResponsiblePartnerName != nil {
			respName = *re.ResponsiblePartnerName
		}
		summary := &biz.SeaOrderReassignmentEventSummary{
			OrderID:                re.OrderID,
			OrderNo:                re.OrderNo,
			PreviousMasterNo:       prevNo,
			TargetMasterNo:         targetNo,
			ResponsibilityType:     string(re.ResponsibilityType),
			ResponsiblePartnerName: respName,
			Reason:                 re.Reason,
			Confirmation:           externalConfirmationFromReassignment(re),
		}
		allEvents = append(allEvents, &biz.SeaOrderChangeEventSummary{
			ID:                  re.ID,
			EventType:           biz.EventTypeReassignment,
			CreatedAt:           re.CreatedAt,
			OperatorID:          re.CreatedBy,
			OperatorName:        opName,
			NoteOrReason:        re.Reason,
			ReassignmentSummary: summary,
		})
	}

	sort.Slice(allEvents, func(i, j int) bool {
		if allEvents[i].CreatedAt.Equal(allEvents[j].CreatedAt) {
			return allEvents[i].ID.String() > allEvents[j].ID.String()
		}
		return allEvents[i].CreatedAt.After(allEvents[j].CreatedAt)
	})

	total := int32(len(allEvents))
	start := int((page - 1) * pageSize)
	if start >= len(allEvents) {
		return []*biz.SeaOrderChangeEventSummary{}, total, nil
	}
	end := start + int(pageSize)
	if end > len(allEvents) {
		end = len(allEvents)
	}

	return allEvents[start:end], total, nil
}

func (r *seaOrderChangeRepo) GetChangeEvent(ctx context.Context, organizationID, orderID, eventID uuid.UUID, eventType string) (*biz.SeaOrderChangeEventDetail, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	if eventType == biz.EventTypeSplit {
		se, err := client.SeaOrderSplitEvent.Query().
			Where(
				seaorderspliteventent.IDEQ(eventID),
				seaorderspliteventent.OrganizationIDEQ(organizationID),
				seaorderspliteventent.Or(
					seaorderspliteventent.SourceOrderIDEQ(orderID),
					seaorderspliteventent.HasResultsWith(seaordersplitresultent.OrderIDEQ(orderID)),
				),
			).
			WithResults(func(query *ent.SeaOrderSplitResultQuery) {
				query.WithFinalMasterBill()
			}).
			WithCreator().
			Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaOrderSplitInvalidArgument, nil)
		}
		opName := ""
		if se.Edges.Creator != nil {
			opName = se.Edges.Creator.DisplayName
		}
		resItems := make([]*biz.SeaOrderSplitResultSummaryItem, 0, len(se.Edges.Results))
		for _, res := range se.Edges.Results {
			finalMasterBill, err := res.Edges.FinalMasterBillOrErr()
			if err != nil {
				return nil, err
			}
			packageCount, grossWeight, volume, err := decodeSplitResultSnapshotSummary(res.ResultSnapshot)
			if err != nil {
				return nil, err
			}
			resItems = append(resItems, &biz.SeaOrderSplitResultSummaryItem{
				ResultRole:    string(res.ResultRole),
				OrderID:       res.OrderID,
				OrderNo:       res.OrderNo,
				FinalMasterNo: finalMasterBill.MasterNo,
				PackageCount:  packageCount,
				GrossWeightKg: grossWeight,
				VolumeCbm:     volume,
			})
		}
		note := ""
		if se.Note != nil {
			note = *se.Note
		}
		return &biz.SeaOrderChangeEventDetail{
			ID:                       se.ID,
			EventType:                biz.EventTypeSplit,
			CreatedAt:                se.CreatedAt,
			OperatorID:               se.CreatedBy,
			OperatorName:             opName,
			NoteOrReason:             note,
			BeforeSnapshotJSON:       string(se.BeforeSnapshot),
			ConservationSnapshotJSON: string(se.ConservationSnapshot),
			SplitSummary: &biz.SeaOrderSplitEventSummary{
				SourceOrderID: se.SourceOrderID,
				SourceOrderNo: se.SourceOrderNo,
				ResultCount:   int32(len(se.Edges.Results)),
				Results:       resItems,
			},
		}, nil
	}

	re, err := client.SeaOrderReassignmentEvent.Query().
		Where(
			seaorderreassignmenteventent.IDEQ(eventID),
			seaorderreassignmenteventent.OrganizationIDEQ(organizationID),
			seaorderreassignmenteventent.OrderIDEQ(orderID),
		).
		WithPreviousMasterBill().
		WithTargetMasterBill().
		WithCreator().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaOrderReassignmentInvalidArgument, nil)
	}

	opName := ""
	if re.Edges.Creator != nil {
		opName = re.Edges.Creator.DisplayName
	}
	prevNo := ""
	if re.Edges.PreviousMasterBill != nil {
		prevNo = re.Edges.PreviousMasterBill.MasterNo
	}
	targetNo := ""
	if re.Edges.TargetMasterBill != nil {
		targetNo = re.Edges.TargetMasterBill.MasterNo
	}
	respName := ""
	if re.ResponsiblePartnerName != nil {
		respName = *re.ResponsiblePartnerName
	}

	return &biz.SeaOrderChangeEventDetail{
		ID:                 re.ID,
		EventType:          biz.EventTypeReassignment,
		CreatedAt:          re.CreatedAt,
		OperatorID:         re.CreatedBy,
		OperatorName:       opName,
		NoteOrReason:       re.Reason,
		BeforeSnapshotJSON: string(re.BeforeSnapshot),
		AfterSnapshotJSON:  string(re.AfterSnapshot),
		ReassignmentSummary: &biz.SeaOrderReassignmentEventSummary{
			OrderID:                re.OrderID,
			OrderNo:                re.OrderNo,
			PreviousMasterNo:       prevNo,
			TargetMasterNo:         targetNo,
			ResponsibilityType:     string(re.ResponsibilityType),
			ResponsiblePartnerName: respName,
			Reason:                 re.Reason,
			Confirmation:           externalConfirmationFromReassignment(re),
		},
	}, nil
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------
