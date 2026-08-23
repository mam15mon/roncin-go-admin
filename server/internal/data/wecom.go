package data

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"

	"github.com/go-kratos/kratos/v3/log"
)

const wecomAPIBaseURL = "https://qyapi.weixin.qq.com/cgi-bin"

type wecomUserIdentityResponse struct {
	ErrCode int    `json:"errcode"`
	UserID  string `json:"userid"`
}

type wecomIdentityProvider struct {
	enabled     bool
	corpID      string
	agentID     string
	secret      string
	redirectURI string
	client      *http.Client
}

func NewWeComIdentityProvider(security *conf.Security) (biz.WeComIdentityProvider, error) {
	provider := &wecomIdentityProvider{client: &http.Client{Timeout: 5 * time.Second}}
	if security == nil || security.GetWecom() == nil || !security.GetWecom().GetEnabled() {
		return provider, nil
	}
	config := security.GetWecom()
	provider.enabled = true
	provider.corpID = strings.TrimSpace(config.GetCorpId())
	provider.agentID = strconv.FormatInt(config.GetAgentId(), 10)
	provider.secret = strings.TrimSpace(config.GetSecret())
	provider.redirectURI = strings.TrimSpace(config.GetRedirectUri())
	if provider.corpID == "" || config.GetAgentId() <= 0 || provider.secret == "" || provider.redirectURI == "" {
		return nil, fmt.Errorf("企业微信登录已启用，但 corp_id、agent_id、secret 或 redirect_uri 未配置")
	}
	redirectURL, err := url.ParseRequestURI(provider.redirectURI)
	if err != nil || redirectURL.Scheme == "" || redirectURL.Host == "" {
		return nil, fmt.Errorf("企业微信 redirect_uri 必须是完整 URL")
	}
	return provider, nil
}

func (p *wecomIdentityProvider) Enabled() bool { return p.enabled }

func (p *wecomIdentityProvider) AuthorizeURL(state string) (string, error) {
	if !p.enabled {
		return "", biz.ErrWeComDisabled
	}
	values := url.Values{
		"appid":        []string{p.corpID},
		"agentid":      []string{p.agentID},
		"redirect_uri": []string{p.redirectURI},
		"state":        []string{state},
	}
	return "https://open.work.weixin.qq.com/wwopen/sso/qrConnect?" + values.Encode(), nil
}

func (p *wecomIdentityProvider) ResolveIdentity(ctx context.Context, code string) (*biz.WeComIdentity, error) {
	accessToken, err := p.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	identityValues := url.Values{"access_token": []string{accessToken}, "code": []string{code}}
	var identityResponse wecomUserIdentityResponse
	if err := p.getJSON(ctx, wecomAPIBaseURL+"/auth/getuserinfo?"+identityValues.Encode(), &identityResponse); err != nil {
		return nil, biz.ErrWeComLoginFailed
	}
	if identityResponse.ErrCode != 0 {
		return nil, mapWeComAPIError("auth.getuserinfo", identityResponse.ErrCode)
	}
	if identityResponse.UserID == "" {
		return nil, biz.ErrWeComLoginFailed
	}

	profileValues := url.Values{"access_token": []string{accessToken}, "userid": []string{identityResponse.UserID}}
	var profileResponse struct {
		ErrCode int    `json:"errcode"`
		UserID  string `json:"userid"`
		Name    string `json:"name"`
		Email   string `json:"email"`
	}
	if err := p.getJSON(ctx, wecomAPIBaseURL+"/user/get?"+profileValues.Encode(), &profileResponse); err != nil {
		return nil, biz.ErrWeComLoginFailed
	}
	if profileResponse.ErrCode != 0 {
		return nil, mapWeComAPIError("user.get", profileResponse.ErrCode)
	}
	if profileResponse.UserID != identityResponse.UserID || strings.TrimSpace(profileResponse.Name) == "" {
		return nil, biz.ErrWeComLoginFailed
	}
	identity := &biz.WeComIdentity{UserID: profileResponse.UserID, Name: strings.TrimSpace(profileResponse.Name)}
	if strings.TrimSpace(profileResponse.Email) != "" {
		email := strings.TrimSpace(profileResponse.Email)
		identity.Email = &email
	}
	return identity, nil
}

func (p *wecomIdentityProvider) accessToken(ctx context.Context) (string, error) {
	values := url.Values{"corpid": []string{p.corpID}, "corpsecret": []string{p.secret}}
	var response struct {
		ErrCode     int    `json:"errcode"`
		AccessToken string `json:"access_token"`
	}
	if err := p.getJSON(ctx, wecomAPIBaseURL+"/gettoken?"+values.Encode(), &response); err != nil {
		return "", biz.ErrWeComLoginFailed
	}
	if response.ErrCode != 0 {
		return "", mapWeComAPIError("gettoken", response.ErrCode)
	}
	if response.AccessToken == "" {
		return "", biz.ErrWeComLoginFailed
	}
	return response.AccessToken, nil
}

func mapWeComAPIError(stage string, code int) error {
	log.Error("企业微信接口返回错误", "stage", stage, "errcode", code)
	switch code {
	case 40029:
		return biz.ErrWeComCodeInvalid
	case 60020:
		return biz.ErrWeComTrustedIPRequired
	case 48002:
		return biz.ErrWeComPermissionDenied
	default:
		return biz.ErrWeComLoginFailed
	}
}

func (p *wecomIdentityProvider) getJSON(ctx context.Context, endpoint string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	response, err := p.client.Do(request)
	if err != nil {
		return fmt.Errorf("企业微信接口请求失败")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("企业微信接口返回状态码 %d", response.StatusCode)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

var _ biz.WeComIdentityProvider = (*wecomIdentityProvider)(nil)
