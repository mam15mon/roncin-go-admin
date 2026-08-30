package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

type AdminAuditLogListOptions struct {
	Page         int
	PageSize     int
	Action       string
	UserID       *uuid.UUID
	StartTime    *time.Time
	EndTime      *time.Time
	ResourceType string
	ResourceID   string
}

type AdminAuditLogList = PagedList[*AuditLog]

func (uc *AdminUsecase) ListAuditLogs(ctx context.Context, organizationID uuid.UUID, options AdminAuditLogListOptions) (*AdminAuditLogList, error) {
	if organizationID == uuid.Nil || !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrAdminInvalidArgument
	}
	options.Action = strings.TrimSpace(options.Action)
	options.ResourceType = strings.TrimSpace(options.ResourceType)
	options.ResourceID = strings.TrimSpace(options.ResourceID)
	if utf8.RuneCountInString(options.ResourceType) > 100 || utf8.RuneCountInString(options.ResourceID) > 160 {
		return nil, ErrAdminInvalidArgument
	}
	if options.StartTime != nil && options.EndTime != nil && options.StartTime.After(*options.EndTime) {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListAuditLogs(ctx, organizationID, options)
}
