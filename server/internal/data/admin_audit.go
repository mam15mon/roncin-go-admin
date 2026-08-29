package data

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

func (r *adminRepo) ListAuditLogs(ctx context.Context, organizationID uuid.UUID, options biz.AdminAuditLogListOptions) (*biz.AdminAuditLogList, error) {
	query := r.data.db.AuditLog.Query().Where(auditlog.OrganizationIDEQ(organizationID))
	if options.Action != "" {
		query.Where(auditlog.ActionContains(options.Action))
	}
	if options.UserID != nil {
		query.Where(auditlog.UserIDEQ(*options.UserID))
	}
	if options.ResourceType != "" {
		query.Where(auditlog.ResourceTypeEQ(options.ResourceType))
	}
	if options.ResourceID != "" {
		query.Where(auditlog.ResourceIDEQ(options.ResourceID))
	}
	if options.StartTime != nil {
		query.Where(auditlog.CreatedAtGTE(*options.StartTime))
	}
	if options.EndTime != nil {
		query.Where(auditlog.CreatedAtLT(*options.EndTime))
	}
	total, err := query.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := query.Order(auditlog.ByCreatedAt(entsql.OrderDesc())).Offset((options.Page - 1) * options.PageSize).Limit(options.PageSize).All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.AuditLog, 0, len(items))
	userIDs := make(map[uuid.UUID]struct{})
	for _, item := range items {
		mapped, mapErr := auditLogToBiz(item)
		if mapErr != nil {
			return nil, mapErr
		}
		if mapped.UserID != nil {
			userIDs[*mapped.UserID] = struct{}{}
		}
		if targetID := auditTargetUserID(mapped); targetID != nil {
			userIDs[*targetID] = struct{}{}
		}
		result = append(result, mapped)
	}
	if len(userIDs) > 0 {
		ids := make([]uuid.UUID, 0, len(userIDs))
		for id := range userIDs {
			ids = append(ids, id)
		}
		users, userErr := r.data.db.User.Query().Where(userent.IDIn(ids...)).All(ctx)
		if userErr != nil {
			return nil, userErr
		}
		names := make(map[uuid.UUID]string, len(users))
		for _, item := range users {
			names[item.ID] = item.DisplayName
		}
		for _, item := range result {
			if item.UserID != nil {
				if name, ok := names[*item.UserID]; ok {
					item.ActorDisplayName = &name
				}
			}
			if targetID := auditTargetUserID(item); targetID != nil {
				if name, ok := names[*targetID]; ok {
					item.TargetDisplayName = &name
				}
			}
		}
	}
	return &biz.AdminAuditLogList{Items: result, Total: total, Page: options.Page, PageSize: options.PageSize}, nil
}

func auditTargetUserID(item *biz.AuditLog) *uuid.UUID {
	if item == nil || !strings.HasPrefix(item.Action, "admin.user.") {
		return nil
	}
	value := ""
	if item.ResourceID != nil {
		value = *item.ResourceID
	} else {
		value = item.Details["resource_id"]
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil
	}
	return &id
}

func auditLogToBiz(item *ent.AuditLog) (*biz.AuditLog, error) {
	details := make(map[string]string)
	if len(item.Details) > 0 {
		if err := json.Unmarshal(item.Details, &details); err != nil {
			return nil, err
		}
	}
	return &biz.AuditLog{
		ID: item.ID, OrganizationID: item.OrganizationID, UserID: item.UserID,
		Action: item.Action, ResourceType: optionalAuditString(item.ResourceType), ResourceID: optionalAuditString(item.ResourceID),
		Result: item.Result.String(), RequestID: item.RequestID, TraceID: item.TraceID, IPAddress: item.IPAddress,
		Details: details, CreatedAt: item.CreatedAt,
	}, nil
}

func optionalAuditString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
