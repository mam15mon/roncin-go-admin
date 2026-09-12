package data

import (
	"context"
	"sort"
	"strings"
	"time"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	invitationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkinvitation"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/session"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// dingTalkRegistrationRepo 实现钉钉邀请与注册审批的全部仓储契约
// （biz.DingTalkRegistrationRepo 管理侧 + biz.DingTalkLoginRegistrationRepo 登录侧）。
type dingTalkRegistrationRepo struct{ data *Data }

func NewDingTalkRegistrationRepo(data *Data) *dingTalkRegistrationRepo {
	return &dingTalkRegistrationRepo{data: data}
}

// ===== 邀请管理 =====

func (r *dingTalkRegistrationRepo) CreateInvitation(ctx context.Context, input *biz.DingTalkInvitation, audit *biz.AuditEvent) (*biz.DingTalkInvitation, error) {
	var createdID uuid.UUID
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if _, queryErr := tx.Organization.Query().Where(organization.IDEQ(input.OrganizationID), organization.EnabledEQ(true)).Only(ctx); queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminOrganizationNotFound, nil)
		}
		if input.RoleID != nil {
			// 初始角色必须属于目标组织且启用。
			if _, queryErr := rolesForOrganization(ctx, tx.Role.Query(), input.OrganizationID, []uuid.UUID{*input.RoleID}); queryErr != nil {
				return queryErr
			}
		}
		if input.Kind == biz.DingTalkInvitationKindTargeted && input.Mobile != nil {
			// 惰性收割同手机号的过期 PENDING 行：过期邀请按无邀请处理，
			// 不应继续占用「活跃邀请唯一」索引位阻断重建。
			if _, updateErr := tx.DingTalkInvitation.Update().
				Where(
					invitationent.KindEQ(invitationent.KindTARGETED),
					invitationent.OrganizationIDEQ(input.OrganizationID),
					invitationent.MobileEQ(*input.Mobile),
					invitationent.StatusEQ(invitationent.StatusPENDING),
					invitationent.ExpiresAtLTE(time.Now().UTC()),
				).
				SetStatus(invitationent.Status(biz.DingTalkInvitationStatusExpired)).
				Save(ctx); updateErr != nil {
				return updateErr
			}
		}
		create := tx.DingTalkInvitation.Create().
			SetToken(input.Token).
			SetKind(invitationent.Kind(input.Kind)).
			SetOrganizationID(input.OrganizationID).
			SetDisplayName(input.DisplayName).
			SetInvitedBy(input.InvitedBy).
			SetStatus(invitationent.Status(input.Status)).
			SetExpiresAt(input.ExpiresAt)
		if input.RoleID != nil {
			create.SetRoleID(*input.RoleID)
		}
		if input.Mobile != nil {
			create.SetMobile(*input.Mobile)
		}
		created, createErr := create.Save(ctx)
		if createErr != nil {
			// 活跃邀请唯一索引兜底同手机号重复创建。
			return mapEntConstraint(createErr, "dingtalkinvitation_organization_id_mobile", biz.ErrDingTalkInvitationExists)
		}
		createdID = created.ID
		audit.Details["resource_id"] = created.ID.String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findInvitation(ctx, createdID)
}

func (r *dingTalkRegistrationRepo) findInvitation(ctx context.Context, id uuid.UUID) (*biz.DingTalkInvitation, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.DingTalkInvitation.Query().
		Where(invitationent.IDEQ(id)).
		WithOrganization().WithRole().WithInviter().WithConsumer().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkInvitationNotFound, nil)
	}
	return invitationToBiz(item, time.Now().UTC()), nil
}

func (r *dingTalkRegistrationRepo) GetInvitation(ctx context.Context, id uuid.UUID, organizationIDs []uuid.UUID) (*biz.DingTalkInvitation, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.DingTalkInvitation.Query().
		Where(invitationent.IDEQ(id)).
		WithOrganization().WithRole().WithInviter().WithConsumer().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkInvitationNotFound, nil)
	}
	if !uuidInValues(organizationIDs, item.OrganizationID) {
		return nil, biz.ErrPermissionDenied
	}
	return invitationToBiz(item, time.Now().UTC()), nil
}

func (r *dingTalkRegistrationRepo) ListInvitations(ctx context.Context, organizationIDs []uuid.UUID, options biz.DingTalkInvitationListOptions) (*biz.DingTalkInvitationList, error) {
	now := time.Now().UTC()
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.DingTalkInvitation.Query().
		Where(invitationent.OrganizationIDIn(organizationIDs...)).
		WithOrganization().WithRole().WithInviter().WithConsumer()
	if options.OrganizationID != uuid.Nil {
		query = query.Where(invitationent.OrganizationIDEQ(options.OrganizationID))
	}
	if options.Status != nil {
		query = query.Where(invitationStatusFilter(*options.Status, now))
	}
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.DingTalkInvitation, error) {
		return query.Clone().
			Order(invitationent.ByCreatedAt(entsql.OrderDesc()), invitationent.ByID(entsql.OrderDesc())).
			Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, infalliblePageConverter(func(item *ent.DingTalkInvitation) *biz.DingTalkInvitation {
		return invitationToBiz(item, now)
	}))
}

