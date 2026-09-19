package data

import (
	"context"
	"fmt"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seahousebill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func syncOrderSeaDocumentOnCreate(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, order *ent.Order, input *biz.Order, audit *biz.AuditEvent) error {
	if input.BusinessType != biz.OrderBusinessSE || input.SeaDocumentInput == nil {
		return nil
	}
	docInput := input.SeaDocumentInput

	link, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(order.ID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		WithMasterBill().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	mbl := link.Edges.MasterBill
	if mbl == nil {
		return biz.ErrSeaMasterBillNotFound
	}

	if docInput.MasterBillContent != nil {
		updater := mbl.Update()
		setSeaMasterBillContent(updater, docInput.MasterBillContent)
		if _, err := updater.Save(ctx); err != nil {
			return err
		}
	}

	hasHBL := docInput.HouseBill != nil
	targetStructure := seamasterbillorderlink.DocumentStructureHOUSE

	if docInput.DocumentStructure != nil {
		switch *docInput.DocumentStructure {
		case biz.SeaDocumentStructureDirect:
			if hasHBL {
				return biz.ErrSeaDocumentStructureInvalid
			}
			targetStructure = seamasterbillorderlink.DocumentStructureDIRECT
		case biz.SeaDocumentStructureHouse:
			if !hasHBL {
				return errors.BadRequest("SEA_DOCUMENT_STRUCTURE_INVALID", "HOUSE 单证结构必须包含分单")
			}
			targetStructure = seamasterbillorderlink.DocumentStructureHOUSE
		}
	}

	if targetStructure != link.DocumentStructure {
		if _, err := link.Update().SetDocumentStructure(targetStructure).Save(ctx); err != nil {
			return err
		}
	}

	if hasHBL {
		hbInput := docInput.HouseBill
		normalized, err := biz.NormalizeSeaHouseNo(hbInput.HouseNo)
		if err != nil {
			return err
		}
		// 批次内排重预查（含作废行，一号一案）：先给出含冲突分单号的友好报错，
		// 唯一索引仅作并发兜底。跨批次重号不受限制。
		duplicateExists, err := tx.SeaHouseBill.Query().
			Where(
				seahousebill.MasterBillIDEQ(mbl.ID),
				seahousebill.NormalizedHouseNoEQ(normalized),
			).
			Exist(ctx)
		if err != nil {
			return err
		}
		if duplicateExists {
			return biz.SeaHouseBillBatchNoDuplicateError([]string{normalized})
		}
		if hbInput.Content != nil {
			if _, err := biz.ValidateSeaBillContent(hbInput.Content); err != nil {
				return err
			}
		}
		issuerOrgID, issuerPartnerID, err := validateSeaHouseBillIssuer(ctx, tx.Client(), organizationID, order.OrganizationID, order.CustomerID, hbInput)
		if err != nil {
			return err
		}
		builder := tx.SeaHouseBill.Create().
			SetID(uuid.Must(uuid.NewV7())).
			SetOrganizationID(organizationID).
			SetOrderID(order.ID).
			SetMasterBillID(mbl.ID).
			SetHouseNo(hbInput.HouseNo).
			SetNormalizedHouseNo(normalized).
			SetIssuerSource(seahousebill.IssuerSource(hbInput.IssuerSource)).
			SetStatus(seahousebill.StatusDRAFT).
			SetVersion(1)
		if issuerOrgID != nil {
			builder.SetIssuerOrganizationID(*issuerOrgID)
		}
		if issuerPartnerID != nil {
			builder.SetIssuerPartnerID(*issuerPartnerID)
		}
		if hbInput.Note != nil {
			builder.SetNote(*hbInput.Note)
		}
		setSeaHouseBillContentCreate(builder, hbInput.Content)
		if _, err := builder.Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return biz.ErrSeaHouseBillBatchNoDuplicate
			}
			return err
		}
	}

	if audit.Details == nil {
		audit.Details = make(map[string]string)
	}
	audit.Details["sea_document.initial_structure"] = string(targetStructure)
	initialCount := 0
	if hasHBL {
		initialCount = 1
	}
	audit.Details["sea_house_bills.initial_count"] = fmt.Sprintf("%d", initialCount)

	return nil
}

