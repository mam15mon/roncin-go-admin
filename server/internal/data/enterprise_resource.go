package data

import (
	"context"
	"sort"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	regionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/administrativeregion"
	resourceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresource"
	addressent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceaddress"
	addresstypeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceaddresstype"
	assigneeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceassignee"
	imageent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceimage"
	partnerlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourcepartner"
	partyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceparty"
	remarkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterpriseresourceremark"
	tagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterprisetag"
	taggroupent "github.com/roncin/roncin-go-admin/server/internal/data/ent/enterprisetaggroup"
	billtaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillenterprisetag"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	ordertaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderenterprisetag"
	feetaglinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraliasent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneralias"
	partnercontactent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnercontact"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type enterpriseResourceRepo struct{ data *Data }

func NewEnterpriseResourceRepo(data *Data) biz.EnterpriseResourceRepo {
	return &enterpriseResourceRepo{data: data}
}

func enterpriseResourceQuery(client *ent.Client) *ent.EnterpriseResourceQuery {
	return client.EnterpriseResource.Query().
		WithAddress().WithRemark().WithImage().WithParty().WithTag().
		WithPartnerLinks().WithAssignees().WithAddressTypes()
}

func (r *enterpriseResourceRepo) SearchPartnerOptions(ctx context.Context, organizationID uuid.UUID, keyword string, page, pageSize int) ([]*biz.EnterpriseResourcePartnerOption, int64, error) {
	query := r.data.db.Partner.Query().Where(partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true))
	if keyword != "" {
		query.Where(partnerent.Or(
			partnerent.CodeContainsFold(keyword), partnerent.LegalNameContainsFold(keyword), partnerent.SearchKeywordsContainsFold(keyword),
			partnerent.HasAliasesWith(partneraliasent.Or(partneraliasent.AliasNameContainsFold(keyword), partneraliasent.SearchKeywordsContainsFold(keyword))),
			partnerent.HasContactsWith(partnercontactent.Or(partnercontactent.NameContainsFold(keyword), partnercontactent.PhoneContainsFold(keyword), partnercontactent.EmailContainsFold(keyword))),
		))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.Order(partnerent.ByLegalName(), partnerent.ByID()).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.EnterpriseResourcePartnerOption, 0, len(items))
	for _, item := range items {
		result = append(result, &biz.EnterpriseResourcePartnerOption{ID: item.ID, Code: item.Code, Name: item.LegalName})
	}
	return result, int64(total), nil
}

func (r *enterpriseResourceRepo) SearchAssigneeOptions(ctx context.Context, organizationID uuid.UUID, keyword string, page, pageSize int) ([]*biz.EnterpriseResourceAssigneeOption, int64, error) {
	query := r.data.db.Membership.Query().Where(
		membershipent.OrganizationIDEQ(organizationID), membershipent.EnabledEQ(true), membershipent.HasUserWith(userent.EnabledEQ(true)),
	)
	if keyword != "" {
		query.Where(membershipent.HasUserWith(userent.Or(userent.UsernameContainsFold(keyword), userent.DisplayNameContainsFold(keyword), userent.SearchKeywordsContainsFold(keyword))))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.WithUser().Order(membershipent.ByUserField(userent.FieldDisplayName), membershipent.ByID()).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.EnterpriseResourceAssigneeOption, 0, len(items))
	for _, item := range items {
		if item.Edges.User != nil {
			result = append(result, &biz.EnterpriseResourceAssigneeOption{ID: item.UserID, Username: item.Edges.User.Username, DisplayName: item.Edges.User.DisplayName})
		}
	}
	return result, int64(total), nil
}

func (r *enterpriseResourceRepo) ListRegionOptions(ctx context.Context, level int, parentCode *string, page, pageSize int) ([]*biz.EnterpriseResourceRegionOption, int64, error) {
	query := r.data.db.AdministrativeRegion.Query().Where(regionent.LevelEQ(level), regionent.EnabledEQ(true))
	if parentCode != nil {
		query.Where(regionent.ParentCodeEQ(*parentCode))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := query.Order(regionent.ByCode()).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.EnterpriseResourceRegionOption, 0, len(items))
	for _, item := range items {
		result = append(result, &biz.EnterpriseResourceRegionOption{Code: item.Code, Name: item.Name, Level: item.Level, ParentCode: stringValue(item.ParentCode)})
	}
	return result, int64(total), nil
}

func (r *enterpriseResourceRepo) ImageUsage(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	var result []struct {
		Total *int64 `json:"total"`
	}
	err := r.data.db.EnterpriseResourceImage.Query().
		Where(imageent.HasResourceWith(resourceent.OrganizationIDEQ(organizationID))).
		Aggregate(ent.As(ent.Sum(imageent.FieldFileSize), "total")).
		Scan(ctx, &result)
	if err != nil {
		return 0, err
	}
	if len(result) == 0 || result[0].Total == nil {
		return 0, nil
	}
	return *result[0].Total, nil
}