// invitationStatusFilter 把展示状态翻译为 SQL 谓词：EXPIRED 同时覆盖显式过期与
// PENDING 已过有效期（过期按有效期惰性判定，不回写状态行）。
func invitationStatusFilter(status biz.DingTalkInvitationStatus, now time.Time) predicate.DingTalkInvitation {
	switch status {
	case biz.DingTalkInvitationStatusPending:
		return invitationent.And(invitationent.StatusEQ(invitationent.StatusPENDING), invitationent.ExpiresAtGT(now))
	case biz.DingTalkInvitationStatusExpired:
		return invitationent.Or(
			invitationent.StatusEQ(invitationent.StatusEXPIRED),
			invitationent.And(invitationent.StatusEQ(invitationent.StatusPENDING), invitationent.ExpiresAtLTE(now)),
		)
	default:
		return invitationent.StatusEQ(invitationent.Status(status))
	}
}

func (r *dingTalkRegistrationRepo) RevokeInvitation(ctx context.Context, actorID, id uuid.UUID, organizationIDs []uuid.UUID, audit *biz.AuditEvent) (*biz.DingTalkInvitation, error) {
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 悲观锁防止与扫码消费并发：撤销与消费只有一个能改变 PENDING 状态。
		item, queryErr := tx.DingTalkInvitation.Query().Where(invitationent.IDEQ(id)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrDingTalkInvitationNotFound, nil)
		}
		if !uuidInValues(organizationIDs, item.OrganizationID) {
			return biz.ErrPermissionDenied
		}
		if item.Status != invitationent.StatusPENDING {
			return biz.ErrDingTalkInvitationNotRevocable
		}
		if _, updateErr := tx.DingTalkInvitation.UpdateOneID(id).SetStatus(invitationent.Status(biz.DingTalkInvitationStatusRevoked)).Save(ctx); updateErr != nil {
			return updateErr
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	return r.findInvitation(ctx, id)
}

func invitationToBiz(item *ent.DingTalkInvitation, now time.Time) *biz.DingTalkInvitation {
	result := &biz.DingTalkInvitation{
		ID:             item.ID,
		Token:          item.Token,
		Kind:           biz.DingTalkInvitationKind(item.Kind),
		OrganizationID: item.OrganizationID,
		RoleID:         item.RoleID,
		Mobile:         item.Mobile,
		DisplayName:    item.DisplayName,
		InvitedBy:      item.InvitedBy,
		Status:         biz.DingTalkInvitationStatus(item.Status),
		ConsumedAt:     item.ConsumedAt,
		ExpiresAt:      item.ExpiresAt,
		CreatedAt:      item.CreatedAt,
	}
	if org, err := item.Edges.OrganizationOrErr(); err == nil {
		result.OrganizationName = org.Name
	}
	if assignedRole, err := item.Edges.RoleOrErr(); err == nil {
		result.RoleName = assignedRole.Name
	}
	if inviter, err := item.Edges.InviterOrErr(); err == nil {
		result.InviterName = inviter.DisplayName
	}
	if consumer := item.Edges.Consumer; consumer != nil {
		result.ConsumedByName = consumer.DisplayName
	}
	// 展示口径统一按有效期计算：PENDING 且已过期按 EXPIRED 呈现。
	result.Status = result.EffectiveStatus(now)
	return result
}

// FindInvitationByToken 通过 128-bit Token 查找专属邀请（通用/定向）。
func (r *dingTalkRegistrationRepo) FindInvitationByToken(ctx context.Context, token string) (*biz.DingTalkInvitation, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.DingTalkInvitation.Query().
		Where(invitationent.TokenEQ(token)).
		WithOrganization().WithRole().WithInviter().WithConsumer().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkInvitationNotFound, nil)
	}
	return invitationToBiz(item, time.Now().UTC()), nil
}

// ===== 通道 A：扫码匹配与自动激活 =====

