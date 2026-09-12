package biz

import (
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrDingTalkInvitationNotFound     = errors.NotFound("DINGTALK_INVITATION_NOT_FOUND", "邀请不存在")
	ErrDingTalkInvitationExists       = errors.Conflict("DINGTALK_INVITATION_EXISTS", "该手机号在目标组织已存在有效邀请")
	ErrDingTalkInvitationNotRevocable = errors.Conflict("DINGTALK_INVITATION_NOT_REVOCABLE", "邀请已消费或已撤销，不能再撤销")
	// ErrDingTalkInvitationNotConsumable 表示邀请在消费窗口外（并发消费、过期、
	// 目标组织或角色失效）；仅内部使用，登录链路据此降级人工审批通道，不外抛。
	ErrDingTalkInvitationNotConsumable   = errors.Conflict("DINGTALK_INVITATION_NOT_CONSUMABLE", "邀请不可消费")
	ErrDingTalkRegistrationNotFound      = errors.NotFound("DINGTALK_REGISTRATION_NOT_FOUND", "注册申请不存在或已处理")
	ErrDingTalkRegistrationProcessed     = errors.Conflict("DINGTALK_REGISTRATION_ALREADY_PROCESSED", "该注册申请已处理")
	ErrDingTalkRegistrationOrgInvalid    = errors.BadRequest("DINGTALK_REGISTRATION_ORGANIZATION_INVALID", "所选组织无效或不可选")
	ErrDingTalkRegistrationReasonMissing = errors.BadRequest("DINGTALK_REGISTRATION_REASON_MISSING", "拒绝原因不能为空")
)

// 邀请默认有效期与上下限（小时）。
const (
	DingTalkInvitationDefaultTTLHours = 72
	DingTalkInvitationMinTTLHours     = 1
	DingTalkInvitationMaxTTLHours     = 720
)

type DingTalkInvitationStatus string

const (
	DingTalkInvitationStatusPending  DingTalkInvitationStatus = "PENDING"
	DingTalkInvitationStatusConsumed DingTalkInvitationStatus = "CONSUMED"
	DingTalkInvitationStatusExpired  DingTalkInvitationStatus = "EXPIRED"
	DingTalkInvitationStatusRevoked  DingTalkInvitationStatus = "REVOKED"
)

// DingTalkInvitation 是管理员预建的扫码邀请。数据归属原则：只承载管理员决策
// 字段（手机号/目标组织/初始角色/备注姓名），账号身份（姓名/头像/unionId）永远
// 以钉钉扫码返回为准，Mobile 存规范化明文且仅用于等值匹配，出展示层必须脱敏。
type DingTalkInvitation struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	OrganizationName string
	RoleID           uuid.UUID
	RoleName         string
	Mobile           string
	DisplayName      string
	InvitedBy        uuid.UUID
	InviterName      string
	Status           DingTalkInvitationStatus
	ConsumedByName   string
	ConsumedAt       *time.Time
	ExpiresAt        time.Time
	CreatedAt        time.Time
}

// EffectiveStatus 返回考虑过期时间后的展示状态：PENDING 且已过期的邀请按
// EXPIRED 展示；匹配链只消费未过期的 PENDING，过期行由该口径惰性判定。
func (i *DingTalkInvitation) EffectiveStatus(now time.Time) DingTalkInvitationStatus {
	if i == nil || i.Status != DingTalkInvitationStatusPending {
		if i == nil {
			return ""
		}
		return i.Status
	}
	if !i.ExpiresAt.After(now) {
		return DingTalkInvitationStatusExpired
	}
	return DingTalkInvitationStatusPending
}

type DingTalkInvitationListOptions struct {
	Page           int
	PageSize       int
	OrganizationID uuid.UUID // 组织过滤；零值表示范围内全部组织
	Status         *DingTalkInvitationStatus
}

type DingTalkInvitationList = PagedList[*DingTalkInvitation]

var dingTalkMobilePattern = regexp.MustCompile(`^1[3-9]\d{9}$`)

// NormalizeDingTalkMobile 归一化国内手机号：去除空白与连字符，剥离 +86/0086
// 国际前缀后返回 11 位本体；不剥离裸 86 前缀（11 位号码可能合法地以 86 开头）。
func NormalizeDingTalkMobile(raw string) string {
	var builder strings.Builder
	for _, symbol := range strings.TrimSpace(raw) {
		if unicode.IsSpace(symbol) || symbol == '-' {
			continue
		}
		builder.WriteRune(symbol)
	}
	value := builder.String()
	for _, prefix := range []string{"+86", "0086"} {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimPrefix(value, prefix)
		}
	}
	return value
}

// ValidDingTalkMobile 校验归一化后的手机号是否为合法国内 11 位号码。
func ValidDingTalkMobile(mobile string) bool {
	return dingTalkMobilePattern.MatchString(mobile)
}

// MaskDingTalkMobile 返回 138****1234 形式的脱敏手机号；位数不足时降低保留位数，
// 任何情况下都不输出完整号码。日志与审计 Details 只允许记录该形式。
func MaskDingTalkMobile(mobile string) string {
	mobile = NormalizeDingTalkMobile(mobile)
	digits := []rune(mobile)
	switch {
	case len(digits) >= 11:
		return string(digits[:3]) + "****" + string(digits[len(digits)-4:])
	case len(digits) >= 6:
		return string(digits[:3]) + "****" + string(digits[len(digits)-2:])
	case len(digits) >= 3:
		return string(digits[:2]) + "****"
	default:
		return "****"
	}
}
