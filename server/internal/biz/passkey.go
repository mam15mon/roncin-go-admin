package biz

import (
	"context"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

var (
	ErrPasskeyDisabled = errors.ServiceUnavailable("AUTH_PASSKEY_DISABLED", "通行密钥尚未启用")
	ErrPasskeyOrigin   = errors.Forbidden("AUTH_PASSKEY_ORIGIN_FORBIDDEN", "当前访问地址不支持通行密钥，请使用已配置的安全站点")
	ErrPasskeyInvalid  = errors.Unauthorized("AUTH_PASSKEY_INVALID", "通行密钥验证失败，请重新尝试")
	ErrPasskeyInput    = errors.BadRequest("AUTH_PASSKEY_INPUT_INVALID", "通行密钥名称或验证内容不正确")
	ErrPasskeyExists   = errors.Conflict("AUTH_PASSKEY_EXISTS", "此通行密钥已经注册")
	ErrPasskeyNotFound = errors.NotFound("AUTH_PASSKEY_NOT_FOUND", "通行密钥不存在")
	ErrPasskeyBusy     = errors.ServiceUnavailable("AUTH_PASSKEY_BUSY", "通行密钥请求较多，请稍后重试")
)

type PasskeySettings struct {
	Enabled      bool
	RPID, RPName string
	Origins      []string
}
type PasskeyCaller struct{ Origin, Host string }
type PasskeyCeremony struct {
	ID, OptionsJSON string
	ExpiresAt       time.Time
}
type PasskeyCredentialRecord struct {
	ID         uuid.UUID
	Name       string
	CreatedAt  time.Time
	LastUsedAt *time.Time
}

// PasskeyRepo 把库会话、公钥数据与验签保留在数据层，领域只接收本站用户身份。
type PasskeyRepo interface {
	Settings() PasskeySettings
	BeginLogin(context.Context, string) (*PasskeyCeremony, error)
	FinishLogin(context.Context, string, string) (*Credential, error)
	BeginRegistration(context.Context, uuid.UUID, string) (*PasskeyCeremony, error)
	FinishRegistration(context.Context, uuid.UUID, string, string, string, *AuditEvent) (*PasskeyCredentialRecord, error)
	DiscardCeremony(string)
	List(context.Context, uuid.UUID) ([]PasskeyCredentialRecord, error)
	Delete(context.Context, uuid.UUID, uuid.UUID, *AuditEvent) error
}

type PasskeyUsecase struct {
	repo       PasskeyRepo
	auth       *AuthUsecase
	transactor Transactor
}

func NewPasskeyUsecase(repo PasskeyRepo, auth *AuthUsecase, transactor Transactor) *PasskeyUsecase {
	return &PasskeyUsecase{repo: repo, auth: auth, transactor: transactor}
}

func (uc *PasskeyUsecase) Config(caller PasskeyCaller) PasskeySettings {
	settings := uc.repo.Settings()
	if _, err := uc.allowedOrigin(caller, false); err != nil {
		settings.Enabled = false
	}
	return settings
}

func (uc *PasskeyUsecase) allowedOrigin(caller PasskeyCaller, requireHeader bool) (string, error) {
	settings := uc.repo.Settings()
	if !settings.Enabled {
		return "", ErrPasskeyDisabled
	}
	for _, origin := range settings.Origins {
		parsed, err := url.Parse(origin)
		if err != nil || !strings.EqualFold(caller.Host, parsed.Host) {
			continue
		}
		if caller.Origin == origin || (!requireHeader && caller.Origin == "") {
			return origin, nil
		}
	}
	return "", ErrPasskeyOrigin
}

func (uc *PasskeyUsecase) BeginLogin(ctx context.Context, caller PasskeyCaller) (*PasskeyCeremony, error) {
	origin, err := uc.allowedOrigin(caller, true)
	if err != nil {
		return nil, err
	}
	return uc.repo.BeginLogin(ctx, origin)
}
func (uc *PasskeyUsecase) FinishLogin(ctx context.Context, caller PasskeyCaller, ceremonyID, response, userAgent string) (*AuthSessionResult, error) {
	defer uc.repo.DiscardCeremony(ceremonyID)
	if _, err := uc.allowedOrigin(caller, true); err != nil {
		return nil, err
	}
	if ceremonyID == "" || len(response) == 0 || len(response) > 1<<20 {
		return nil, ErrPasskeyInvalid
	}
	var result *AuthSessionResult
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		credential, err := uc.repo.FinishLogin(txCtx, ceremonyID, response)
		if err != nil {
			return err
		}
		if !credential.Enabled || credential.PendingAssignment {
			return ErrInvalidCredentials
		}
		choices, err := uc.auth.listOrganizationChoices(txCtx, credential.UserID)
		if err != nil {
			return err
		}
		if len(choices) == 0 {
			return ErrOrganizationForbidden
		}
		result, err = uc.auth.createSession(txCtx, credential, choices[0].OrganizationID, choices, userAgent, "auth.passkey.login", "")
		return err
	})
	return result, err
}
func (uc *PasskeyUsecase) BeginRegistration(ctx context.Context, caller PasskeyCaller, principal *Principal) (*PasskeyCeremony, error) {
	origin, err := uc.allowedOrigin(caller, true)
	if err != nil {
		return nil, err
	}
	return uc.repo.BeginRegistration(ctx, principal.UserID, origin)
}
func (uc *PasskeyUsecase) FinishRegistration(ctx context.Context, caller PasskeyCaller, principal *Principal, ceremonyID, response, name string) (*PasskeyCredentialRecord, error) {
	defer uc.repo.DiscardCeremony(ceremonyID)
	if _, err := uc.allowedOrigin(caller, true); err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > 100 || len(response) == 0 || len(response) > 1<<20 {
		return nil, ErrPasskeyInput
	}
	if ceremonyID == "" {
		return nil, ErrPasskeyInvalid
	}
	var result *PasskeyCredentialRecord
	err := uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		var err error
		result, err = uc.repo.FinishRegistration(txCtx, principal.UserID, ceremonyID, response, name, &AuditEvent{OrganizationID: &principal.Organization.ID, UserID: &principal.UserID, Action: "auth.passkey.register", Result: "success"})
		return err
	})
	return result, err
}
func (uc *PasskeyUsecase) List(ctx context.Context, caller PasskeyCaller, principal *Principal) ([]PasskeyCredentialRecord, error) {
	if _, err := uc.allowedOrigin(caller, false); err != nil {
		return nil, err
	}
	return uc.repo.List(ctx, principal.UserID)
}
func (uc *PasskeyUsecase) Delete(ctx context.Context, caller PasskeyCaller, principal *Principal, id uuid.UUID) error {
	if _, err := uc.allowedOrigin(caller, true); err != nil {
		return err
	}
	return uc.transactor.WithinTransaction(ctx, func(txCtx context.Context) error {
		return uc.repo.Delete(txCtx, principal.UserID, id, &AuditEvent{OrganizationID: &principal.Organization.ID, UserID: &principal.UserID, Action: "auth.passkey.delete", Result: "success"})
	})
}