// FindActiveInvitationByMobile 返回该手机号最早创建的未过期活跃定向邀请；
// 仅定向邀请（kind = TARGETED）参与手机号匹配，通用码不参与。
// 同一手机号可在多个组织各持一条活跃邀请（唯一索引按组织隔离），按创建
// 时间先建先得（First 语义），命中任一即返回，不再因多行命中报错。
func (r *dingTalkRegistrationRepo) FindActiveInvitationByMobile(ctx context.Context, mobile string) (*biz.DingTalkInvitation, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.DingTalkInvitation.Query().
		Where(
			invitationent.KindEQ(invitationent.KindTARGETED),
			invitationent.MobileEQ(mobile),
			invitationent.StatusEQ(invitationent.StatusPENDING),
			invitationent.ExpiresAtGT(time.Now().UTC()),
		).
		Order(invitationent.ByCreatedAt(), invitationent.ByID()).
		First(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkInvitationNotFound, nil)
	}
	return invitationToBiz(item, time.Now().UTC()), nil
}

// ConsumeInvitationAndActivate 在单事务内完成通道 A：锁定并消费邀请 → 创建启用
// 账号 → 目标组织成员资格（primary，新账号无其他成员资格）→ 初始角色 →
// 通知本人与邀请人 → 审计。仅 TARGETED 邀请可消费。任何一步不可用（并发消费、角色被删、组织停用）都
// 整体回滚并返回错误，由登录链路降级通道 B。
func (r *dingTalkRegistrationRepo) ConsumeInvitationAndActivate(ctx context.Context, identity *biz.DingTalkIdentity, invitationID uuid.UUID) (*biz.Credential, error) {
	now := time.Now().UTC()
	var credential *biz.Credential
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		invitation, queryErr := tx.DingTalkInvitation.Query().Where(invitationent.IDEQ(invitationID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrDingTalkInvitationNotFound, nil)
		}
		if invitation.Kind != invitationent.KindTARGETED || invitation.Status != invitationent.StatusPENDING || !invitation.ExpiresAt.After(now) {
			return biz.ErrDingTalkInvitationNotConsumable
		}
		targetOrganization, queryErr := tx.Organization.Query().Where(organization.IDEQ(invitation.OrganizationID), organization.EnabledEQ(true)).Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrAdminOrganizationNotFound, nil)
		}
		var roles []*ent.Role
		if invitation.RoleID != nil {
			var roleErr error
			roles, roleErr = rolesForOrganization(ctx, tx.Role.Query(), invitation.OrganizationID, []uuid.UUID{*invitation.RoleID})
			if roleErr != nil {
				return roleErr
			}
		}
		// 钉钉扫码返回是身份唯一真相源：姓名/头像/unionId/userId 只取自 identity。
		create := tx.User.Create().
			SetDisplayName(identity.Name).
			SetDingtalkUnionid(identity.UnionID).
			SetDingtalkUserid(identity.UserID).
			SetDingtalkName(identity.Name).
			SetEnabled(true)
		if identity.Email != nil && strings.TrimSpace(*identity.Email) != "" {
			create.SetEmail(strings.TrimSpace(*identity.Email))
		}
		if identity.AvatarURL != nil && strings.TrimSpace(*identity.AvatarURL) != "" {
			create.SetAvatarURL(strings.TrimSpace(*identity.AvatarURL))
		}
		account, createErr := create.Save(ctx)
		if createErr != nil {
			// 并发窗口内账号已被注册：降级通道 B，下次扫码命中既有账号。
			return mapEntConstraint(createErr, "user_dingtalk_unionid", biz.ErrDingTalkInvitationNotConsumable)
		}
		targetMembership, createErr := tx.Membership.Create().
			SetUserID(account.ID).
			SetOrganizationID(targetOrganization.ID).
			SetPrimary(true).
			SetEnabled(true).
			Save(ctx)
		if createErr != nil {
			return createErr
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, targetMembership.ID, roles); replaceErr != nil {
			return replaceErr
		}
		if _, updateErr := tx.DingTalkInvitation.UpdateOneID(invitation.ID).
			SetStatus(invitationent.Status(biz.DingTalkInvitationStatusConsumed)).
			SetConsumedBy(account.ID).
			SetConsumedAt(now).
			Save(ctx); updateErr != nil {
			return updateErr
		}
		if err := enqueueDingTalkUserAuthorizedNotification(ctx, tx, targetOrganization.ID, account, biz.NewDingTalkUserAuthorizedNotification(account.ID)); err != nil {
			return err
		}
		inviter, queryErr := tx.User.Query().Where(user.IDEQ(invitation.InvitedBy)).Only(ctx)
		if queryErr != nil {
			return queryErr
		}
		if err := enqueueDingTalkInvitationActivatedNotification(ctx, tx, targetOrganization.ID, inviter, identity.Name, targetOrganization.Name, biz.NewDingTalkInvitationActivatedNotification(inviter.ID)); err != nil {
			return err
		}
		maskedMobile := ""
		if invitation.Mobile != nil {
			maskedMobile = biz.MaskDingTalkMobile(*invitation.Mobile)
		}
		if err := writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{
			OrganizationID: &targetOrganization.ID,
			UserID:         &account.ID,
			Action:         "auth.dingtalk.invitation.consume",
			Result:         "success",
			Details: map[string]string{
				// 手机号只记脱敏形式。
				"mobile":                 maskedMobile,
				"invitation.id":          invitation.ID.String(),
				"invited_by":             invitation.InvitedBy.String(),
				"target_organization.id": targetOrganization.ID.String(),
			},
		}); err != nil {
			return err
		}
		credential = credentialFromAccount(account, targetOrganization.ID)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return credential, nil
}