func (r *enterpriseResourceRepo) List(ctx context.Context, organizationID uuid.UUID, options biz.EnterpriseResourceListOptions) ([]*biz.EnterpriseResource, int64, error) {
	query := enterpriseResourceQuery(r.data.db).Where(
		resourceent.OrganizationIDEQ(organizationID),
		resourceent.ResourceTypeEQ(resourceent.ResourceType(options.ResourceType)),
	)
	if options.PartnerID != nil {
		query.Where(resourceent.HasPartnerLinksWith(partnerlinkent.PartnerIDEQ(*options.PartnerID)))
	}
	if options.Linked != nil {
		if *options.Linked {
			query.Where(resourceent.HasPartnerLinks())
		} else {
			query.Where(resourceent.Not(resourceent.HasPartnerLinks()))
		}
	}
	if options.Enabled != nil {
		query.Where(resourceent.EnabledEQ(*options.Enabled))
	}
	if options.AddressType != nil {
		query.Where(resourceent.HasAddressTypesWith(addresstypeent.AddressTypeEQ(addresstypeent.AddressType(*options.AddressType))))
	}
	if options.AssigneeID != nil {
		query.Where(resourceent.HasAssigneesWith(assigneeent.UserIDEQ(*options.AssigneeID)))
	}
	if options.Keyword != "" {
		keyword := options.Keyword
		query.Where(resourceent.Or(
			resourceent.ShortNameContainsFold(keyword), resourceent.SearchKeywordsContainsFold(keyword),
			resourceent.HasAddressWith(addressent.Or(addressent.AddressDetailContainsFold(keyword), addressent.ContactNameContainsFold(keyword), addressent.ContactPhoneContainsFold(keyword))),
			resourceent.HasRemarkWith(remarkent.ContentContainsFold(keyword)),
			resourceent.HasPartyWith(partyent.Or(partyent.CompanyNameContainsFold(keyword), partyent.BusinessCodeContainsFold(keyword), partyent.AddressContainsFold(keyword), partyent.ContactNameContainsFold(keyword))),
		))
	}
	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	order := sql.OrderAsc()
	if options.SortDesc {
		order = sql.OrderDesc()
	}
	orders := []resourceent.OrderOption{resourceent.BySortOrder(), resourceent.ByUpdatedAt(sql.OrderDesc())}
	switch options.SortBy {
	case "short_name":
		orders = []resourceent.OrderOption{resourceent.ByShortName(order), resourceent.ByID()}
	case "updated_at":
		orders = []resourceent.OrderOption{resourceent.ByUpdatedAt(order), resourceent.ByID()}
	case "sort_order":
		orders = []resourceent.OrderOption{resourceent.BySortOrder(order), resourceent.ByID()}
	}
	items, err := query.Order(orders...).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, 0, err
	}
	result := make([]*biz.EnterpriseResource, 0, len(items))
	for _, item := range items {
		result = append(result, enterpriseResourceToBiz(item))
	}
	return result, int64(total), nil
}

func (r *enterpriseResourceRepo) Get(ctx context.Context, organizationID, id uuid.UUID) (*biz.EnterpriseResource, error) {
	item, err := enterpriseResourceQuery(r.data.db).Where(resourceent.IDEQ(id), resourceent.OrganizationIDEQ(organizationID)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, biz.ErrEnterpriseResourceNotFound
	}
	if err != nil {
		return nil, err
	}
	return enterpriseResourceToBiz(item), nil
}