func syncOrderSeaDocumentOnUpdate(
	ctx context.Context,
	tx *ent.Tx,
	organizationID uuid.UUID,
	order *ent.Order,
	input *biz.Order,
	audit *biz.AuditEvent,
	lockContext *seaMasterBillUpdateLockContext,
) error {
	if input.BusinessType != biz.OrderBusinessSE || input.SeaDocumentInput == nil {
		return nil
	}
	docInput := input.SeaDocumentInput
	if docInput.HouseBill != nil {
		return errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "订单整单更新禁止直接提交分单变更，请使用专用单证命令")
	}

	// 1. Order 已在 UpdateDraft 中加锁 ForUpdate
	// 2. 无锁查询定位 active link ID 与 master_bill_id
	activeLinkQuery, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
			seamasterbillorderlink.OrderIDEQ(order.ID),
			seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
		).
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	// 3. 锁 MBL (ForUpdate)
	mbl, err := tx.SeaMasterBill.Query().
		Where(
			seamasterbill.IDEQ(activeLinkQuery.MasterBillID),
			seamasterbill.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}

	// 4. 按 ID 锁 Active Link (ForUpdate) 并重验
	link, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlink.IDEQ(activeLinkQuery.ID),
			seamasterbillorderlink.OrganizationIDEQ(organizationID),
		).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrSeaMasterBillNotFound, nil)
	}
	if link.Status != seamasterbillorderlink.StatusACTIVE || link.MasterBillID != mbl.ID {
		return biz.ErrSeaDocumentStructureConflict
	}

	// 5. 更新 MBL 内容（严格校验 expected_mbl_version）
	if docInput.MasterBillContent != nil {
		if _, err := revalidateSeaMasterBillUpdateMemberSet(
			ctx, tx, organizationID, link.ID, mbl.ID, lockContext,
		); err != nil {
			return err
		}
		if err := ensureLockedMembersAllowSharedMasterBillUpdate(lockContext); err != nil {
			return err
		}
		if docInput.ExpectedMblVersion == nil {
			return errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "修改主单内容必须提供 expected_mbl_version")
		}
		if mbl.Version != *docInput.ExpectedMblVersion {
			return biz.ErrSeaMasterBillConflict
		}
		if _, err := biz.ValidateSeaBillContent(docInput.MasterBillContent); err != nil {
			return err
		}
		updater := mbl.Update().SetVersion(mbl.Version + 1)
		setSeaMasterBillContent(updater, docInput.MasterBillContent)
		if _, err := updater.Save(ctx); err != nil {
			return err
		}
		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["sea_master_bill.content_updated"] = "true"
		audit.Details["sea_master_bill.old_version"] = fmt.Sprintf("%d", mbl.Version)
		audit.Details["sea_master_bill.new_version"] = fmt.Sprintf("%d", mbl.Version+1)
	}

	// 6. 更新单证结构（严格校验 expected_link_version 并复用状态门禁）
	if docInput.DocumentStructure != nil {
		if docInput.ExpectedLinkVersion == nil {
			return errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "修改单证结构必须提供 expected_link_version")
		}
		if link.Version != *docInput.ExpectedLinkVersion {
			return biz.ErrSeaDocumentStructureConflict
		}
		targetStructure := *docInput.DocumentStructure
		if targetStructure == biz.SeaDocumentStructureDirect {
			hbCount, err := tx.SeaHouseBill.Query().
				Where(
					seahousebill.OrganizationIDEQ(organizationID),
					seahousebill.OrderIDEQ(order.ID),
					seahousebill.MasterBillIDEQ(mbl.ID),
				).Count(ctx)
			if err != nil {
				return err
			}
			if hbCount > 0 {
				return biz.ErrSeaDocumentStructureInvalid
			}
			if link.DocumentStructure != seamasterbillorderlink.DocumentStructureDIRECT {
				if _, err := link.Update().SetDocumentStructure(seamasterbillorderlink.DocumentStructureDIRECT).SetVersion(link.Version + 1).Save(ctx); err != nil {
					return err
				}
			}
		} else if targetStructure == biz.SeaDocumentStructureHouse {
			return errors.BadRequest("SEA_DOCUMENT_INVALID_ARGUMENT", "切换单证结构请使用专用模式切换命令")
		}

		if audit.Details == nil {
			audit.Details = make(map[string]string)
		}
		audit.Details["sea_document.structure_updated"] = "true"
		audit.Details["sea_document.old_structure"] = string(link.DocumentStructure)
		audit.Details["sea_document.new_structure"] = string(targetStructure)
		audit.Details["sea_document.old_link_version"] = fmt.Sprintf("%d", link.Version)
		audit.Details["sea_document.new_link_version"] = fmt.Sprintf("%d", link.Version+1)
	}

	return nil
}