// ===== 通道 B：注册目标组织与审批路由 =====

// ListRegistrationOrganizations 返回注册确认页可选的启用中公司组织
// （排序规则与登录候选一致：名称 + ID 确定性排序）。
func (r *dingTalkRegistrationRepo) ListRegistrationOrganizations(ctx context.Context) ([]biz.OrganizationChoice, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.Organization.Query().
		Where(organization.KindEQ(organization.KindCompany), organization.EnabledEQ(true)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	choices := make([]biz.OrganizationChoice, 0, len(items))
	for _, item := range items {
		choices = append(choices, biz.OrganizationChoice{OrganizationID: item.ID, OrganizationName: item.Name, OrganizationCode: item.Code})
	}
	sort.Slice(choices, func(i, j int) bool {
		if choices[i].OrganizationName != choices[j].OrganizationName {
			return choices[i].OrganizationName < choices[j].OrganizationName
		}
		return choices[i].OrganizationID.String() < choices[j].OrganizationID.String()
	})
	return choices, nil
}

func (r *dingTalkRegistrationRepo) FindRegistrationOrganization(ctx context.Context, organizationID uuid.UUID) (*biz.Organization, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.Organization.Query().
		Where(organization.IDEQ(organizationID), organization.EnabledEQ(true), organization.KindEQ(organization.KindCompany)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkRegistrationOrgInvalid, nil)
	}
	organizationView := &biz.Organization{ID: item.ID, Code: item.Code, Name: item.Name}
	if item.BaseCurrency != nil {
		organizationView.BaseCurrency = *item.BaseCurrency
	}
	return organizationView, nil
}

func (r *dingTalkRegistrationRepo) FindHeadquartersOrganizationID(ctx context.Context) (uuid.UUID, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	return findHeadquartersOrganizationID(ctx, client)
}

func findHeadquartersOrganizationID(ctx context.Context, client *ent.Client) (uuid.UUID, error) {
	item, err := client.Organization.Query().
		Where(organization.KindEQ(organization.KindHeadquarters), organization.ParentIDIsNil(), organization.EnabledEQ(true)).
		Only(ctx)
	if err != nil {
		return uuid.Nil, mapEntError(err, biz.ErrAdminOrganizationNotFound, nil)
	}
	return item.ID, nil
}

// ListApproverRecipients 返回目标组织内持有「钉钉邀请与注册审批」权限的启用
// 中用户。路由口径：目标组织内启用成员资格 × 启用角色真实持有权限——总部
// 管理员即使持 ALL 范围也不会被分公司注册提醒打扰（仍可在审批队列处理）。
// 收件人过多时按最近会话活跃时间取前 10 人（无会话者排最后，ID 稳定排序），
// 防止大组织一次性给几十位管理员群发提醒。
func (r *dingTalkRegistrationRepo) ListApproverRecipients(ctx context.Context, organizationID uuid.UUID) ([]*biz.DingTalkApproverRecipient, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return listApproverRecipients(ctx, client, organizationID, inTransaction(ctx))
}

// inTransaction 报告当前上下文是否携带共享事务，用于决定事务内读取是否加
// FOR SHARE（非事务读取不加锁，保持原有语义）。
func inTransaction(ctx context.Context) bool {
	_, ok := transactionFromContext(ctx)
	return ok
}