func (r *enterpriseResourceRepo) Create(ctx context.Context, organizationID, actorID uuid.UUID, input *biz.EnterpriseResource, audit *biz.AuditEvent) (*biz.EnterpriseResource, error) {
	var created *ent.EnterpriseResource
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		var err error
		created, err = createEnterpriseResource(ctx, tx, organizationID, actorID, input)
		if err != nil {
			return err
		}
		audit.ResourceID = created.ID.String()
		audit.Details["resource.id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, created.ID)
}

func createEnterpriseResource(ctx context.Context, tx *ent.Tx, organizationID, actorID uuid.UUID, input *biz.EnterpriseResource) (*ent.EnterpriseResource, error) {
	if err := validateEnterpriseResourceRelations(ctx, tx, organizationID, input); err != nil {
		return nil, err
	}
	created, err := tx.EnterpriseResource.Create().SetOrganizationID(organizationID).SetResourceType(resourceent.ResourceType(input.ResourceType)).SetShortName(input.ShortName).SetEnabled(input.Enabled).SetSortOrder(input.SortOrder).SetCreatedBy(actorID).SetUpdatedBy(actorID).Save(ctx)
	if err != nil {
		return nil, err
	}
	if err := createEnterpriseResourceDetail(ctx, tx, created.ID, organizationID, actorID, input); err != nil {
		return nil, err
	}
	if err := replaceEnterpriseResourceRelations(ctx, tx, created.ID, input.ResourceType, input.PartnerIDs, input.AssigneeIDs, input.AddressTypes); err != nil {
		return nil, err
	}
	return created, nil
}

func (r *enterpriseResourceRepo) Update(ctx context.Context, organizationID, actorID, id uuid.UUID, input *biz.EnterpriseResource, audit *biz.AuditEvent) (*biz.EnterpriseResource, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		existing, err := tx.EnterpriseResource.Query().Where(resourceent.IDEQ(id), resourceent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if ent.IsNotFound(err) {
			return biz.ErrEnterpriseResourceNotFound
		}
		if err != nil {
			return err
		}
		if biz.EnterpriseResourceType(existing.ResourceType) != input.ResourceType {
			return biz.ErrEnterpriseResourceInvalidArgument
		}
		if err := validateEnterpriseResourceRelations(ctx, tx, organizationID, input); err != nil {
			return err
		}
		if _, err := existing.Update().SetShortName(input.ShortName).SetEnabled(input.Enabled).SetSortOrder(input.SortOrder).SetUpdatedBy(actorID).Save(ctx); err != nil {
			return err
		}
		if err := updateEnterpriseResourceDetail(ctx, tx, id, organizationID, input); err != nil {
			return err
		}
		if input.PartnerIDs != nil || input.AssigneeIDs != nil || input.AddressTypes != nil {
			if err := replaceEnterpriseResourceRelations(ctx, tx, id, input.ResourceType, input.PartnerIDs, input.AssigneeIDs, input.AddressTypes); err != nil {
				return err
			}
		}
		if !input.Enabled {
			if _, err := tx.EnterpriseResourcePartner.Update().Where(partnerlinkent.ResourceIDEQ(id)).SetIsDefault(false).Save(ctx); err != nil {
				return err
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, id)
}

func (r *enterpriseResourceRepo) Delete(ctx context.Context, organizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		item, err := tx.EnterpriseResource.Query().Where(resourceent.IDEQ(id), resourceent.OrganizationIDEQ(organizationID)).Only(ctx)
		if ent.IsNotFound(err) {
			return biz.ErrEnterpriseResourceNotFound
		}
		if err != nil {
			return err
		}
		audit.Details["resource.type"] = string(item.ResourceType)
		if item.ResourceType == resourceent.ResourceType(biz.EnterpriseResourceTagType) {
			// 与批量关联写入使用同一行锁：先锁标签资源再统计使用，防止统计与删除之间并发写入关联
			locked, err := tx.EnterpriseResource.Query().Where(resourceent.IDEQ(id)).Order(resourceent.ByID()).ForUpdate().All(ctx)
			if err != nil || len(locked) != 1 {
				if err != nil {
					return err
				}
				return biz.ErrEnterpriseResourceNotFound
			}
			partnerCount, err := tx.EnterpriseResourcePartner.Query().Where(partnerlinkent.ResourceIDEQ(id)).Count(ctx)
			if err != nil {
				return err
			}
			orderTagCount, err := tx.OrderEnterpriseTag.Query().Where(ordertaglinkent.TagResourceIDEQ(id)).Count(ctx)
			if err != nil {
				return err
			}
			feeTagCount, err := tx.OrderFeeEnterpriseTag.Query().Where(feetaglinkent.TagResourceIDEQ(id)).Count(ctx)
			if err != nil {
				return err
			}
			billTagCount, err := tx.FinanceBillEnterpriseTag.Query().Where(billtaglinkent.TagResourceIDEQ(id)).Count(ctx)
			if err != nil {
				return err
			}
			if partnerCount > 0 || orderTagCount > 0 || feeTagCount > 0 || billTagCount > 0 {
				return biz.ErrEnterpriseTagInUse.WithMetadata(map[string]string{
					"partner_count":      stringInt(partnerCount),
					"order_count":        stringInt(orderTagCount),
					"order_fee_count":    stringInt(feeTagCount),
					"finance_bill_count": stringInt(billTagCount),
				})
			}
		}
		for _, deleteRows := range []func(context.Context) (int, error){
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseResourcePartner.Delete().Where(partnerlinkent.ResourceIDEQ(id)).Exec(ctx)
			},
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseResourceAssignee.Delete().Where(assigneeent.ResourceIDEQ(id)).Exec(ctx)
			},
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseResourceAddressType.Delete().Where(addresstypeent.ResourceIDEQ(id)).Exec(ctx)
			},
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseResourceAddress.Delete().Where(func(s *sql.Selector) { s.Where(sql.EQ(s.C("resource_id"), id)) }).Exec(ctx)
			},
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseResourceRemark.Delete().Where(func(s *sql.Selector) { s.Where(sql.EQ(s.C("resource_id"), id)) }).Exec(ctx)
			},
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseResourceImage.Delete().Where(func(s *sql.Selector) { s.Where(sql.EQ(s.C("resource_id"), id)) }).Exec(ctx)
			},
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseResourceParty.Delete().Where(partyent.ResourceIDEQ(id)).Exec(ctx)
			},
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseResourceShippingText.Delete().Where(func(s *sql.Selector) { s.Where(sql.EQ(s.C("resource_id"), id)) }).Exec(ctx)
			},
			func(ctx context.Context) (int, error) {
				return tx.EnterpriseTag.Delete().Where(tagent.ResourceIDEQ(id)).Exec(ctx)
			},
		} {
			if _, err := deleteRows(ctx); err != nil {
				return err
			}
		}
		if err := tx.EnterpriseResource.DeleteOneID(id).Exec(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

func (r *enterpriseResourceRepo) BatchPartners(ctx context.Context, organizationID uuid.UUID, resourceIDs, partnerIDs []uuid.UUID, create bool, audit *biz.AuditEvent) (int, error) {
	affected := 0
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		resources, err := tx.EnterpriseResource.Query().Where(resourceent.OrganizationIDEQ(organizationID), resourceent.IDIn(resourceIDs...)).All(ctx)
		if err != nil || len(resources) != len(uniqueEnterpriseUUIDs(resourceIDs)) {
			if err != nil {
				return err
			}
			return biz.ErrEnterpriseResourceInvalidArgument
		}
		for _, item := range resources {
			if !biz.EnterpriseResourceType(item.ResourceType).BatchAssociable() {
				return biz.ErrEnterpriseResourceInvalidArgument
			}
		}
		count, err := tx.Partner.Query().Where(partnerent.OrganizationIDEQ(organizationID), partnerent.IDIn(partnerIDs...)).Count(ctx)
		if err != nil || count != len(uniqueEnterpriseUUIDs(partnerIDs)) {
			if err != nil {
				return err
			}
			return biz.ErrPartnerNotFound
		}
		if create {
			for _, item := range resources {
				for _, partnerID := range uniqueEnterpriseUUIDs(partnerIDs) {
					exists, err := tx.EnterpriseResourcePartner.Query().Where(partnerlinkent.ResourceIDEQ(item.ID), partnerlinkent.PartnerIDEQ(partnerID)).Exist(ctx)
					if err != nil {
						return err
					}
					if exists {
						continue
					}
					if _, err := tx.EnterpriseResourcePartner.Create().SetResourceID(item.ID).SetPartnerID(partnerID).SetResourceType(partnerlinkent.ResourceType(item.ResourceType)).Save(ctx); err != nil {
						return err
					}
					affected++
				}
			}
		} else {
			affected, err = tx.EnterpriseResourcePartner.Delete().Where(partnerlinkent.ResourceIDIn(resourceIDs...), partnerlinkent.PartnerIDIn(partnerIDs...)).Exec(ctx)
			if err != nil {
				return err
			}
		}
		key := "unlinked_count"
		if create {
			key = "linked_count"
		}
		audit.Details[key] = stringInt(affected)
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *enterpriseResourceRepo) BatchAddressTypes(ctx context.Context, organizationID uuid.UUID, resourceIDs []uuid.UUID, values []biz.EnterpriseAddressType, assign bool, audit *biz.AuditEvent) (int, error) {
	affected := 0
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		count, err := tx.EnterpriseResource.Query().Where(resourceent.OrganizationIDEQ(organizationID), resourceent.ResourceTypeEQ(resourceent.ResourceTypeADDRESS), resourceent.IDIn(resourceIDs...)).Count(ctx)
		if err != nil || count != len(uniqueEnterpriseUUIDs(resourceIDs)) {
			if err != nil {
				return err
			}
			return biz.ErrEnterpriseResourceInvalidArgument
		}
		converted := make([]addresstypeent.AddressType, len(values))
		for i, value := range values {
			converted[i] = addresstypeent.AddressType(value)
		}
		if assign {
			for _, resourceID := range uniqueEnterpriseUUIDs(resourceIDs) {
				for _, value := range converted {
					exists, err := tx.EnterpriseResourceAddressType.Query().Where(addresstypeent.ResourceIDEQ(resourceID), addresstypeent.AddressTypeEQ(value)).Exist(ctx)
					if err != nil {
						return err
					}
					if !exists {
						if _, err := tx.EnterpriseResourceAddressType.Create().SetResourceID(resourceID).SetAddressType(value).Save(ctx); err != nil {
							return err
						}
						affected++
					}
				}
			}
		} else {
			affected, err = tx.EnterpriseResourceAddressType.Delete().Where(addresstypeent.ResourceIDIn(resourceIDs...), addresstypeent.AddressTypeIn(converted...)).Exec(ctx)
			if err != nil {
				return err
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *enterpriseResourceRepo) BatchAssignees(ctx context.Context, organizationID uuid.UUID, resourceIDs, userIDs []uuid.UUID, assign bool, audit *biz.AuditEvent) (int, error) {
	affected := 0
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		count, err := tx.EnterpriseResource.Query().Where(resourceent.OrganizationIDEQ(organizationID), resourceent.ResourceTypeEQ(resourceent.ResourceTypeADDRESS), resourceent.IDIn(resourceIDs...)).Count(ctx)
		if err != nil || count != len(uniqueEnterpriseUUIDs(resourceIDs)) {
			if err != nil {
				return err
			}
			return biz.ErrEnterpriseResourceInvalidArgument
		}
		members, err := tx.Membership.Query().Where(membershipent.OrganizationIDEQ(organizationID), membershipent.UserIDIn(userIDs...), membershipent.EnabledEQ(true)).Count(ctx)
		if err != nil || members != len(uniqueEnterpriseUUIDs(userIDs)) {
			if err != nil {
				return err
			}
			return biz.ErrEnterpriseResourceInvalidArgument
		}
		if assign {
			for _, resourceID := range uniqueEnterpriseUUIDs(resourceIDs) {
				for _, userID := range uniqueEnterpriseUUIDs(userIDs) {
					exists, err := tx.EnterpriseResourceAssignee.Query().Where(assigneeent.ResourceIDEQ(resourceID), assigneeent.UserIDEQ(userID)).Exist(ctx)
					if err != nil {
						return err
					}
					if !exists {
						if _, err := tx.EnterpriseResourceAssignee.Create().SetResourceID(resourceID).SetUserID(userID).Save(ctx); err != nil {
							return err
						}
						affected++
					}
				}
			}
		} else {
			affected, err = tx.EnterpriseResourceAssignee.Delete().Where(assigneeent.ResourceIDIn(resourceIDs...), assigneeent.UserIDIn(userIDs...)).Exec(ctx)
			if err != nil {
				return err
			}
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return 0, err
	}
	return affected, nil
}

func (r *enterpriseResourceRepo) ListTagGroups(ctx context.Context, organizationID uuid.UUID) ([]*biz.EnterpriseTagGroup, error) {
	items, err := r.data.db.EnterpriseTagGroup.Query().Where(taggroupent.OrganizationIDEQ(organizationID)).Order(taggroupent.BySortOrder(), taggroupent.ByName()).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.EnterpriseTagGroup, len(items))
	for i, item := range items {
		result[i] = enterpriseTagGroupToBiz(item)
	}
	return result, nil
}
func (r *enterpriseResourceRepo) CreateTagGroup(ctx context.Context, organizationID uuid.UUID, input *biz.EnterpriseTagGroup, audit *biz.AuditEvent) (*biz.EnterpriseTagGroup, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	created, err := tx.EnterpriseTagGroup.Create().SetOrganizationID(organizationID).SetName(input.Name).SetNormalizedName(strings.ToUpper(input.Name)).SetNillableColor(optionalString(input.Color)).SetSortOrder(input.SortOrder).Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	audit.ResourceID = created.ID.String()
	audit.Details["resource.id"] = created.ID.String()
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return enterpriseTagGroupToBiz(created), nil
}
func (r *enterpriseResourceRepo) UpdateTagGroup(ctx context.Context, organizationID, id uuid.UUID, input *biz.EnterpriseTagGroup, audit *biz.AuditEvent) (*biz.EnterpriseTagGroup, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.EnterpriseTagGroup.Query().Where(taggroupent.IDEQ(id), taggroupent.OrganizationIDEQ(organizationID)).Only(ctx)
	if ent.IsNotFound(err) {
		_ = tx.Rollback()
		return nil, biz.ErrEnterpriseResourceNotFound
	}
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	builder := existing.Update().SetName(input.Name).SetNormalizedName(strings.ToUpper(input.Name)).SetSortOrder(input.SortOrder)
	if input.Color == "" {
		builder.ClearColor()
	} else {
		builder.SetColor(input.Color)
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return enterpriseTagGroupToBiz(updated), nil
}
func (r *enterpriseResourceRepo) DeleteTagGroup(ctx context.Context, organizationID, id uuid.UUID, audit *biz.AuditEvent) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	group, err := tx.EnterpriseTagGroup.Query().Where(taggroupent.IDEQ(id), taggroupent.OrganizationIDEQ(organizationID)).Only(ctx)
	if ent.IsNotFound(err) {
		_ = tx.Rollback()
		return biz.ErrEnterpriseResourceNotFound
	}
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	has, err := group.QueryTags().Exist(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if has {
		_ = tx.Rollback()
		return biz.ErrEnterpriseTagGroupNotEmpty
	}
	if err := tx.EnterpriseTagGroup.DeleteOne(group).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
func (r *enterpriseResourceRepo) FindImportConflicts(ctx context.Context, organizationID uuid.UUID, inputs []*biz.EnterpriseResource) ([]*biz.EnterpriseResourceImportConflict, error) {
	return findEnterpriseResourceImportConflicts(ctx, r.data.db.EnterpriseResourceParty, organizationID, inputs)
}

func (r *enterpriseResourceRepo) Import(ctx context.Context, organizationID, actorID uuid.UUID, inputs []*biz.EnterpriseResource, overwriteConflicts bool, audit *biz.AuditEvent) ([]*biz.EnterpriseResource, int, int, []*biz.EnterpriseResourceImportConflict, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, 0, 0, nil, err
	}
	conflicts, err := findEnterpriseResourceImportConflicts(ctx, tx.EnterpriseResourceParty, organizationID, inputs)
	if err != nil {
		_ = tx.Rollback()
		return nil, 0, 0, nil, err
	}
	if len(conflicts) > 0 && !overwriteConflicts {
		_ = tx.Rollback()
		return inputs, 0, 0, conflicts, nil
	}
	conflictsByRow := make(map[int][]*biz.EnterpriseResourceImportConflict)
	for _, conflict := range conflicts {
		conflictsByRow[conflict.RowNumber] = append(conflictsByRow[conflict.RowNumber], conflict)
	}
	for _, rowConflicts := range conflictsByRow {
		if len(rowConflicts) > 1 {
			_ = tx.Rollback()
			return nil, 0, 0, conflicts, biz.ErrEnterpriseResourceImportAmbiguous
		}
	}
	resourceIDsToLock := make([]uuid.UUID, 0, len(conflictsByRow))
	resourceRows := make(map[uuid.UUID]int, len(conflictsByRow))
	for rowNumber, rowConflicts := range conflictsByRow {
		resourceID := rowConflicts[0].ExistingResourceID
		if existingRow, exists := resourceRows[resourceID]; exists && existingRow != rowNumber {
			_ = tx.Rollback()
			return nil, 0, 0, conflicts, biz.ErrEnterpriseResourceImportAmbiguous
		}
		resourceRows[resourceID] = rowNumber
		resourceIDsToLock = append(resourceIDsToLock, resourceID)
	}
	sort.Slice(resourceIDsToLock, func(i, j int) bool { return resourceIDsToLock[i].String() < resourceIDsToLock[j].String() })
	if len(resourceIDsToLock) > 0 {
		locked, err := tx.EnterpriseResource.Query().Where(resourceent.IDIn(resourceIDsToLock...), resourceent.OrganizationIDEQ(organizationID)).Order(resourceent.ByID()).ForUpdate().All(ctx)
		if err != nil {
			_ = tx.Rollback()
			return nil, 0, 0, nil, err
		}
		if len(locked) != len(resourceIDsToLock) {
			_ = tx.Rollback()
			return nil, 0, 0, nil, biz.ErrEnterpriseResourceInvalidArgument
		}
	}
	createdCount := 0
	updatedCount := 0
	for index, input := range inputs {
		rowConflicts := conflictsByRow[index+1]
		if len(rowConflicts) == 0 {
			_, err := createEnterpriseResource(ctx, tx, organizationID, actorID, input)
			if err != nil {
				_ = tx.Rollback()
				return nil, 0, 0, nil, err
			}
			createdCount++
			continue
		}
		id := rowConflicts[0].ExistingResourceID
		if err := validateEnterpriseResourceRelations(ctx, tx, organizationID, input); err != nil {
			_ = tx.Rollback()
			return nil, 0, 0, nil, err
		}
		if _, err := tx.EnterpriseResource.Update().Where(resourceent.IDEQ(id), resourceent.OrganizationIDEQ(organizationID), resourceent.ResourceTypeEQ(resourceent.ResourceType(input.ResourceType))).SetShortName(input.ShortName).SetUpdatedBy(actorID).Save(ctx); err != nil {
			_ = tx.Rollback()
			return nil, 0, 0, nil, err
		}
		if err := updateImportedEnterpriseResourceParty(ctx, tx, id, input.Party); err != nil {
			_ = tx.Rollback()
			return nil, 0, 0, nil, err
		}
		updatedCount++
	}
	audit.Details["created_count"] = stringInt(createdCount)
	audit.Details["updated_count"] = stringInt(updatedCount)
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, 0, 0, nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, 0, 0, nil, err
	}
	return inputs, createdCount, updatedCount, conflicts, nil
}

func findEnterpriseResourceImportConflicts(ctx context.Context, client *ent.EnterpriseResourcePartyClient, organizationID uuid.UUID, inputs []*biz.EnterpriseResource) ([]*biz.EnterpriseResourceImportConflict, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	conditions := make([]predicate.EnterpriseResourceParty, 0, len(inputs)*2)
	codes := make([]string, 0, len(inputs))
	for _, input := range inputs {
		conditions = append(conditions, partyent.CompanyNameEqualFold(input.Party.CompanyName))
		if input.Party.BusinessCode != "" {
			codes = append(codes, strings.ToUpper(input.Party.BusinessCode))
		}
	}
	if len(codes) > 0 {
		conditions = append(conditions, partyent.NormalizedBusinessCodeIn(codes...))
	}
	items, err := client.Query().Where(
		partyent.OrganizationIDEQ(organizationID),
		partyent.ResourceTypeEQ(partyent.ResourceType(inputs[0].ResourceType)),
		partyent.Or(conditions...),
	).WithResource().All(ctx)
	if err != nil {
		return nil, err
	}
	byCode := make(map[string][]*ent.EnterpriseResourceParty)
	byName := make(map[string][]*ent.EnterpriseResourceParty)
	for _, item := range items {
		if item.NormalizedBusinessCode != nil {
			byCode[*item.NormalizedBusinessCode] = append(byCode[*item.NormalizedBusinessCode], item)
		}
		nameKey := strings.ToUpper(strings.TrimSpace(item.CompanyName))
		byName[nameKey] = append(byName[nameKey], item)
	}
	conflicts := make([]*biz.EnterpriseResourceImportConflict, 0)
	for index, input := range inputs {
		matches := make(map[uuid.UUID]*biz.EnterpriseResourceImportConflict)
		addMatches := func(items []*ent.EnterpriseResourceParty, field string) {
			for _, item := range items {
				resource := item.Edges.Resource
				if resource == nil {
					continue
				}
				conflict, exists := matches[resource.ID]
				if !exists {
					conflict = &biz.EnterpriseResourceImportConflict{RowNumber: index + 1, ExistingResourceID: resource.ID, ExistingShortName: resource.ShortName}
					matches[resource.ID] = conflict
				}
				conflict.MatchedFields = append(conflict.MatchedFields, field)
			}
		}
		if input.Party.BusinessCode != "" {
			addMatches(byCode[strings.ToUpper(input.Party.BusinessCode)], "business_code")
		}
		addMatches(byName[strings.ToUpper(strings.TrimSpace(input.Party.CompanyName))], "company_name")
		for _, conflict := range matches {
			sort.Strings(conflict.MatchedFields)
			conflicts = append(conflicts, conflict)
		}
	}
	sort.Slice(conflicts, func(i, j int) bool {
		if conflicts[i].RowNumber != conflicts[j].RowNumber {
			return conflicts[i].RowNumber < conflicts[j].RowNumber
		}
		return conflicts[i].ExistingResourceID.String() < conflicts[j].ExistingResourceID.String()
	})
	return conflicts, nil
}

func validateEnterpriseResourceRelations(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, input *biz.EnterpriseResource) error {
	if input.Address != nil {
		if err := validateEnterpriseAddressRegions(ctx, tx, input.Address); err != nil {
			return err
		}
	}
	if input.PartnerIDs != nil {
		count, err := tx.Partner.Query().Where(partnerent.OrganizationIDEQ(organizationID), partnerent.IDIn(input.PartnerIDs...)).Count(ctx)
		if err != nil {
			return err
		}
		if count != len(uniqueEnterpriseUUIDs(input.PartnerIDs)) {
			return biz.ErrPartnerNotFound
		}
	}
	if input.AssigneeIDs != nil {
		if input.ResourceType != biz.EnterpriseResourceAddressType {
			return biz.ErrEnterpriseResourceInvalidArgument
		}
		count, err := tx.Membership.Query().Where(membershipent.OrganizationIDEQ(organizationID), membershipent.UserIDIn(input.AssigneeIDs...), membershipent.EnabledEQ(true)).Count(ctx)
		if err != nil {
			return err
		}
		if count != len(uniqueEnterpriseUUIDs(input.AssigneeIDs)) {
			return biz.ErrEnterpriseResourceInvalidArgument
		}
	}
	if input.AddressTypes != nil && input.ResourceType != biz.EnterpriseResourceAddressType {
		return biz.ErrEnterpriseResourceInvalidArgument
	}
	if input.Tag != nil {
		exists, err := tx.EnterpriseTagGroup.Query().Where(taggroupent.IDEQ(input.Tag.GroupID), taggroupent.OrganizationIDEQ(organizationID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return biz.ErrEnterpriseResourceInvalidArgument
		}
	}
	return nil
}

func validateEnterpriseAddressRegions(ctx context.Context, tx *ent.Tx, address *biz.EnterpriseResourceAddress) error {
	codes := []string{address.ProvinceCode, address.CityCode, address.DistrictCode}
	if address.CountryCode != "CN" {
		for _, code := range codes {
			if code != "" {
				return biz.ErrEnterpriseResourceInvalidArgument
			}
		}
		return nil
	}
	if address.CityCode != "" && address.ProvinceCode == "" || address.DistrictCode != "" && address.CityCode == "" {
		return biz.ErrEnterpriseResourceInvalidArgument
	}
	levels := []int{1, 2, 3}
	parents := []string{"", address.ProvinceCode, address.CityCode}
	for i, code := range codes {
		if code == "" {
			continue
		}
		query := tx.AdministrativeRegion.Query().Where(regionent.CodeEQ(code), regionent.LevelEQ(levels[i]), regionent.EnabledEQ(true))
		if parents[i] != "" {
			query.Where(regionent.ParentCodeEQ(parents[i]))
		}
		exists, err := query.Exist(ctx)
		if err != nil {
			return err
		}
		if !exists {
			return biz.ErrEnterpriseResourceInvalidArgument
		}
	}
	return nil
}

func createEnterpriseResourceDetail(ctx context.Context, tx *ent.Tx, id, organizationID, actorID uuid.UUID, input *biz.EnterpriseResource) error {
	switch input.ResourceType {
	case biz.EnterpriseResourceAddressType:
		v := input.Address
		_, err := tx.EnterpriseResourceAddress.Create().SetResourceID(id).SetNillableContactName(optionalString(v.ContactName)).SetNillableContactPhone(optionalString(v.ContactPhone)).SetCountryCode(v.CountryCode).SetNillableProvinceCode(optionalString(v.ProvinceCode)).SetNillableCityCode(optionalString(v.CityCode)).SetNillableDistrictCode(optionalString(v.DistrictCode)).SetAddressDetail(v.AddressDetail).SetNillableRemark(optionalString(v.Remark)).Save(ctx)
		return err
	case biz.EnterpriseResourceRemarkType:
		_, err := tx.EnterpriseResourceRemark.Create().SetResourceID(id).SetRemarkType(remarkent.RemarkType(input.Remark.RemarkType)).SetContent(input.Remark.Content).Save(ctx)
		return err
	case biz.EnterpriseResourceImageType:
		v := input.Image
		builder := tx.EnterpriseResourceImage.Create().SetResourceID(id).SetFileName(v.FileName).SetMimeType(v.MIMEType).SetFileSize(v.FileSize).SetObjectKey(v.ObjectKey).SetChecksum(v.Checksum).SetUploadedBy(actorID)
		if v.Width != nil {
			builder.SetWidth(*v.Width)
		}
		if v.Height != nil {
			builder.SetHeight(*v.Height)
		}
		_, err := builder.Save(ctx)
		return err
	case biz.EnterpriseResourceTagType:
		_, err := tx.EnterpriseTag.Create().SetResourceID(id).SetOrganizationID(organizationID).SetGroupID(input.Tag.GroupID).SetNormalizedName(strings.ToUpper(input.ShortName)).SetSortOrder(input.SortOrder).Save(ctx)
		return err
	default:
		v := input.Party
		_, err := tx.EnterpriseResourceParty.Create().SetResourceID(id).SetOrganizationID(organizationID).SetResourceType(partyent.ResourceType(input.ResourceType)).SetCompanyName(v.CompanyName).SetNillableBusinessCode(optionalString(v.BusinessCode)).SetNillableNormalizedBusinessCode(optionalString(strings.ToUpper(v.BusinessCode))).SetNillableAddress(optionalString(v.Address)).SetCountryCode(v.CountryCode).SetNillableContactName(optionalString(v.ContactName)).SetNillableContactPhone(optionalString(v.ContactPhone)).SetNillableEmail(optionalString(v.Email)).SetNillableTaxIdentifier(optionalString(v.TaxIdentifier)).SetNillableAeoCode(optionalString(v.AEOCode)).SetCustomDisplay(v.CustomDisplay).SetNillableDisplayContent(optionalString(v.DisplayContent)).SetNillableRemark(optionalString(v.Remark)).Save(ctx)
		return err
	}
}

func updateEnterpriseResourceDetail(ctx context.Context, tx *ent.Tx, id, organizationID uuid.UUID, input *biz.EnterpriseResource) error {
	switch input.ResourceType {
	case biz.EnterpriseResourceAddressType:
		v := input.Address
		builder := tx.EnterpriseResourceAddress.Update().Where(func(s *sql.Selector) { s.Where(sql.EQ(s.C("resource_id"), id)) }).SetCountryCode(v.CountryCode).SetAddressDetail(v.AddressDetail).ClearContactName().ClearContactPhone().ClearProvinceCode().ClearCityCode().ClearDistrictCode().ClearRemark()
		builder.SetNillableContactName(optionalString(v.ContactName)).SetNillableContactPhone(optionalString(v.ContactPhone)).SetNillableProvinceCode(optionalString(v.ProvinceCode)).SetNillableCityCode(optionalString(v.CityCode)).SetNillableDistrictCode(optionalString(v.DistrictCode)).SetNillableRemark(optionalString(v.Remark))
		_, err := builder.Save(ctx)
		return err
	case biz.EnterpriseResourceRemarkType:
		_, err := tx.EnterpriseResourceRemark.Update().Where(func(s *sql.Selector) { s.Where(sql.EQ(s.C("resource_id"), id)) }).SetRemarkType(remarkent.RemarkType(input.Remark.RemarkType)).SetContent(input.Remark.Content).Save(ctx)
		return err
	case biz.EnterpriseResourceImageType:
		return biz.ErrEnterpriseResourceInvalidArgument
	case biz.EnterpriseResourceTagType:
		_, err := tx.EnterpriseTag.Update().Where(tagent.ResourceIDEQ(id)).SetGroupID(input.Tag.GroupID).SetNormalizedName(strings.ToUpper(input.ShortName)).SetSortOrder(input.SortOrder).Save(ctx)
		return err
	default:
		v := input.Party
		builder := tx.EnterpriseResourceParty.Update().Where(partyent.ResourceIDEQ(id)).SetCompanyName(v.CompanyName).SetCountryCode(v.CountryCode).SetCustomDisplay(v.CustomDisplay).ClearBusinessCode().ClearNormalizedBusinessCode().ClearAddress().ClearContactName().ClearContactPhone().ClearEmail().ClearTaxIdentifier().ClearAeoCode().ClearDisplayContent().ClearRemark()
		builder.SetNillableBusinessCode(optionalString(v.BusinessCode)).SetNillableNormalizedBusinessCode(optionalString(strings.ToUpper(v.BusinessCode))).SetNillableAddress(optionalString(v.Address)).SetNillableContactName(optionalString(v.ContactName)).SetNillableContactPhone(optionalString(v.ContactPhone)).SetNillableEmail(optionalString(v.Email)).SetNillableTaxIdentifier(optionalString(v.TaxIdentifier)).SetNillableAeoCode(optionalString(v.AEOCode)).SetNillableDisplayContent(optionalString(v.DisplayContent)).SetNillableRemark(optionalString(v.Remark))
		_, err := builder.Save(ctx)
		return err
	}
}

func updateImportedEnterpriseResourceParty(ctx context.Context, tx *ent.Tx, id uuid.UUID, input *biz.EnterpriseResourceParty) error {
	builder := tx.EnterpriseResourceParty.Update().Where(partyent.ResourceIDEQ(id)).
		SetCompanyName(input.CompanyName).
		SetCountryCode(input.CountryCode).
		ClearBusinessCode().
		ClearNormalizedBusinessCode().
		ClearAddress().
		ClearContactName().
		ClearContactPhone().
		ClearEmail().
		ClearTaxIdentifier().
		ClearAeoCode()
	builder.SetNillableBusinessCode(optionalString(input.BusinessCode)).
		SetNillableNormalizedBusinessCode(optionalString(strings.ToUpper(input.BusinessCode))).
		SetNillableAddress(optionalString(input.Address)).
		SetNillableContactName(optionalString(input.ContactName)).
		SetNillableContactPhone(optionalString(input.ContactPhone)).
		SetNillableEmail(optionalString(input.Email)).
		SetNillableTaxIdentifier(optionalString(input.TaxIdentifier)).
		SetNillableAeoCode(optionalString(input.AEOCode))
	_, err := builder.Save(ctx)
	return err
}

func replaceEnterpriseResourceRelations(ctx context.Context, tx *ent.Tx, id uuid.UUID, resourceType biz.EnterpriseResourceType, partnerIDs, assigneeIDs []uuid.UUID, addressTypes []biz.EnterpriseAddressType) error {
	if partnerIDs != nil {
		existing, err := tx.EnterpriseResourcePartner.Query().Where(partnerlinkent.ResourceIDEQ(id)).All(ctx)
		if err != nil {
			return err
		}
		removedIDs, addedIDs := enterprisePartnerRelationChanges(existing, partnerIDs)
		if len(removedIDs) > 0 {
			if _, err := tx.EnterpriseResourcePartner.Delete().Where(partnerlinkent.IDIn(removedIDs...)).Exec(ctx); err != nil {
				return err
			}
		}
		for _, partnerID := range addedIDs {
			if _, err := tx.EnterpriseResourcePartner.Create().SetResourceID(id).SetPartnerID(partnerID).SetResourceType(partnerlinkent.ResourceType(resourceType)).Save(ctx); err != nil {
				return err
			}
		}
	}
	if assigneeIDs != nil {
		if _, err := tx.EnterpriseResourceAssignee.Delete().Where(assigneeent.ResourceIDEQ(id)).Exec(ctx); err != nil {
			return err
		}
		for _, userID := range uniqueEnterpriseUUIDs(assigneeIDs) {
			if _, err := tx.EnterpriseResourceAssignee.Create().SetResourceID(id).SetUserID(userID).Save(ctx); err != nil {
				return err
			}
		}
	}
	if addressTypes != nil {
		if _, err := tx.EnterpriseResourceAddressType.Delete().Where(addresstypeent.ResourceIDEQ(id)).Exec(ctx); err != nil {
			return err
		}
		for _, value := range uniqueAddressTypes(addressTypes) {
			if _, err := tx.EnterpriseResourceAddressType.Create().SetResourceID(id).SetAddressType(addresstypeent.AddressType(value)).Save(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}

func enterprisePartnerRelationChanges(existing []*ent.EnterpriseResourcePartner, desired []uuid.UUID) ([]uuid.UUID, []uuid.UUID) {
	desiredSet := make(map[uuid.UUID]struct{}, len(desired))
	for _, partnerID := range uniqueEnterpriseUUIDs(desired) {
		desiredSet[partnerID] = struct{}{}
	}
	existingSet := make(map[uuid.UUID]struct{}, len(existing))
	removedIDs := make([]uuid.UUID, 0)
	for _, link := range existing {
		existingSet[link.PartnerID] = struct{}{}
		if _, keep := desiredSet[link.PartnerID]; !keep {
			removedIDs = append(removedIDs, link.ID)
		}
	}
	addedIDs := make([]uuid.UUID, 0)
	for _, partnerID := range uniqueEnterpriseUUIDs(desired) {
		if _, exists := existingSet[partnerID]; !exists {
			addedIDs = append(addedIDs, partnerID)
		}
	}
	return removedIDs, addedIDs
}

func enterpriseResourceToBiz(item *ent.EnterpriseResource) *biz.EnterpriseResource {
	result := &biz.EnterpriseResource{ID: item.ID, OrganizationID: item.OrganizationID, ResourceType: biz.EnterpriseResourceType(item.ResourceType), ShortName: item.ShortName, Enabled: item.Enabled, SortOrder: item.SortOrder, CreatedBy: item.CreatedBy, UpdatedBy: item.UpdatedBy, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
	for _, link := range item.Edges.PartnerLinks {
		result.PartnerIDs = append(result.PartnerIDs, link.PartnerID)
	}
	for _, link := range item.Edges.Assignees {
		result.AssigneeIDs = append(result.AssigneeIDs, link.UserID)
	}
	for _, link := range item.Edges.AddressTypes {
		result.AddressTypes = append(result.AddressTypes, biz.EnterpriseAddressType(link.AddressType))
	}
	if v := item.Edges.Address; v != nil {
		result.Address = &biz.EnterpriseResourceAddress{ContactName: stringValue(v.ContactName), ContactPhone: stringValue(v.ContactPhone), CountryCode: v.CountryCode, ProvinceCode: stringValue(v.ProvinceCode), CityCode: stringValue(v.CityCode), DistrictCode: stringValue(v.DistrictCode), AddressDetail: v.AddressDetail, Remark: stringValue(v.Remark)}
	}
	if v := item.Edges.Remark; v != nil {
		result.Remark = &biz.EnterpriseResourceRemark{RemarkType: biz.EnterpriseRemarkType(v.RemarkType), Content: v.Content}
	}
	if v := item.Edges.Party; v != nil {
		result.Party = &biz.EnterpriseResourceParty{CompanyName: v.CompanyName, BusinessCode: stringValue(v.BusinessCode), Address: stringValue(v.Address), CountryCode: v.CountryCode, ContactName: stringValue(v.ContactName), ContactPhone: stringValue(v.ContactPhone), Email: stringValue(v.Email), TaxIdentifier: stringValue(v.TaxIdentifier), AEOCode: stringValue(v.AeoCode), CustomDisplay: v.CustomDisplay, DisplayContent: stringValue(v.DisplayContent), Remark: stringValue(v.Remark)}
	}
	if v := item.Edges.Image; v != nil {
		result.Image = &biz.EnterpriseResourceImage{FileName: v.FileName, MIMEType: v.MimeType, FileSize: v.FileSize, ObjectKey: v.ObjectKey, Checksum: v.Checksum, Width: v.Width, Height: v.Height}
	}
	if v := item.Edges.Tag; v != nil {
		result.Tag = &biz.EnterpriseResourceTag{GroupID: v.GroupID}
	}
	return result
}
func enterpriseTagGroupToBiz(item *ent.EnterpriseTagGroup) *biz.EnterpriseTagGroup {
	return &biz.EnterpriseTagGroup{ID: item.ID, OrganizationID: item.OrganizationID, Name: item.Name, Color: stringValue(item.Color), SortOrder: item.SortOrder, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
func uniqueEnterpriseUUIDs(values []uuid.UUID) []uuid.UUID {
	result := make([]uuid.UUID, 0, len(values))
	seen := make(map[uuid.UUID]struct{})
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}
func uniqueAddressTypes(values []biz.EnterpriseAddressType) []biz.EnterpriseAddressType {
	result := make([]biz.EnterpriseAddressType, 0, len(values))
	seen := make(map[biz.EnterpriseAddressType]struct{})
	for _, value := range values {
		if _, ok := seen[value]; !ok {
			seen[value] = struct{}{}
			result = append(result, value)
		}
	}
	return result
}
func stringInt(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 12)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
