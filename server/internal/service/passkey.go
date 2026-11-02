package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v3/transport"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type PasskeyService struct {
	v1.UnimplementedPasskeyServiceServer
	usecase *biz.PasskeyUsecase
	auth    *AuthService
}

func NewPasskeyService(usecase *biz.PasskeyUsecase, auth *AuthService) *PasskeyService {
	return &PasskeyService{usecase: usecase, auth: auth}
}
func (s *PasskeyService) caller(ctx context.Context) biz.PasskeyCaller {
	if request, ok := kratoshttp.RequestFromServerContext(ctx); ok {
		return biz.PasskeyCaller{Origin: request.Header.Get("Origin"), Host: request.Host}
	}
	return biz.PasskeyCaller{}
}
func (s *PasskeyService) challengeName(purpose string) string {
	return s.auth.policy.CookieName + "_passkey_" + purpose
}
func (s *PasskeyService) readChallenge(ctx context.Context, purpose string) string {
	if tr, ok := transport.FromServerContext(ctx); ok {
		return cookieValue(tr.RequestHeader().Get("Cookie"), s.challengeName(purpose))
	}
	return ""
}
func (s *PasskeyService) writeChallenge(ctx context.Context, purpose string, ceremony *biz.PasskeyCeremony) {
	s.auth.setCookieNamed(ctx, s.challengeName(purpose), ceremony.ID, ceremony.ExpiresAt, int(time.Until(ceremony.ExpiresAt)/time.Second))
}
func (s *PasskeyService) clearChallenge(ctx context.Context, purpose string) {
	s.auth.setCookieNamed(ctx, s.challengeName(purpose), "", time.Unix(1, 0), -1)
}
func (s *PasskeyService) GetPasskeyConfig(ctx context.Context, _ *v1.GetPasskeyConfigRequest) (*v1.GetPasskeyConfigResponse, error) {
	authNoStore(ctx)
	config := s.usecase.Config(s.caller(ctx))
	return ok(ctx, &v1.GetPasskeyConfigResponse{Data: &v1.PasskeyConfig{Enabled: config.Enabled, RpId: config.RPID, RpName: config.RPName, Origins: config.Origins}}), nil
}
func (s *PasskeyService) BeginPasskeyLogin(ctx context.Context, _ *v1.BeginPasskeyLoginRequest) (*v1.BeginPasskeyLoginResponse, error) {
	authNoStore(ctx)
	ceremony, err := s.usecase.BeginLogin(ctx, s.caller(ctx))
	if err != nil {
		return nil, err
	}
	s.writeChallenge(ctx, "login", ceremony)
	return ok(ctx, &v1.BeginPasskeyLoginResponse{Data: &v1.PasskeyCeremony{OptionsJson: ceremony.OptionsJSON, ExpiresAt: ceremony.ExpiresAt.Format(time.RFC3339Nano)}}), nil
}
func (s *PasskeyService) FinishPasskeyLogin(ctx context.Context, request *v1.FinishPasskeyLoginRequest) (*v1.FinishPasskeyLoginResponse, error) {
	authNoStore(ctx)
	challenge := s.readChallenge(ctx, "login")
	s.clearChallenge(ctx, "login")
	userAgent := ""
	if tr, ok := transport.FromServerContext(ctx); ok {
		userAgent = tr.RequestHeader().Get("User-Agent")
	}
	result, err := s.usecase.FinishLogin(ctx, s.caller(ctx), challenge, request.GetCredentialJson(), userAgent)
	if err != nil {
		return nil, err
	}
	s.auth.setSessionCookies(ctx, result.Token, rememberLogin(request.RememberMe))
	return ok(ctx, &v1.FinishPasskeyLoginResponse{Data: principalToAPI(result.Principal), OrganizationChoices: organizationChoicesToAPI(result.OrganizationChoices)}), nil
}
func (s *PasskeyService) BeginPasskeyRegistration(ctx context.Context, _ *v1.BeginPasskeyRegistrationRequest) (*v1.BeginPasskeyRegistrationResponse, error) {
	authNoStore(ctx)
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	ceremony, err := s.usecase.BeginRegistration(ctx, s.caller(ctx), principal)
	if err != nil {
		return nil, err
	}
	s.writeChallenge(ctx, "registration", ceremony)
	return ok(ctx, &v1.BeginPasskeyRegistrationResponse{Data: &v1.PasskeyCeremony{OptionsJson: ceremony.OptionsJSON, ExpiresAt: ceremony.ExpiresAt.Format(time.RFC3339Nano)}}), nil
}
func (s *PasskeyService) FinishPasskeyRegistration(ctx context.Context, request *v1.FinishPasskeyRegistrationRequest) (*v1.FinishPasskeyRegistrationResponse, error) {
	authNoStore(ctx)
	challenge := s.readChallenge(ctx, "registration")
	s.clearChallenge(ctx, "registration")
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	record, err := s.usecase.FinishRegistration(ctx, s.caller(ctx), principal, challenge, request.GetCredentialJson(), request.GetName())
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.FinishPasskeyRegistrationResponse{Data: passkeyCredentialToAPI(record)}), nil
}
func (s *PasskeyService) ListPasskeys(ctx context.Context, _ *v1.ListPasskeysRequest) (*v1.ListPasskeysResponse, error) {
	authNoStore(ctx)
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	records, err := s.usecase.List(ctx, s.caller(ctx), principal)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.PasskeyCredential, 0, len(records))
	for i := range records {
		items = append(items, passkeyCredentialToAPI(&records[i]))
	}
	return ok(ctx, &v1.ListPasskeysResponse{Data: &v1.PasskeyCredentialList{Items: items}}), nil
}
func (s *PasskeyService) DeletePasskey(ctx context.Context, request *v1.DeletePasskeyRequest) (*v1.DeletePasskeyResponse, error) {
	authNoStore(ctx)
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	id, err := uuid.Parse(request.GetId())
	if err != nil {
		return nil, biz.ErrPasskeyNotFound
	}
	if err = s.usecase.Delete(ctx, s.caller(ctx), principal, id); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.DeletePasskeyResponse{}), nil
}
func passkeyCredentialToAPI(record *biz.PasskeyCredentialRecord) *v1.PasskeyCredential {
	result := &v1.PasskeyCredential{Id: record.ID.String(), Name: record.Name, CreatedAt: record.CreatedAt.Format(time.RFC3339Nano)}
	if record.LastUsedAt != nil {
		value := record.LastUsedAt.Format(time.RFC3339Nano)
		result.LastUsedAt = &value
	}
	return result
}
