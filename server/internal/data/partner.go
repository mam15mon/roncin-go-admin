package data

import (
	"context"
	"strconv"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	administrativeregionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/administrativeregion"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraliasent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneralias"
	partnerassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
	partnercontactent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnercontact"
	partnerprofileent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerprofile"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	partnersettlementruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnersettlementrule"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

type partnerRepo struct{ data *Data }

func NewPartnerRepo(data *Data) biz.PartnerRepo { return &partnerRepo{data: data} }

func (r *partnerRepo) Get(ctx context.Context, organizationID, id uuid.UUID) (*biz.Partner, error) {
	item, err := withPartnerEdges(r.data.db.Partner.Query().Where(partnerent.IDEQ(id), partnerent.OrganizationIDEQ(organizationID))).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerNotFound
		}
		return nil, err
	}
	return partnerToBiz(item), nil
}

func (r *partnerRepo) List(ctx context.Context, organizationID uuid.UUID, options biz.PartnerListOptions) (*biz.PartnerList, error) {
	query := r.data.db.Partner.Query().Where(partnerent.OrganizationIDEQ(organizationID))
	if options.Keyword != "" {
		query.Where(partnerent.Or(
			partnerent.CodeContainsFold(options.Keyword),
			partnerent.LegalNameContainsFold(options.Keyword),
			partnerent.SearchKeywordsContainsFold(options.Keyword),
			partnerent.UnifiedSocialCreditCodeContainsFold(options.Keyword),
			partnerent.HasAliasesWith(partneraliasent.Or(
				partneraliasent.AliasNameContainsFold(options.Keyword),
				partneraliasent.SearchKeywordsContainsFold(options.Keyword),
			)),
			partnerent.HasContactsWith(partnercontactent.Or(
				partnercontactent.NameContainsFold(options.Keyword),
				partnercontactent.PhoneContainsFold(options.Keyword),
				partnercontactent.EmailContainsFold(options.Keyword),
			)),
		))
	}
	if options.Role != "" {
		query.Where(partnerent.HasRolesWith(
			partnerroleent.RoleTypeEQ(partnerroleent.RoleType(options.Role)),
			partnerroleent.EnabledEQ(true),
		))
	}
	if options.Enabled != nil {
		query.Where(partnerent.EnabledEQ(*options.Enabled))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := withPartnerEdges(query).
		Order(partnerent.ByLegalName()).
		Offset((options.Page - 1) * options.PageSize).
		Limit(options.PageSize).
		All(ctx)
	if err != nil {
		return nil, err
	}
	partners := make([]*biz.Partner, 0, len(items))
	for _, item := range items {
		partners = append(partners, partnerToBiz(item))
	}
	return &biz.PartnerList{Items: partners, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *partnerRepo) ListAssignmentOptions(ctx context.Context, organizationID uuid.UUID, options biz.SelectorListOptions) (*biz.PagedList[*biz.PartnerAssignmentOption], error) {
	organizations, err := r.data.db.Organization.Query().
		Select(organizationent.FieldID, organizationent.FieldParentID).
		All(ctx)
	if err != nil {
		return nil, err
	}
	parentByID := make(map[uuid.UUID]*uuid.UUID, len(organizations))
	for _, organization := range organizations {
		parentByID[organization.ID] = organization.ParentID
	}
	organizationIDs := make([]uuid.UUID, 0, len(organizations))
	for _, organization := range organizations {
		if organizationWithinRoot(parentByID, organizationID, organization.ID) {
			organizationIDs = append(organizationIDs, organization.ID)
		}
	}
	query := r.data.db.Membership.Query().Where(
		membershipent.OrganizationIDIn(organizationIDs...),
		membershipent.EnabledEQ(true),
		membershipent.HasUserWith(userent.EnabledEQ(true)),
		membershipent.HasOrganizationWith(organizationent.EnabledEQ(true)),
	)
	if options.Keyword != "" {
		query.Where(membershipent.Or(
			membershipent.HasUserWith(userent.Or(userent.UsernameContainsFold(options.Keyword), userent.DisplayNameContainsFold(options.Keyword), userent.SearchKeywordsContainsFold(options.Keyword))),
			membershipent.HasOrganizationWith(organizationent.Or(organizationent.CodeContainsFold(options.Keyword), organizationent.NameContainsFold(options.Keyword), organizationent.SearchKeywordsContainsFold(options.Keyword))),
		))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	memberships, err := query.WithUser().WithOrganization().
		Order(membershipent.ByUserField(userent.FieldDisplayName), membershipent.ByOrganizationField(organizationent.FieldName)).
		Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.PartnerAssignmentOption, 0, len(memberships))
	for _, item := range memberships {
		result = append(result, &biz.PartnerAssignmentOption{
			UserID: item.UserID, DisplayName: item.Edges.User.DisplayName,
			OrganizationID: item.OrganizationID, OrganizationName: item.Edges.Organization.Name,
			MembershipEnabled: item.Enabled,
		})
	}
	return &biz.PagedList[*biz.PartnerAssignmentOption]{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func (r *partnerRepo) ListAuditLogs(ctx context.Context, organizationID, partnerID uuid.UUID, page, pageSize int) (*biz.PartnerAuditLogList, error) {
	partnerExists, err := r.data.db.Partner.Query().Where(partnerent.IDEQ(partnerID), partnerent.OrganizationIDEQ(organizationID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !partnerExists {
		return nil, biz.ErrPartnerNotFound
	}
	query := r.data.db.AuditLog.Query().Where(
		auditlogent.OrganizationIDEQ(organizationID),
		auditlogent.ResourceTypeEQ("partner"),
		auditlogent.ResourceIDEQ(partnerID.String()),
	)
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(auditlogent.ByCreatedAt(entsql.OrderDesc())).Offset((page - 1) * pageSize).Limit(pageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	userIDs := make([]uuid.UUID, 0, len(items))
	seenUserIDs := make(map[uuid.UUID]struct{}, len(items))
	for _, item := range items {
		if item.UserID != nil {
			if _, exists := seenUserIDs[*item.UserID]; !exists {
				seenUserIDs[*item.UserID] = struct{}{}
				userIDs = append(userIDs, *item.UserID)
			}
		}
	}
	displayNames := make(map[uuid.UUID]string, len(userIDs))
	if len(userIDs) > 0 {
		users, err := r.data.db.User.Query().Where(userent.IDIn(userIDs...)).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, user := range users {
			displayNames[user.ID] = user.DisplayName
		}
	}
	result := make([]*biz.PartnerAuditLog, 0, len(items))
	for _, item := range items {
		log, err := auditLogToBiz(item)
		if err != nil {
			return nil, err
		}
		entry := &biz.PartnerAuditLog{Log: log}
		if item.UserID != nil {
			entry.UserDisplayName = displayNames[*item.UserID]
		}
		result = append(result, entry)
	}
	return &biz.PartnerAuditLogList{Items: result, Total: total, Page: page, PageSize: pageSize}, nil
}

func (r *partnerRepo) Create(ctx context.Context, organizationID uuid.UUID, input *biz.Partner, audit *biz.AuditEvent) (*biz.Partner, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	create := tx.Partner.Create().
		SetOrganizationID(organizationID).
		SetCode(input.Code).
		SetLegalName(input.LegalName).
		SetNormalizedName(input.NormalizedName).
		SetRegisteredAddress(input.RegisteredAddress).
		SetEnabled(true)
	if input.UnifiedSocialCreditCode != "" {
		create.SetUnifiedSocialCreditCode(input.UnifiedSocialCreditCode)
	}
	created, err := create.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, mapPartnerConstraint(err)
	}
	if err := createPartnerChildren(ctx, tx, organizationID, created.ID, input); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	audit.ResourceID = created.ID.String()
	audit.Details["partner.id"] = created.ID.String()
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, created.ID)
}

func (r *partnerRepo) Update(ctx context.Context, organizationID, id uuid.UUID, input *biz.Partner, audit *biz.AuditEvent) (*biz.PartnerUpdateResult, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := tx.Partner.Query().
		Where(partnerent.IDEQ(id), partnerent.OrganizationIDEQ(organizationID)).
		WithRoles(func(query *ent.PartnerRoleQuery) { query.Order(partnerroleent.ByRoleType()) }).
		ForUpdate().
		Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerNotFound
		}
		return nil, err
	}
	previousRoles := partnerRolesToBiz(existing.Edges.Roles)
	update := existing.Update().
		SetLegalName(input.LegalName).
		SetNormalizedName(input.NormalizedName).
		SetRegisteredAddress(input.RegisteredAddress).
		SetEnabled(input.Enabled)
	if input.UnifiedSocialCreditCode == "" {
		update.ClearUnifiedSocialCreditCode()
	} else {
		update.SetUnifiedSocialCreditCode(input.UnifiedSocialCreditCode)
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, mapPartnerConstraint(err)
	}
	if err := replacePartnerRoles(ctx, tx, id, existing.Edges.Roles, input.Roles); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := savePartnerRoleSettlementRules(ctx, tx, id, input.Roles); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.PartnerContact.Delete().Where(partnercontactent.PartnerIDEQ(id)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.PartnerAlias.Delete().Where(partneraliasent.PartnerIDEQ(id)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := createPartnerContacts(ctx, tx, id, input.Contacts); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := createPartnerAliases(ctx, tx, id, input.Aliases); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := replacePartnerProfile(ctx, tx, id, input.Profile); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := replacePartnerAssignments(ctx, tx, organizationID, id, input.Assignments); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	audit.Details["from_roles"] = partnerRolesAuditValue(previousRoles)
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	updated, err := r.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	return &biz.PartnerUpdateResult{Partner: updated, PreviousRoles: previousRoles}, nil
}

func (r *partnerRepo) SetSupplierBlacklist(ctx context.Context, organizationID, id uuid.UUID, input biz.PartnerBlacklistUpdate, audit *biz.AuditEvent) (*biz.PartnerBlacklistResult, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Partner.Query().Where(partnerent.IDEQ(id), partnerent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerNotFound
		}
		return nil, err
	}
	role, err := tx.PartnerRole.Query().Where(
		partnerroleent.PartnerIDEQ(id),
		partnerroleent.RoleTypeEQ(partnerroleent.RoleTypeSupplier),
	).Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrPartnerSupplierRoleRequired
		}
		return nil, err
	}
	previouslyBlacklisted := role.Blacklisted
	update := role.Update().SetBlacklisted(input.Blacklisted)
	if input.Blacklisted {
		update.SetBlacklistReason(input.Reason).SetBlacklistedAt(input.ChangedAt).SetBlacklistedBy(input.ChangedBy)
	} else {
		update.ClearBlacklistReason().ClearBlacklistedAt().ClearBlacklistedBy()
	}
	if _, err := update.Save(ctx); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	audit.Details["previously_blacklisted"] = strconv.FormatBool(previouslyBlacklisted)
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	updated, err := r.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	return &biz.PartnerBlacklistResult{Partner: updated, PreviouslyBlacklisted: previouslyBlacklisted}, nil
}

func (r *partnerRepo) Import(ctx context.Context, organizationID uuid.UUID, mode biz.PartnerImportMode, inputs []*biz.Partner, audit *biz.AuditEvent) (*biz.PartnerImportResult, error) {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.PartnerImportResult{}
	for _, input := range inputs {
		existing, queryErr := tx.Partner.Query().
			Where(partnerent.OrganizationIDEQ(organizationID), partnerent.CodeEQ(input.Code)).
			WithRoles(func(query *ent.PartnerRoleQuery) { query.Order(partnerroleent.ByRoleType()) }).
			WithContacts(func(query *ent.PartnerContactQuery) {
				query.Order(partnercontactent.ByIsPrimary(entsql.OrderDesc()), partnercontactent.ByName())
			}).
			WithAliases(func(query *ent.PartnerAliasQuery) {
				query.Order(partneraliasent.BySortOrder(), partneraliasent.ByAliasName())
			}).
			Only(ctx)
		if queryErr != nil && !ent.IsNotFound(queryErr) {
			_ = tx.Rollback()
			return nil, queryErr
		}
		if ent.IsNotFound(queryErr) {
			create := tx.Partner.Create().
				SetOrganizationID(organizationID).
				SetCode(input.Code).
				SetLegalName(input.LegalName).
				SetNormalizedName(input.NormalizedName).
				SetRegisteredAddress(input.RegisteredAddress).
				SetEnabled(true)
			if input.UnifiedSocialCreditCode != "" {
				create.SetUnifiedSocialCreditCode(input.UnifiedSocialCreditCode)
			}
			created, saveErr := create.Save(ctx)
			if saveErr != nil {
				_ = tx.Rollback()
				return nil, mapPartnerConstraint(saveErr)
			}
			if childErr := createPartnerChildren(ctx, tx, organizationID, created.ID, input); childErr != nil {
				_ = tx.Rollback()
				return nil, childErr
			}
			result.CreatedCount++
			continue
		}
		if mode == biz.PartnerImportCreateOnly {
			_ = tx.Rollback()
			return nil, biz.ErrPartnerCodeExists
		}
		if updateErr := updatePartnerInTx(ctx, tx, organizationID, existing, input); updateErr != nil {
			_ = tx.Rollback()
			return nil, updateErr
		}
		result.UpdatedCount++
	}
	audit.Details["created_count"] = strconv.Itoa(result.CreatedCount)
	audit.Details["updated_count"] = strconv.Itoa(result.UpdatedCount)
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func updatePartnerInTx(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, existing *ent.Partner, input *biz.Partner) error {
	update := existing.Update().
		SetLegalName(input.LegalName).
		SetNormalizedName(input.NormalizedName).
		SetRegisteredAddress(input.RegisteredAddress).
		SetEnabled(true)
	if input.UnifiedSocialCreditCode == "" {
		update.ClearUnifiedSocialCreditCode()
	} else {
		update.SetUnifiedSocialCreditCode(input.UnifiedSocialCreditCode)
	}
	if _, err := update.Save(ctx); err != nil {
		return mapPartnerConstraint(err)
	}
	if err := replacePartnerRoles(ctx, tx, existing.ID, existing.Edges.Roles, input.Roles); err != nil {
		return err
	}
	if err := savePartnerRoleSettlementRules(ctx, tx, existing.ID, input.Roles); err != nil {
		return err
	}
	if _, err := tx.PartnerContact.Delete().Where(partnercontactent.PartnerIDEQ(existing.ID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.PartnerAlias.Delete().Where(partneraliasent.PartnerIDEQ(existing.ID)).Exec(ctx); err != nil {
		return err
	}
	if err := createPartnerContacts(ctx, tx, existing.ID, input.Contacts); err != nil {
		return err
	}
	if err := createPartnerAliases(ctx, tx, existing.ID, input.Aliases); err != nil {
		return err
	}
	if err := replacePartnerProfile(ctx, tx, existing.ID, input.Profile); err != nil {
		return err
	}
	return replacePartnerAssignments(ctx, tx, organizationID, existing.ID, editablePartnerAssignments(input.Assignments))
}

func partnerRolesAuditValue(roles []*biz.PartnerRole) string {
	values := make([]string, 0, len(roles))
	for _, role := range roles {
		values = append(values, string(role.Type)+":"+strconv.FormatBool(role.Enabled))
	}
	return strings.Join(values, ",")
}

func editablePartnerAssignments(assignments []*biz.PartnerAssignment) []*biz.PartnerAssignment {
	result := make([]*biz.PartnerAssignment, 0, len(assignments))
	for _, assignment := range assignments {
		if assignment.Role != biz.PartnerAssignmentCreator {
			result = append(result, assignment)
		}
	}
	return result
}

func withPartnerEdges(query *ent.PartnerQuery) *ent.PartnerQuery {
	return query.
		WithRoles(func(roleQuery *ent.PartnerRoleQuery) {
			roleQuery.Order(partnerroleent.ByRoleType()).WithSettlementRules(func(query *ent.PartnerSettlementRuleQuery) {
				query.Order(partnersettlementruleent.ByCreatedAt())
			})
		}).
		WithContacts(func(contactQuery *ent.PartnerContactQuery) {
			contactQuery.Order(partnercontactent.ByIsPrimary(entsql.OrderDesc()), partnercontactent.ByName())
		}).
		WithAliases(func(aliasQuery *ent.PartnerAliasQuery) {
			aliasQuery.Order(partneraliasent.BySortOrder(), partneraliasent.ByAliasName())
		}).
		WithProfile().
		WithAssignments(func(query *ent.PartnerAssignmentQuery) {
			query.Order(partnerassignmentent.ByRole(), partnerassignmentent.BySortOrder())
		})
}

func createPartnerChildren(ctx context.Context, tx *ent.Tx, organizationID, partnerID uuid.UUID, input *biz.Partner) error {
	for _, role := range input.Roles {
		if _, err := tx.PartnerRole.Create().
			SetPartnerID(partnerID).
			SetRoleType(partnerroleent.RoleType(role.Type)).
			SetEnabled(role.Enabled).
			Save(ctx); err != nil {
			return mapPartnerConstraint(err)
		}
	}
	if err := savePartnerRoleSettlementRules(ctx, tx, partnerID, input.Roles); err != nil {
		return err
	}
	if err := createPartnerContacts(ctx, tx, partnerID, input.Contacts); err != nil {
		return err
	}
	if err := createPartnerAliases(ctx, tx, partnerID, input.Aliases); err != nil {
		return err
	}
	if err := replacePartnerProfile(ctx, tx, partnerID, input.Profile); err != nil {
		return err
	}
	return replacePartnerAssignments(ctx, tx, organizationID, partnerID, input.Assignments)
}

func replacePartnerProfile(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, profile *biz.PartnerProfile) error {
	if _, err := tx.PartnerProfile.Delete().Where(partnerprofileent.PartnerIDEQ(partnerID)).Exec(ctx); err != nil {
		return err
	}
	if profile == nil {
		return nil
	}
	if err := validatePartnerProfileRegions(ctx, tx, profile); err != nil {
		return err
	}
	_, err := tx.PartnerProfile.Create().
		SetPartnerID(partnerID).
		SetNameEn(profile.NameEN).
		SetAddressEn(profile.AddressEN).
		SetCountryCode(profile.CountryCode).
		SetProvinceCode(profile.ProvinceCode).
		SetCityCode(profile.CityCode).
		SetDistrictCode(profile.DistrictCode).
		SetAddressDetail(profile.AddressDetail).
		SetNature(profile.Nature).
		SetDevelopmentMethod(profile.DevelopmentMethod).
		SetCustomerTypes(partnerCustomerTypesToStrings(profile.CustomerTypes)).
		SetBusinessTypes(partnerBusinessTypesToStrings(profile.BusinessTypes)).
		SetRemark(profile.Remark).
		Save(ctx)
	return err
}

func partnerCustomerTypesToStrings(values []biz.PartnerCustomerType) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func partnerBusinessTypesToStrings(values []biz.PartnerBusinessType) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func validatePartnerProfileRegions(ctx context.Context, tx *ent.Tx, profile *biz.PartnerProfile) error {
	if profile.CountryCode != "CN" || profile.ProvinceCode == "" {
		return nil
	}
	province, err := tx.AdministrativeRegion.Query().Where(
		administrativeregionent.CodeEQ(profile.ProvinceCode), administrativeregionent.LevelEQ(1), administrativeregionent.EnabledEQ(true),
	).Only(ctx)
	if err != nil {
		return biz.ErrPartnerInvalidArgument
	}
	if profile.CityCode == "" {
		return nil
	}
	city, err := tx.AdministrativeRegion.Query().Where(
		administrativeregionent.CodeEQ(profile.CityCode), administrativeregionent.LevelEQ(2),
		administrativeregionent.ParentCodeEQ(province.Code), administrativeregionent.EnabledEQ(true),
	).Only(ctx)
	if err != nil {
		return biz.ErrPartnerInvalidArgument
	}
	if profile.DistrictCode == "" {
		return nil
	}
	if _, err := tx.AdministrativeRegion.Query().Where(
		administrativeregionent.CodeEQ(profile.DistrictCode), administrativeregionent.LevelEQ(3),
		administrativeregionent.ParentCodeEQ(city.Code), administrativeregionent.EnabledEQ(true),
	).Only(ctx); err != nil {
		return biz.ErrPartnerInvalidArgument
	}
	return nil
}

func replacePartnerAssignments(ctx context.Context, tx *ent.Tx, rootOrganizationID, partnerID uuid.UUID, assignments []*biz.PartnerAssignment) error {
	if _, err := tx.PartnerAssignment.Delete().Where(
		partnerassignmentent.PartnerIDEQ(partnerID),
		partnerassignmentent.RoleNEQ(partnerassignmentent.RoleCREATOR),
	).Exec(ctx); err != nil {
		return err
	}
	if len(assignments) == 0 {
		return nil
	}
	organizations, err := tx.Organization.Query().Select(organizationent.FieldID, organizationent.FieldParentID).All(ctx)
	if err != nil {
		return err
	}
	parentByID := make(map[uuid.UUID]*uuid.UUID, len(organizations))
	for _, organization := range organizations {
		parentByID[organization.ID] = organization.ParentID
	}
	for _, assignment := range assignments {
		if !organizationWithinRoot(parentByID, rootOrganizationID, assignment.OrganizationID) {
			return biz.ErrPartnerInvalidArgument
		}
		validMembership, err := tx.Membership.Query().Where(
			membershipent.UserIDEQ(assignment.UserID), membershipent.OrganizationIDEQ(assignment.OrganizationID), membershipent.EnabledEQ(true),
			membershipent.HasUserWith(userent.EnabledEQ(true)),
		).Exist(ctx)
		if err != nil {
			return err
		}
		if !validMembership {
			return biz.ErrPartnerInvalidArgument
		}
		if _, err := tx.PartnerAssignment.Create().SetPartnerID(partnerID).SetUserID(assignment.UserID).
			SetOrganizationID(assignment.OrganizationID).SetRole(partnerassignmentent.Role(assignment.Role)).
			SetSortOrder(assignment.SortOrder).Save(ctx); err != nil {
			return mapPartnerConstraint(err)
		}
	}
	return nil
}

func organizationWithinRoot(parentByID map[uuid.UUID]*uuid.UUID, rootID, targetID uuid.UUID) bool {
	for current := targetID; current != uuid.Nil; {
		if current == rootID {
			return true
		}
		parent, exists := parentByID[current]
		if !exists || parent == nil {
			return false
		}
		current = *parent
	}
	return false
}

func createPartnerContacts(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, contacts []*biz.PartnerContact) error {
	for _, contact := range contacts {
		if _, err := tx.PartnerContact.Create().
			SetPartnerID(partnerID).
			SetName(contact.Name).
			SetPhone(contact.Phone).
			SetEmail(contact.Email).
			SetNote(contact.Note).
			SetIsPrimary(contact.IsPrimary).
			Save(ctx); err != nil {
			return mapPartnerConstraint(err)
		}
	}
	return nil
}

func createPartnerAliases(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, aliases []*biz.PartnerAlias) error {
	for _, alias := range aliases {
		if _, err := tx.PartnerAlias.Create().
			SetPartnerID(partnerID).
			SetAliasName(alias.AliasName).
			SetNormalizedAliasName(alias.NormalizedAliasName).
			SetSortOrder(alias.SortOrder).
			Save(ctx); err != nil {
			return mapPartnerConstraint(err)
		}
	}
	return nil
}

func replacePartnerRoles(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, existing []*ent.PartnerRole, requested []*biz.PartnerRole) error {
	requestedByType := make(map[biz.PartnerRoleType]*biz.PartnerRole, len(requested))
	for _, role := range requested {
		requestedByType[role.Type] = role
	}
	for _, role := range existing {
		requestedRole, keep := requestedByType[biz.PartnerRoleType(role.RoleType)]
		if !keep {
			if role.RoleType == partnerroleent.RoleTypeSupplier && role.Blacklisted {
				return biz.ErrPartnerBlacklistedSupplierRole
			}
			if _, err := role.Update().SetEnabled(false).Save(ctx); err != nil {
				return err
			}
			continue
		}
		if _, err := role.Update().SetEnabled(requestedRole.Enabled).Save(ctx); err != nil {
			return err
		}
		delete(requestedByType, biz.PartnerRoleType(role.RoleType))
	}
	for _, role := range requestedByType {
		if _, err := tx.PartnerRole.Create().
			SetPartnerID(partnerID).
			SetRoleType(partnerroleent.RoleType(role.Type)).
			SetEnabled(role.Enabled).
			Save(ctx); err != nil {
			return mapPartnerConstraint(err)
		}
	}
	return nil
}

func savePartnerRoleSettlementRules(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, roles []*biz.PartnerRole) error {
	for _, role := range roles {
		if role.SettlementRule == nil {
			continue
		}
		partnerRole, err := tx.PartnerRole.Query().Where(
			partnerroleent.PartnerIDEQ(partnerID),
			partnerroleent.RoleTypeEQ(partnerroleent.RoleType(role.Type)),
		).Only(ctx)
		if err != nil {
			return err
		}
		if err := validateSettlementCurrencies(ctx, tx.Currency.Query(), role.SettlementRule); err != nil {
			return err
		}
		existingRules, err := tx.PartnerSettlementRule.Query().Where(
			partnersettlementruleent.PartnerRoleIDEQ(partnerRole.ID),
		).ForUpdate().All(ctx)
		if err != nil {
			return err
		}
		if len(existingRules) > 1 {
			return biz.ErrPartnerSettlementRuleInvalidArgument
		}
		if len(existingRules) == 0 {
			if _, err := createPartnerSettlementRule(ctx, tx.PartnerSettlementRule.Create().SetPartnerRoleID(partnerRole.ID), role.SettlementRule); err != nil {
				return mapPartnerSettlementRuleConstraint(err)
			}
			continue
		}
		if _, err := updatePartnerSettlementRule(ctx, existingRules[0].Update(), role.SettlementRule); err != nil {
			return mapPartnerSettlementRuleConstraint(err)
		}
	}
	return nil
}

func mapPartnerConstraint(err error) error {
	if !ent.IsConstraintError(err) {
		return err
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "partner_org_code_key"):
		return biz.ErrPartnerCodeExists
	case strings.Contains(message, "partner_org_name_key"):
		return biz.ErrPartnerNameExists
	case strings.Contains(message, "partner_org_uscc_key"):
		return biz.ErrPartnerUSCCExists
	case strings.Contains(message, "partner_primary_contact_key"):
		return biz.ErrPartnerPrimaryContactConflict
	case strings.Contains(message, "partner_alias_name_key"):
		return biz.ErrPartnerAliasExists
	case strings.Contains(message, "partner_role_type_key"):
		return biz.ErrPartnerInvalidRole
	case strings.Contains(message, "partnerassignment_partner_id_role_sort_order"):
		return biz.ErrPartnerInvalidArgument
	default:
		return err
	}
}

func partnerToBiz(item *ent.Partner) *biz.Partner {
	result := &biz.Partner{
		ID:                item.ID,
		OrganizationID:    item.OrganizationID,
		Code:              item.Code,
		LegalName:         item.LegalName,
		NormalizedName:    item.NormalizedName,
		RegisteredAddress: item.RegisteredAddress,
		Enabled:           item.Enabled,
		Roles:             partnerRolesToBiz(item.Edges.Roles),
		CreatedAt:         item.CreatedAt,
		UpdatedAt:         item.UpdatedAt,
	}
	if item.UnifiedSocialCreditCode != nil {
		result.UnifiedSocialCreditCode = *item.UnifiedSocialCreditCode
	}
	result.Contacts = make([]*biz.PartnerContact, 0, len(item.Edges.Contacts))
	for _, contact := range item.Edges.Contacts {
		result.Contacts = append(result.Contacts, &biz.PartnerContact{
			ID: contact.ID, Name: contact.Name, Phone: contact.Phone, Email: contact.Email, Note: contact.Note,
			IsPrimary: contact.IsPrimary, CreatedAt: contact.CreatedAt, UpdatedAt: contact.UpdatedAt,
		})
	}
	result.Aliases = make([]*biz.PartnerAlias, 0, len(item.Edges.Aliases))
	for _, alias := range item.Edges.Aliases {
		result.Aliases = append(result.Aliases, &biz.PartnerAlias{
			ID: alias.ID, AliasName: alias.AliasName, NormalizedAliasName: alias.NormalizedAliasName,
			SortOrder: alias.SortOrder, CreatedAt: alias.CreatedAt, UpdatedAt: alias.UpdatedAt,
		})
	}
	if profile := item.Edges.Profile; profile != nil {
		result.Profile = &biz.PartnerProfile{
			NameEN: profile.NameEn, AddressEN: profile.AddressEn, CountryCode: profile.CountryCode,
			ProvinceCode: profile.ProvinceCode, CityCode: profile.CityCode, DistrictCode: profile.DistrictCode,
			AddressDetail: profile.AddressDetail, Nature: profile.Nature, DevelopmentMethod: profile.DevelopmentMethod,
			CustomerTypes: partnerCustomerTypesToBiz(profile.CustomerTypes), BusinessTypes: partnerBusinessTypesToBiz(profile.BusinessTypes),
			Remark: profile.Remark,
		}
	}
	result.Assignments = make([]*biz.PartnerAssignment, 0, len(item.Edges.Assignments))
	for _, assignment := range item.Edges.Assignments {
		result.Assignments = append(result.Assignments, &biz.PartnerAssignment{
			ID: assignment.ID, Role: biz.PartnerAssignmentRole(assignment.Role), UserID: assignment.UserID,
			OrganizationID: assignment.OrganizationID, SortOrder: assignment.SortOrder, CreatedAt: assignment.CreatedAt, UpdatedAt: assignment.UpdatedAt,
		})
	}
	return result
}

func partnerCustomerTypesToBiz(values []string) []biz.PartnerCustomerType {
	result := make([]biz.PartnerCustomerType, 0, len(values))
	for _, value := range values {
		result = append(result, biz.PartnerCustomerType(value))
	}
	return result
}

func partnerBusinessTypesToBiz(values []string) []biz.PartnerBusinessType {
	result := make([]biz.PartnerBusinessType, 0, len(values))
	for _, value := range values {
		result = append(result, biz.PartnerBusinessType(value))
	}
	return result
}

func partnerRolesToBiz(items []*ent.PartnerRole) []*biz.PartnerRole {
	roles := make([]*biz.PartnerRole, 0, len(items))
	for _, role := range items {
		result := &biz.PartnerRole{
			Type: biz.PartnerRoleType(role.RoleType), Enabled: role.Enabled, Blacklisted: role.Blacklisted,
			BlacklistReason: role.BlacklistReason, BlacklistedAt: role.BlacklistedAt, BlacklistedBy: role.BlacklistedBy,
		}
		if len(role.Edges.SettlementRules) == 1 {
			result.SettlementRule = partnerSettlementRuleToBiz(role.Edges.SettlementRules[0])
		}
		roles = append(roles, result)
	}
	return roles
}
