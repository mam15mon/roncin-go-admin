package data

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

const (
	dingTalkAuthorizeURL = "https://login.dingtalk.com/oauth2/auth"
	dingTalkTokenURL     = "https://api.dingtalk.com/v1.0/oauth2/userAccessToken"
	dingTalkProfileURL   = "https://api.dingtalk.com/v1.0/contact/users/me"
)

type dingTalkTokenResponse struct {
	AccessToken string `json:"accessToken"`
	CorpID      string `json:"corpId"`
}

type dingTalkProfileResponse struct {
	UnionID   string  `json:"unionId"`
	Nick      string  `json:"nick"`
	Email     *string `json:"email"`
	AvatarURL *string `json:"avatarUrl"`
}

type dingTalkIdentityProvider struct {
	enabled      bool
	clientID     string
	clientSecret string
	redirectURI  string
	corpID       string
	client       *http.Client
}

func NewDingTalkIdentityProvider(security *conf.Security) (biz.DingTalkIdentityProvider, error) {
	provider := &dingTalkIdentityProvider{client: &http.Client{Timeout: 5 * time.Second}}
	if security == nil || security.Dingtalk == nil || !security.Dingtalk.Enabled {
		return provider, nil
	}
	config := security.Dingtalk
	provider.enabled = true
	provider.clientID = strings.TrimSpace(config.ClientId)
	provider.clientSecret = strings.TrimSpace(config.ClientSecret)
	provider.redirectURI = strings.TrimSpace(config.RedirectUri)
	provider.corpID = strings.TrimSpace(config.CorpId)
	if provider.clientID == "" || provider.clientSecret == "" || provider.redirectURI == "" || provider.corpID == "" {
		return nil, fmt.Errorf("钉钉认证已启用，但 corp_id、client_id、client_secret 或 redirect_uri 未配置")
	}
	if parsed, err := url.ParseRequestURI(provider.redirectURI); err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("钉钉 redirect_uri 必须是完整 URL")
	}
	return provider, nil
}

func (p *dingTalkIdentityProvider) Enabled() bool { return p.enabled }

func (p *dingTalkIdentityProvider) AuthorizeURL(state string) (string, error) {
	if !p.enabled {
		return "", biz.ErrDingTalkDisabled
	}
	values := url.Values{}
	values.Set("redirect_uri", p.redirectURI)
	values.Set("response_type", "code")
	values.Set("client_id", p.clientID)
	values.Set("scope", "openid corpid")
	values.Set("state", state)
	values.Set("prompt", "consent")
	return dingTalkAuthorizeURL + "?" + values.Encode(), nil
}

func (p *dingTalkIdentityProvider) ResolveIdentity(ctx context.Context, authCode string) (*biz.DingTalkIdentity, error) {
	tokenPayload := map[string]string{
		"clientId":     p.clientID,
		"clientSecret": p.clientSecret,
		"code":         authCode,
		"grantType":    "authorization_code",
	}
	var token dingTalkTokenResponse
	if status, err := p.requestJSON(ctx, http.MethodPost, dingTalkTokenURL, tokenPayload, "", &token); err != nil {
		if status == http.StatusBadRequest || status == http.StatusUnauthorized {
			return nil, biz.ErrDingTalkCodeInvalid
		}
		return nil, biz.ErrDingTalkLoginFailed
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return nil, biz.ErrDingTalkLoginFailed
	}
	corpID := strings.TrimSpace(token.CorpID)
	if corpID == "" || subtle.ConstantTimeCompare([]byte(corpID), []byte(p.corpID)) != 1 {
		return nil, biz.ErrDingTalkOrganizationMismatch
	}

	var profile dingTalkProfileResponse
	if status, err := p.requestJSON(ctx, http.MethodGet, dingTalkProfileURL, nil, token.AccessToken, &profile); err != nil {
		if status == http.StatusForbidden {
			return nil, biz.ErrDingTalkPermissionDenied
		}
		return nil, biz.ErrDingTalkLoginFailed
	}
	unionID := strings.TrimSpace(profile.UnionID)
	name := strings.TrimSpace(profile.Nick)
	if unionID == "" || name == "" {
		return nil, biz.ErrDingTalkLoginFailed
	}
	return &biz.DingTalkIdentity{UnionID: unionID, CorpID: corpID, Name: name, Email: profile.Email, AvatarURL: profile.AvatarURL}, nil
}

func (p *dingTalkIdentityProvider) requestJSON(ctx context.Context, method, endpoint string, payload any, bearerToken string, target any) (int, error) {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return 0, err
		}
		body = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		request.Header.Set("x-acs-dingtalk-access-token", bearerToken)
	}
	response, err := p.client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return response.StatusCode, fmt.Errorf("钉钉接口返回状态码 %d", response.StatusCode)
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(target); err != nil {
		return response.StatusCode, err
	}
	return response.StatusCode, nil
}

var _ biz.DingTalkIdentityProvider = (*dingTalkIdentityProvider)(nil)