func listApproverRecipients(ctx context.Context, client *ent.Client, organizationID uuid.UUID, lockReads bool) ([]*biz.DingTalkApproverRecipient, error) {
	const maxRecipients = 10
	candidateQuery := client.User.Query().
		Where(
			user.EnabledEQ(true),
			user.IsBootstrapAdminEQ(false),
			user.DingtalkUseridNotNil(),
			user.HasMembershipsWith(
				membership.EnabledEQ(true),
				membership.OrganizationIDEQ(organizationID),
				membership.HasRoleAssignmentsWith(
					roleassignment.HasRoleWith(
						role.EnabledEQ(true),
						role.HasPermissionsWith(permission.KeyEQ(access.UserDingTalkInvitationManage)),
					),
				),
			),
		)
	if lockReads {
		candidateQuery.ForShare()
	}
	candidates, err := candidateQuery.All(ctx)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}
	candidateIDs := make([]uuid.UUID, 0, len(candidates))
	for _, candidate := range candidates {
		candidateIDs = append(candidateIDs, candidate.ID)
	}
	// 候选集为组织管理员量级，一次查询取全部会话后在内存内确定最近活跃排序。
	latestSessions, err := client.Session.Query().
		Where(session.UserIDIn(candidateIDs...)).
		Order(session.ByCreatedAt(entsql.OrderDesc()), session.ByID(entsql.OrderDesc())).
		All(ctx)
	if err != nil {
		return nil, err
	}
	lastActiveByUser := make(map[uuid.UUID]time.Time, len(candidates))
	for _, item := range latestSessions {
		if _, seen := lastActiveByUser[item.UserID]; !seen {
			lastActiveByUser[item.UserID] = item.CreatedAt
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, leftSeen := lastActiveByUser[candidates[i].ID]
		right, rightSeen := lastActiveByUser[candidates[j].ID]
		if leftSeen != rightSeen {
			return leftSeen
		}
		if leftSeen && !left.Equal(right) {
			return left.After(right)
		}
		return candidates[i].ID.String() < candidates[j].ID.String()
	})
	if len(candidates) > maxRecipients {
		candidates = candidates[:maxRecipients]
	}
	recipients := make([]*biz.DingTalkApproverRecipient, 0, len(candidates))
	for _, candidate := range candidates {
		recipients = append(recipients, &biz.DingTalkApproverRecipient{UserID: candidate.ID, DisplayName: candidate.DisplayName})
	}
	return recipients, nil
}

// ===== 注册审批队列与一站式审批 =====

// membershipHasNoRoleAssignments 判定成员资格上没有任何角色分配（待审批注册的
// 核心特征，与用户管理 PENDING_AUTHORIZATION 状态口径一致）。
func membershipHasNoRoleAssignments() predicate.Membership {
	return predicate.Membership(func(s *entsql.Selector) {
		assignments := entsql.Table(roleassignment.Table)
		s.Where(entsql.NotExists(
			entsql.Select(assignments.C(roleassignment.FieldMembershipID)).
				From(assignments).
				Where(entsql.ColumnsEQ(s.C(membership.FieldID), assignments.C(roleassignment.FieldMembershipID))),
		))
	})
}

// pendingRegistrationBasePredicates 组装「待审批钉钉注册」的基础用户谓词：
// 账号禁用 + 已绑钉钉身份 + 存在启用成员资格且其上没有角色分配。
func pendingRegistrationBasePredicates() []predicate.User {
	return []predicate.User{
		user.EnabledEQ(false),
		user.DingtalkUnionidNotNil(),
		user.HasMembershipsWith(
			membership.EnabledEQ(true),
			membership.HasOrganizationWith(organization.EnabledEQ(true)),
			membershipHasNoRoleAssignments(),
		),
	}
}

// pendingRegistrationRoutingPredicate 限定注册的路由组织在调用者范围内：
// 自选目标组织命中，或未自选时收口成员资格组织命中（注册收口为总部根，即总部兜底）。
func pendingRegistrationRoutingPredicate(organizationIDs []uuid.UUID) predicate.User {
	return user.Or(
		user.DingtalkRequestedOrganizationIDIn(organizationIDs...),
		user.And(
			user.DingtalkRequestedOrganizationIDIsNil(),
			user.HasMembershipsWith(membership.EnabledEQ(true), membership.OrganizationIDIn(organizationIDs...)),
		),
	)
}

func registrationFromUser(account *ent.User) (*biz.DingTalkRegistration, error) {
	result := &biz.DingTalkRegistration{
		UserID:       account.ID,
		DisplayName:  account.DisplayName,
		AvatarURL:    account.AvatarURL,
		RegisteredAt: account.CreatedAt,
	}
	if account.DingtalkRequestedOrganizationID != nil {
		requested := *account.DingtalkRequestedOrganizationID
		result.RequestedOrganizationID = &requested
		if org, err := account.Edges.DingtalkRequestedOrganizationOrErr(); err == nil {
			result.RequestedOrganizationName = org.Name
		}
	}
	for _, member := range account.Edges.Memberships {
		if !member.Enabled {
			continue
		}
		if org, err := member.Edges.OrganizationOrErr(); err == nil {
			result.IntakeOrganizationID = org.ID
			if result.RequestedOrganizationID == nil {
				// 未自选目标组织：按总部兜底展示收口组织。
				result.RequestedOrganizationName = org.Name
			}
			break
		}
	}
	return result, nil
}

