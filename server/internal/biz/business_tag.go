package biz

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var ErrBusinessTagInvalidArgument = errors.BadRequest("BUSINESS_TAG_INVALID_ARGUMENT", "业务标签参数不合法")

// BusinessTagSummary 描述业务对象上展示的组织标签概要。
type BusinessTagSummary struct {
	ID         uuid.UUID
	Name       string
	GroupID    uuid.UUID
	GroupName  string
	GroupColor string
	Enabled    bool
}

// BusinessTagRepo 定义订单、费用和账单共用的标签查询与关联写入接口。
type BusinessTagRepo interface {
	ListTagOptions(ctx context.Context, organizationID uuid.UUID, keyword string, page, pageSize int) ([]*BusinessTagSummary, int64, error)
	LoadOrderTags(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]*BusinessTagSummary, error)
	AssignOrderTags(ctx context.Context, organizationID uuid.UUID, businessType OrderBusinessType, orderIDs, tagResourceIDs []uuid.UUID, audit *AuditEvent) (int, error)
	RemoveOrderTags(ctx context.Context, organizationID uuid.UUID, businessType OrderBusinessType, orderIDs, tagResourceIDs []uuid.UUID, audit *AuditEvent) (int, error)
	CountTagUsages(ctx context.Context, organizationID, tagResourceID uuid.UUID) (partnerCount, orderCount, orderFeeCount, financeBillCount int, err error)
}

type BusinessTagUsecase struct {
	repo BusinessTagRepo
}

func NewBusinessTagUsecase(repo BusinessTagRepo) *BusinessTagUsecase {
	return &BusinessTagUsecase{repo: repo}
}

func (uc *BusinessTagUsecase) ListTagOptions(ctx context.Context, organizationID uuid.UUID, keyword string, page, pageSize int) ([]*BusinessTagSummary, int64, error) {
	keyword = strings.TrimSpace(keyword)
	if organizationID == uuid.Nil || !ValidListPagination(page, pageSize) || utf8.RuneCountInString(keyword) > 100 {
		return nil, 0, ErrBusinessTagInvalidArgument
	}
	return uc.repo.ListTagOptions(ctx, organizationID, keyword, page, pageSize)
}

func (uc *BusinessTagUsecase) LoadOrderTags(ctx context.Context, orderIDs []uuid.UUID) (map[uuid.UUID][]*BusinessTagSummary, error) {
	if len(orderIDs) == 0 {
		return map[uuid.UUID][]*BusinessTagSummary{}, nil
	}
	return uc.repo.LoadOrderTags(ctx, orderIDs)
}

func (uc *BusinessTagUsecase) AssignOrderTags(ctx context.Context, organizationID, actorID uuid.UUID, businessType OrderBusinessType, orderIDs, tagResourceIDs []uuid.UUID) (int, error) {
	orderIDs, tagResourceIDs, err := normalizeBusinessTagBatch(organizationID, businessType, orderIDs, tagResourceIDs)
	if err != nil {
		return 0, err
	}
	audit := businessTagAudit(organizationID, actorID, "order.tag.batch_assign", orderIDs, tagResourceIDs)
	return uc.repo.AssignOrderTags(ctx, organizationID, businessType, orderIDs, tagResourceIDs, audit)
}

func (uc *BusinessTagUsecase) RemoveOrderTags(ctx context.Context, organizationID, actorID uuid.UUID, businessType OrderBusinessType, orderIDs, tagResourceIDs []uuid.UUID) (int, error) {
	orderIDs, tagResourceIDs, err := normalizeBusinessTagBatch(organizationID, businessType, orderIDs, tagResourceIDs)
	if err != nil {
		return 0, err
	}
	audit := businessTagAudit(organizationID, actorID, "order.tag.batch_remove", orderIDs, tagResourceIDs)
	return uc.repo.RemoveOrderTags(ctx, organizationID, businessType, orderIDs, tagResourceIDs, audit)
}

// CountTagUsages 供标签删除防线统计四类关联数量。
func (uc *BusinessTagUsecase) CountTagUsages(ctx context.Context, organizationID, tagResourceID uuid.UUID) (int, int, int, int, error) {
	if organizationID == uuid.Nil || tagResourceID == uuid.Nil {
		return 0, 0, 0, 0, ErrBusinessTagInvalidArgument
	}
	return uc.repo.CountTagUsages(ctx, organizationID, tagResourceID)
}

func normalizeBusinessTagBatch(organizationID uuid.UUID, businessType OrderBusinessType, orderIDs, tagResourceIDs []uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
	if organizationID == uuid.Nil || !businessType.Valid() {
		return nil, nil, ErrBusinessTagInvalidArgument
	}
	orderIDs = uniqueBusinessTagUUIDs(orderIDs)
	tagResourceIDs = uniqueBusinessTagUUIDs(tagResourceIDs)
	if len(orderIDs) == 0 || len(tagResourceIDs) == 0 || len(orderIDs) > 200 || len(tagResourceIDs) > 200 {
		return nil, nil, ErrBusinessTagInvalidArgument
	}
	return orderIDs, tagResourceIDs, nil
}

func uniqueBusinessTagUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if value == uuid.Nil {
			return nil
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func businessTagAudit(organizationID, actorID uuid.UUID, action string, targetIDs, tagResourceIDs []uuid.UUID) *AuditEvent {
	details := map[string]string{
		"order.ids": joinBusinessTagUUIDs(targetIDs),
		"tag.ids":   joinBusinessTagUUIDs(tagResourceIDs),
	}
	return &AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, ResourceType: "enterprise_tag", Result: "success", Details: details}
}

func joinBusinessTagUUIDs(values []uuid.UUID) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = value.String()
	}
	return strings.Join(parts, ",")
}
