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
	"sync"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

const (
	dingTalkAuthorizeURL = "https://login.dingtalk.com/oauth2/auth"
	dingTalkTokenURL     = "https://api.dingtalk.com/v1.0/oauth2/userAccessToken"
	dingTalkProfileURL   = "https://api.dingtalk.com/v1.0/contact/users/me"
	dingTalkAppTokenURL  = "https://api.dingtalk.com/v1.0/oauth2/accessToken"
	dingTalkUserIDURL    = "https://oapi.dingtalk.com/topapi/user/getbyunionid"
	dingTalkRobotURL     = "https://api.dingtalk.com/v1.0/robot/oToMessages/batchSend"
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

type dingTalkAppTokenResponse struct {
	AccessToken string `json:"accessToken"`
	ExpireIn    int64  `json:"expireIn"`
}

type dingTalkUserIDResponse struct {
	ErrorCode    int    `json:"errcode"`
	ErrorMessage string `json:"errmsg"`
	Result       struct {
		UserID string `json:"userid"`
	} `json:"result"`
}

type dingTalkRobotResponse struct {
	ProcessQueryKey string `json:"processQueryKey"`
}

type dingTalkIdentityProvider struct {
	enabled      bool
	clientID     string
	clientSecret string
	redirectURI  string
	corpID       string
	client       *http.Client
	tokenMu      sync.Mutex
	appToken     string
	appTokenTill time.Time
}

func NewDingTalkIdentityProvider(security *conf.Security) (*dingTalkIdentityProvider, error) {
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
	values.Set("corpId", p.corpID)
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
	userID, err := p.resolveEnterpriseUserID(ctx, unionID)
	if err != nil {
		return nil, err
	}
	return &biz.DingTalkIdentity{UnionID: unionID, UserID: userID, CorpID: corpID, Name: name, Email: profile.Email, AvatarURL: profile.AvatarURL}, nil
}

func (p *dingTalkIdentityProvider) resolveEnterpriseUserID(ctx context.Context, unionID string) (string, error) {
	accessToken, err := p.enterpriseAccessToken(ctx)
	if err != nil {
		return "", biz.ErrDingTalkPermissionDenied
	}
	endpoint := dingTalkUserIDURL + "?access_token=" + url.QueryEscape(accessToken)
	var response dingTalkUserIDResponse
	status, err := p.requestJSON(ctx, http.MethodPost, endpoint, map[string]string{"unionid": unionID}, "", &response)
	if err != nil || status == http.StatusForbidden || response.ErrorCode != 0 {
		return "", biz.ErrDingTalkPermissionDenied
	}
	userID := strings.TrimSpace(response.Result.UserID)
	if userID == "" {
		return "", biz.ErrDingTalkPermissionDenied
	}
	return userID, nil
}

func (p *dingTalkIdentityProvider) enterpriseAccessToken(ctx context.Context) (string, error) {
	p.tokenMu.Lock()
	defer p.tokenMu.Unlock()
	if p.appToken != "" && time.Now().Before(p.appTokenTill) {
		return p.appToken, nil
	}
	var response dingTalkAppTokenResponse
	_, err := p.requestJSON(ctx, http.MethodPost, dingTalkAppTokenURL, map[string]string{
		"appKey":    p.clientID,
		"appSecret": p.clientSecret,
	}, "", &response)
	if err != nil {
		return "", err
	}
	accessToken := strings.TrimSpace(response.AccessToken)
	if accessToken == "" || response.ExpireIn <= 0 {
		return "", fmt.Errorf("钉钉企业访问令牌响应不完整")
	}
	refreshAdvance := 5 * time.Minute
	lifetime := time.Duration(response.ExpireIn) * time.Second
	if lifetime <= refreshAdvance {
		refreshAdvance = lifetime / 10
	}
	p.appToken = accessToken
	p.appTokenTill = time.Now().Add(lifetime - refreshAdvance)
	return accessToken, nil
}

func (p *dingTalkIdentityProvider) SendText(ctx context.Context, userID, content string) error {
	if !p.enabled {
		return biz.ErrDingTalkDisabled
	}
	userID = strings.TrimSpace(userID)
	content = strings.TrimSpace(content)
	if userID == "" || content == "" {
		return fmt.Errorf("钉钉通知收件人或内容为空")
	}
	accessToken, err := p.enterpriseAccessToken(ctx)
	if err != nil {
		return fmt.Errorf("获取钉钉企业访问令牌: %w", err)
	}
	messageParameter, err := json.Marshal(map[string]string{"content": content})
	if err != nil {
		return err
	}
	payload := map[string]any{
		"robotCode": p.clientID,
		"userIds":   []string{userID},
		"msgKey":    "sampleText",
		"msgParam":  string(messageParameter),
	}
	var response dingTalkRobotResponse
	if _, err := p.requestJSON(ctx, http.MethodPost, dingTalkRobotURL, payload, accessToken, &response); err != nil {
		return err
	}
	if strings.TrimSpace(response.ProcessQueryKey) == "" {
		return fmt.Errorf("钉钉机器人发送响应不完整")
	}
	return nil
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
var _ biz.DingTalkNotificationSender = (*dingTalkIdentityProvider)(nil)