func (r *dingTalkRegistrationRepo) GetPendingRegistration(ctx context.Context, userID uuid.UUID) (*biz.DingTalkRegistration, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	account, err := client.User.Query().
		Where(append(pendingRegistrationBasePredicates(), user.IDEQ(userID))...).
		WithMemberships(func(query *ent.MembershipQuery) {
			query.Where(membership.EnabledEQ(true)).WithOrganization()
		}).
		WithDingtalkRequestedOrganization().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkRegistrationNotFound, nil)
	}
	return registrationFromUser(account)
}

func (r *dingTalkRegistrationRepo) ListPendingRegistrations(ctx context.Context, organizationIDs []uuid.UUID, options biz.DingTalkRegistrationListOptions) (*biz.DingTalkRegistrationList, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	query := client.User.Query().
		Where(append(pendingRegistrationBasePredicates(), pendingRegistrationRoutingPredicate(organizationIDs))...).
		WithMemberships(func(query *ent.MembershipQuery) {
			query.Where(membership.EnabledEQ(true)).WithOrganization()
		}).
		WithDingtalkRequestedOrganization()
	return paginate(ctx, func(ctx context.Context) (int, error) {
		return query.Clone().Count(ctx)
	}, func(ctx context.Context, offset, limit int) ([]*ent.User, error) {
		return query.Clone().
			Order(user.ByCreatedAt(entsql.OrderDesc()), user.ByID(entsql.OrderDesc())).
			Offset(offset).Limit(limit).All(ctx)
	}, options.Page, options.PageSize, func(account *ent.User) (*biz.DingTalkRegistration, error) {
		return registrationFromUser(account)
	})
}

// lockPendingRegistration 在事务内锁定并复核待审批注册；账号已启用说明注册已被
// 处理（幂等冲突），其余不符按不存在处理。
func lockPendingRegistration(ctx context.Context, tx *ent.Tx, userID uuid.UUID) (*ent.User, error) {
	account, err := tx.User.Query().Where(user.IDEQ(userID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrDingTalkRegistrationNotFound, nil)
	}
	if account.DingtalkUnionid == nil {
		return nil, biz.ErrDingTalkRegistrationNotFound
	}
	if account.Enabled {
		return nil, biz.ErrDingTalkRegistrationProcessed
	}
	return account, nil
}

// resolveRegistrationRouting 在事务内解析注册的路由组织（自选目标 ?? 收口成员
// 资格组织），并复核其属于审批人的可写组织范围。
func resolveRegistrationRouting(ctx context.Context, tx *ent.Tx, account *ent.User, allowedOrganizationIDs []uuid.UUID) (*ent.Organization, error) {
	routingID := uuid.Nil
	if account.DingtalkRequestedOrganizationID != nil {
		routingID = *account.DingtalkRequestedOrganizationID
	}
	if routingID == uuid.Nil {
		intake, queryErr := tx.Membership.Query().
			Where(membership.UserIDEQ(account.ID), membership.EnabledEQ(true)).
			Only(ctx)
		if queryErr != nil {
			return nil, biz.ErrDingTalkRegistrationNotFound
		}
		routingID = intake.OrganizationID
	}
	if !uuidInValues(allowedOrganizationIDs, routingID) {
		return nil, biz.ErrPermissionDenied
	}
	routingOrganization, queryErr := tx.Organization.Query().
		Where(organization.IDEQ(routingID), organization.EnabledEQ(true)).
		Only(ctx)
	if queryErr != nil {
		return nil, mapEntError(queryErr, biz.ErrAdminOrganizationNotFound, nil)
	}
	return routingOrganization, nil
}

