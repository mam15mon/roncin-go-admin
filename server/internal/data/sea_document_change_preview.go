package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func (r *seaDocumentChangeRepo) PreviewAmendment(ctx context.Context, orgID uuid.UUID, input *biz.SeaDocumentAmendmentCommand) (*biz.SeaDocumentChangePreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	base, diffs, orderIDs, err := loadAmendmentPreview(ctx, client, orgID, input)
	if err != nil {
		return nil, err
	}
	impacts, err := collectDocumentImpacts(ctx, client, orgID, orderIDs, amendmentImpactHouseBillID(input))
	if err != nil {
		return nil, err
	}
	if len(orderIDs) > 0 {
		orders, err := client.Order.Query().Where(orderent.OrganizationIDEQ(orgID), orderent.IDIn(orderIDs...)).Order(orderent.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range orders {
			if impact := orderBusinessEditImpact(ctx, client.User, o); impact != nil {
				impacts = append(impacts, impact)
			}
		}
	}
	return &biz.SeaDocumentChangePreview{BaseVersion: base, Differences: diffs, Impacts: impacts, Executable: len(diffs) > 0 && !hasBlockingImpact(impacts)}, nil
}

func (r *seaDocumentChangeRepo) PreviewVoid(ctx context.Context, orgID uuid.UUID, input *biz.SeaDocumentVoidCommand) (*biz.SeaDocumentChangePreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	if input.DocumentType == biz.SeaDocumentTypeHouseBill {
		return nil, biz.ErrSeaDocumentStructureConflict
	}
	base, orderIDs, err := loadCurrentDocumentBase(ctx, client, orgID, input.OrderID, input.DocumentType, input.DocumentID, input.ExpectedOrderVersion, input.ExpectedDocumentVersion, input.ExpectedCurrentVersionID)
	if err != nil {
		return nil, err
	}
	diffs := []*biz.SeaDocumentFieldDifference{{Field: "status", Label: "状态", BeforeValue: base.Status, AfterValue: "VOIDED"}}
	impactHouseBillID := uuid.Nil
	if input.DocumentType == biz.SeaDocumentTypeHouseBill {
		impactHouseBillID = input.DocumentID
	}
	impacts, err := collectDocumentImpacts(ctx, client, orgID, orderIDs, impactHouseBillID)
	if err != nil {
		return nil, err
	}
	if len(orderIDs) > 0 {
		orders, err := client.Order.Query().Where(orderent.OrganizationIDEQ(orgID), orderent.IDIn(orderIDs...)).Order(orderent.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, o := range orders {
			if impact := orderBusinessEditImpact(ctx, client.User, o); impact != nil {
				impacts = append(impacts, impact)
			}
		}
	}
	return &biz.SeaDocumentChangePreview{BaseVersion: base, Differences: diffs, Impacts: impacts, Executable: !hasBlockingImpact(impacts)}, nil
}

func (r *seaDocumentChangeRepo) PreviewModeChange(ctx context.Context, orgID uuid.UUID, input *biz.SeaDocumentModeChangeCommand) (*biz.SeaDocumentChangePreview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	order, err := client.Order.Query().Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(orgID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	link, err := client.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlinkent.OrganizationIDEQ(orgID),
		seamasterbillorderlinkent.OrderIDEQ(input.OrderID),
		seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
	).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrSeaDocumentNoActiveLink, nil)
	}
	current := biz.SeaDocumentStructure(link.DocumentStructure)
	if current == input.TargetMode {
		return nil, biz.ErrSeaDocumentModeChangeConflict
	}
	// 与 ExecuteModeChange 同口径：批次内还有其他活动成员票时禁止转为直单，
	// 预览阶段同样拒绝，避免「预览可执行而执行被阻断」的漂移。
	if input.TargetMode == biz.SeaDocumentStructureDirect {
		batchMembers, err := querySeaMasterBillActiveMemberLinks(ctx, client, orgID, link.MasterBillID, false)
		if err != nil {
			return nil, err
		}
		for _, member := range batchMembers {
			if member.OrderID != order.ID {
				return nil, biz.ErrSeaDocumentBatchMemberExitBlocked
			}
		}
	}
	differences := []*biz.SeaDocumentFieldDifference{{Field: "document_structure", Label: "单证模式", BeforeValue: string(current), AfterValue: string(input.TargetMode)}}
	var base *biz.SeaDocumentVersion
	// HOUSE 起点的模式切换会把当前唯一活动 HBL 置为 VOIDED：影响收集必须携带该
	// HBL 上下文，使其箱货分配进入阻断事实（与 ExecuteModeChange 同口径）。
	impactHouseBillID := uuid.Nil
	if current == biz.SeaDocumentStructureHouse {
		hbl, err := client.SeaHouseBill.Query().Where(
			seahousebillent.OrganizationIDEQ(orgID),
			seahousebillent.OrderIDEQ(input.OrderID),
			seahousebillent.MasterBillIDEQ(link.MasterBillID),
			seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED),
		).Only(ctx)
		if err != nil {
			return nil, mapEntError(err, biz.ErrSeaDocumentStructureConflict, nil)
		}
		impactHouseBillID = hbl.ID
		if hbl.CurrentVersionID != nil {
			version, err := client.SeaHouseBillVersion.Query().Where(seahousebillversionent.IDEQ(*hbl.CurrentVersionID)).Only(ctx)
			if err != nil {
				return nil, err
			}
			base = houseVersionToBiz(version)
		}
	} else {
		if input.NewHouseBill == nil {
			return nil, biz.ErrSeaDocumentInvalidArgument
		}
		normalizedNewHouseNo, err := biz.NormalizeSeaHouseNo(input.NewHouseBill.HouseNo)
		if err != nil {
			return nil, err
		}
		// 与 ExecuteModeChange 同口径：批次内排重预查（含作废行，一号一案），
		// 重复分单号在预览阶段即拒绝。
		duplicateExists, err := client.SeaHouseBill.Query().Where(
			seahousebillent.MasterBillIDEQ(link.MasterBillID),
			seahousebillent.NormalizedHouseNoEQ(normalizedNewHouseNo),
		).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if duplicateExists {
			return nil, biz.SeaHouseBillBatchNoDuplicateError([]string{normalizedNewHouseNo})
		}
		if _, _, err := resolveHouseBillIssuerForDiff(ctx, client, orgID, input.OrderID, input.NewHouseBill); err != nil {
			return nil, err
		}
		differences = append(differences, &biz.SeaDocumentFieldDifference{Field: "house_no", Label: "HBL 号", BeforeValue: "", AfterValue: input.NewHouseBill.HouseNo})
	}
	impacts, err := collectDocumentImpacts(ctx, client, orgID, []uuid.UUID{order.ID}, impactHouseBillID)
	if err != nil {
		return nil, err
	}
	if impact := orderBusinessEditImpact(ctx, client.User, order); impact != nil {
		impacts = append(impacts, impact)
	}
	return &biz.SeaDocumentChangePreview{BaseVersion: base, Differences: differences, Impacts: impacts, Executable: !hasBlockingImpact(impacts)}, nil
}
