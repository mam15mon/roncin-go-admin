package data

import (
	"context"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partneraliasent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneralias"
	partnercontactent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnercontact"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"

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
			partnerent.UnifiedSocialCreditCodeContainsFold(options.Keyword),
			partnerent.HasAliasesWith(partneraliasent.AliasNameContainsFold(options.Keyword)),
			partnerent.HasContactsWith(partnercontactent.Or(
				partnercontactent.NameContainsFold(options.Keyword),
				partnercontactent.PhoneContainsFold(options.Keyword),
				partnercontactent.EmailContainsFold(options.Keyword),
			)),
		))
	}
	if options.Role != "" {
		query.Where(partnerent.HasRolesWith(partnerroleent.RoleTypeEQ(partnerroleent.RoleType(options.Role))))
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

func (r *partnerRepo) Create(ctx context.Context, organizationID uuid.UUID, input *biz.Partner) (*biz.Partner, error) {
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
	if err := createPartnerChildren(ctx, tx, created.ID, input); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.Get(ctx, organizationID, created.ID)
}

func (r *partnerRepo) Update(ctx context.Context, organizationID, id uuid.UUID, input *biz.Partner) (*biz.PartnerUpdateResult, error) {
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
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	updated, err := r.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	return &biz.PartnerUpdateResult{Partner: updated, PreviousRoles: previousRoles}, nil
}

func (r *partnerRepo) SetSupplierBlacklist(ctx context.Context, organizationID, id uuid.UUID, input biz.PartnerBlacklistUpdate) (*biz.PartnerBlacklistResult, error) {
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
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	updated, err := r.Get(ctx, organizationID, id)
	if err != nil {
		return nil, err
	}
	return &biz.PartnerBlacklistResult{Partner: updated, PreviouslyBlacklisted: previouslyBlacklisted}, nil
}

func (r *partnerRepo) Import(ctx context.Context, organizationID uuid.UUID, mode biz.PartnerImportMode, inputs []*biz.Partner) (*biz.PartnerImportResult, error) {
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
			if childErr := createPartnerChildren(ctx, tx, created.ID, input); childErr != nil {
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
		if updateErr := updatePartnerInTx(ctx, tx, existing, input); updateErr != nil {
			_ = tx.Rollback()
			return nil, updateErr
		}
		result.UpdatedCount++
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func updatePartnerInTx(ctx context.Context, tx *ent.Tx, existing *ent.Partner, input *biz.Partner) error {
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
	if _, err := tx.PartnerContact.Delete().Where(partnercontactent.PartnerIDEQ(existing.ID)).Exec(ctx); err != nil {
		return err
	}
	if _, err := tx.PartnerAlias.Delete().Where(partneraliasent.PartnerIDEQ(existing.ID)).Exec(ctx); err != nil {
		return err
	}
	if err := createPartnerContacts(ctx, tx, existing.ID, input.Contacts); err != nil {
		return err
	}
	return createPartnerAliases(ctx, tx, existing.ID, input.Aliases)
}

func withPartnerEdges(query *ent.PartnerQuery) *ent.PartnerQuery {
	return query.
		WithRoles(func(roleQuery *ent.PartnerRoleQuery) {
			roleQuery.Order(partnerroleent.ByRoleType())
		}).
		WithContacts(func(contactQuery *ent.PartnerContactQuery) {
			contactQuery.Order(partnercontactent.ByIsPrimary(entsql.OrderDesc()), partnercontactent.ByName())
		}).
		WithAliases(func(aliasQuery *ent.PartnerAliasQuery) {
			aliasQuery.Order(partneraliasent.BySortOrder(), partneraliasent.ByAliasName())
		})
}

func createPartnerChildren(ctx context.Context, tx *ent.Tx, partnerID uuid.UUID, input *biz.Partner) error {
	for _, role := range input.Roles {
		if _, err := tx.PartnerRole.Create().
			SetPartnerID(partnerID).
			SetRoleType(partnerroleent.RoleType(role.Type)).
			SetEnabled(role.Enabled).
			Save(ctx); err != nil {
			return mapPartnerConstraint(err)
		}
	}
	if err := createPartnerContacts(ctx, tx, partnerID, input.Contacts); err != nil {
		return err
	}
	return createPartnerAliases(ctx, tx, partnerID, input.Aliases)
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
			if _, err := tx.PartnerRole.Delete().Where(partnerroleent.IDEQ(role.ID)).Exec(ctx); err != nil {
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
	return result
}

func partnerRolesToBiz(items []*ent.PartnerRole) []*biz.PartnerRole {
	roles := make([]*biz.PartnerRole, 0, len(items))
	for _, role := range items {
		roles = append(roles, &biz.PartnerRole{
			Type: biz.PartnerRoleType(role.RoleType), Enabled: role.Enabled, Blacklisted: role.Blacklisted,
			BlacklistReason: role.BlacklistReason, BlacklistedAt: role.BlacklistedAt, BlacklistedBy: role.BlacklistedBy,
		})
	}
	return roles
}