// ApproveRegistration 一站式同意：启用账号 → 目标组织成员资格（停用收口与
// 其他成员资格，目标置 primary）→ 授予初始角色 → 通知本人 → 审计。
func (r *dingTalkRegistrationRepo) ApproveRegistration(ctx context.Context, decision *biz.DingTalkRegistrationDecision) (*biz.DingTalkRegistration, error) {
	var approved *biz.DingTalkRegistration
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, queryErr := lockPendingRegistration(ctx, tx, decision.UserID)
		if queryErr != nil {
			return queryErr
		}
		routingOrganization, queryErr := resolveRegistrationRouting(ctx, tx, account, decision.OrganizationIDs)
		if queryErr != nil {
			return queryErr
		}
		roles, queryErr := rolesForOrganization(ctx, tx.Role.Query(), routingOrganization.ID, decision.RoleIDs)
		if queryErr != nil {
			return queryErr
		}
		if _, updateErr := tx.User.UpdateOneID(account.ID).SetEnabled(true).Save(ctx); updateErr != nil {
			return updateErr
		}
		membershipIDs, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(account.ID)).IDs(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(membershipIDs) > 0 {
			if _, deleteErr := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDIn(membershipIDs...)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
		}
		if _, updateErr := tx.Membership.Update().Where(membership.UserIDEQ(account.ID)).SetEnabled(false).SetPrimary(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		targetMembership, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(account.ID), membership.OrganizationIDEQ(routingOrganization.ID)).Only(ctx)
		if ent.IsNotFound(queryErr) {
			targetMembership, queryErr = tx.Membership.Create().SetUserID(account.ID).SetOrganizationID(routingOrganization.ID).SetEnabled(true).SetPrimary(true).Save(ctx)
		} else if queryErr == nil {
			targetMembership, queryErr = tx.Membership.UpdateOneID(targetMembership.ID).SetEnabled(true).SetPrimary(true).Save(ctx)
		}
		if queryErr != nil {
			return queryErr
		}
		if replaceErr := replaceRoleAssignments(ctx, tx, targetMembership.ID, roles); replaceErr != nil {
			return replaceErr
		}
		if decision.Notification != nil {
			if notificationErr := enqueueDingTalkUserAuthorizedNotification(ctx, tx, routingOrganization.ID, account, decision.Notification); notificationErr != nil {
				return notificationErr
			}
		}
		// 账号从禁用回到启用：吊销历史会话，防止旧的登录态复活。
		if _, updateErr := tx.Session.Update().Where(session.UserIDEQ(account.ID), session.RevokedAtIsNil()).SetRevokedAt(time.Now().UTC()).Save(ctx); updateErr != nil {
			return updateErr
		}
		if err := writeAudit(ctx, tx.AuditLog, decision.Audit); err != nil {
			return err
		}
		approved = &biz.DingTalkRegistration{
			UserID:                    account.ID,
			DisplayName:               account.DisplayName,
			AvatarURL:                 account.AvatarURL,
			RegisteredAt:              account.CreatedAt,
			RequestedOrganizationID:   &routingOrganization.ID,
			RequestedOrganizationName: routingOrganization.Name,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return approved, nil
}

// RejectRegistration 拒绝注册：停用收口成员资格（注册离开待审批队列，账号保留
// 钉钉身份与历史留痕）→ 通知本人拒绝原因 → 审计。
func (r *dingTalkRegistrationRepo) RejectRegistration(ctx context.Context, decision *biz.DingTalkRegistrationDecision) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, queryErr := lockPendingRegistration(ctx, tx, decision.UserID)
		if queryErr != nil {
			return queryErr
		}
		routingOrganization, queryErr := resolveRegistrationRouting(ctx, tx, account, decision.OrganizationIDs)
		if queryErr != nil {
			return queryErr
		}
		membershipIDs, queryErr := tx.Membership.Query().Where(membership.UserIDEQ(account.ID), membership.EnabledEQ(true)).IDs(ctx)
		if queryErr != nil {
			return queryErr
		}
		if len(membershipIDs) > 0 {
			if _, deleteErr := tx.RoleAssignment.Delete().Where(roleassignment.MembershipIDIn(membershipIDs...)).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
		}
		if _, updateErr := tx.Membership.Update().Where(membership.UserIDEQ(account.ID)).SetEnabled(false).SetPrimary(false).Save(ctx); updateErr != nil {
			return updateErr
		}
		if decision.Notification != nil && account.DingtalkUserid != nil {
			if notificationErr := enqueueDingTalkRegistrationRejectedNotification(ctx, tx, routingOrganization.ID, account, decision.Reason, decision.Notification); notificationErr != nil {
				return notificationErr
			}
		}
		return writeAudit(ctx, tx.AuditLog, decision.Audit)
	})
}

// GetActorRolesPrivilegeProfiles / GetRolesPrivilegeProfiles 与用户管理提权校验
// 共用同一口径（实现与 adminRepo 相同的查询语义）。
func (r *dingTalkRegistrationRepo) GetActorRolesPrivilegeProfiles(ctx context.Context, organizationID, actorID uuid.UUID) ([]*biz.AdminRoleProfile, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return actorRolesPrivilegeProfiles(ctx, client, organizationID, actorID)
}

func (r *dingTalkRegistrationRepo) GetRolesPrivilegeProfiles(ctx context.Context, organizationID uuid.UUID, roleIDs []uuid.UUID) ([]*biz.AdminRoleProfile, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return rolesPrivilegeProfiles(ctx, client, organizationID, roleIDs)
}

func (r *dingTalkRegistrationRepo) GetParentOrganizationID(ctx context.Context, orgID uuid.UUID) (*uuid.UUID, bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, false, err
	}
	return getParentOrganizationID(ctx, client, orgID, inTransaction(ctx))
}

func getParentOrganizationID(ctx context.Context, client *ent.Client, orgID uuid.UUID, lockReads bool) (*uuid.UUID, bool, error) {
	query := client.Organization.Query().Where(organization.IDEQ(orgID))
	if lockReads {
		query.ForShare()
	}
	item, err := query.Only(ctx)
	if err != nil {
		return nil, false, mapEntError(err, biz.ErrAdminOrganizationNotFound, nil)
	}
	if item.ParentID != nil && *item.ParentID != uuid.Nil {
		return item.ParentID, true, nil
	}
	return nil, false, nil
}

// listApproverRecipientsWithEscalation 沿 parent_id 逐级向上追溯首个有候选审批人的祖先节点（直至总部根节点）。
// 属于通知兜底而非权限变更。仅供转派事务回调内部使用（显式传事务客户端并强制 FOR SHARE），
// 普通上下文的追溯决策在 biz 层组合仓储原语完成（见 biz.ListApproverRecipientsWithEscalation）。
func listApproverRecipientsWithEscalation(ctx context.Context, client *ent.Client, targetOrgID uuid.UUID, lockReads bool) ([]*biz.DingTalkApproverRecipient, uuid.UUID, bool, error) {
	currID := targetOrgID
	isEscalated := false
	visited := make(map[uuid.UUID]bool)
	for depth := 0; depth < 20; depth++ {
		if visited[currID] {
			break
		}
		visited[currID] = true

		recipients, err := listApproverRecipients(ctx, client, currID, lockReads)
		if err != nil {
			return nil, uuid.Nil, false, err
		}
		if len(recipients) > 0 {
			return recipients, currID, isEscalated, nil
		}
		parentID, hasParent, err := getParentOrganizationID(ctx, client, currID, lockReads)
		if err != nil {
			return nil, uuid.Nil, false, err
		}
		if !hasParent || parentID == nil {
			// 已达根组织，若当前不是总部且总部存在，尝试兜底总部
			hqID, hqErr := findHeadquartersOrganizationID(ctx, client)
			if hqErr == nil && hqID != currID {
				hqRecipients, hqQueryErr := listApproverRecipients(ctx, client, hqID, lockReads)
				if hqQueryErr != nil {
					return nil, uuid.Nil, false, hqQueryErr
				}
				if len(hqRecipients) > 0 {
					return hqRecipients, hqID, true, nil
				}
			}
			return nil, currID, isEscalated, nil
		}
		currID = *parentID
		isEscalated = true
	}
	return nil, currID, isEscalated, nil
}

// TransferRegistration 将待审批注册一键转派至目标分公司：
// 在行级悲观锁事务中原子更新 users.dingtalk_requested_organization_id，并向新组织审批人重新入队通知。
func (r *dingTalkRegistrationRepo) TransferRegistration(ctx context.Context, decision *biz.DingTalkRegistrationDecision, targetOrgID uuid.UUID) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		account, queryErr := lockPendingRegistration(ctx, tx, decision.UserID)
		if queryErr != nil {
			return queryErr
		}
		currentRoutingOrg, queryErr := resolveRegistrationRouting(ctx, tx, account, decision.OrganizationIDs)
		if queryErr != nil {
			return queryErr
		}
		if currentRoutingOrg.ID == targetOrgID {
			return biz.ErrDingTalkRegistrationTransferSame
		}
		targetOrg, queryErr := tx.Organization.Query().Where(organization.IDEQ(targetOrgID), organization.EnabledEQ(true)).Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrDingTalkRegistrationOrgInvalid, nil)
		}
		if _, updateErr := tx.User.UpdateOneID(account.ID).SetDingtalkRequestedOrganizationID(targetOrgID).Save(ctx); updateErr != nil {
			return updateErr
		}
		// 向新组织审批人（支持向上追溯）重新入队审批通知。
		// 追溯读取经由事务客户端（FOR SHARE）执行，确保落在转派事务内。
		recipients, _, isEscalated, recipientErr := listApproverRecipientsWithEscalation(ctx, tx.Client(), targetOrgID, true)
		if recipientErr == nil && len(recipients) > 0 {
			recipientUserIDs := make([]uuid.UUID, 0, len(recipients))
			for _, rec := range recipients {
				recipientUserIDs = append(recipientUserIDs, rec.UserID)
			}
			// 代管标注口径与扫码注册路径一致：展示目标组织名并追加统一后缀。
			targetOrgName := targetOrg.Name
			if isEscalated {
				targetOrgName += biz.DingTalkEscalatedOrgSuffix
			}
			if err := enqueueDingTalkRegistrationPendingNotifications(ctx, tx, targetOrg.ID, account.ID, account.DisplayName, targetOrgName, recipientUserIDs); err != nil {
				return err
			}
		}
		return writeAudit(ctx, tx.AuditLog, decision.Audit)
	})
}

func uuidInValues(values []uuid.UUID, target uuid.UUID) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var _ biz.DingTalkRegistrationRepo = (*dingTalkRegistrationRepo)(nil)
var _ biz.DingTalkLoginRegistrationRepo = (*dingTalkRegistrationRepo)(nil)
