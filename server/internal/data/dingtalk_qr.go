package data

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

const (
	dingTalkLoginOrigin   = "https://login.dingtalk.com"
	dingTalkQRMaxAttempts = 2048
	dingTalkQRBodyLimit   = 1 << 20
)

// 扫码协议依据钉钉 h5-dingtalk-login/0.126.3 的生成、长轮询和确认分支实现。
// 上游风控或未知协议响应明确失败，不回退 iframe，也不跳过现有 OAuth 身份验证。
// 每次扫码独占上游 CookieJar 和长轮询锁，不能与其它浏览器交换上游会话。
type dingTalkQRAttempt struct {
	gate      chan struct{}
	client    *http.Client
	params    url.Values
	referer   string
	code      string
	expiresAt time.Time
	finished  bool
}

type dingTalkQRResponse struct {
	Success   bool            `json:"success"`
	ErrorCode string          `json:"errorCode"`
	Result    json.RawMessage `json:"result"`
}

func (p *dingTalkIdentityProvider) CreateQRCode(ctx context.Context, state string, expiresAt time.Time) (*biz.DingTalkQRCode, error) {
	if !p.enabled {
		return nil, biz.ErrDingTalkDisabled
	}
	authorize, err := p.AuthorizeURL(state)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(authorize)
	if err != nil || state == "" || !expiresAt.After(time.Now()) {
		return nil, biz.ErrDingTalkLoginFailed
	}
	query := u.Query()
	// 保持原官方扫码简化协议；该参数控制上游授权响应，不要求浏览器实际嵌入 iframe。
	query.Set("iframe", "true")
	u.RawQuery = query.Encode()
	params := u.Query()
	params.Set("pdmTitle", "Linux Web")
	params.Set("pdmModel", "Unknown")
	// 上游官方脚本在设备模块不可用时使用此值；风控错误仍原样失败，不尝试绕过。
	params.Set("pdmToken", "FAKE_UMID")
	challenge := dingTalkLoginOrigin + "/oauth2/challenge.htm?" + u.RawQuery
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, biz.ErrDingTalkLoginFailed
	}
	client := *p.client
	client.Jar = jar
	client.Timeout = 12 * time.Second
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	attempt := &dingTalkQRAttempt{client: &client, params: params, referer: challenge, expiresAt: expiresAt, gate: make(chan struct{}, 1)}
	// 在开始网络请求前预留容量，防止并发创建突破上限；失败时立即释放。
	p.qrMu.Lock()
	if p.qrAttempts == nil {
		p.qrAttempts = make(map[string]*dingTalkQRAttempt)
	}
	for key, item := range p.qrAttempts {
		if !item.expiresAt.After(time.Now()) {
			delete(p.qrAttempts, key)
		}
	}
	if _, exists := p.qrAttempts[state]; exists || len(p.qrAttempts) >= dingTalkQRMaxAttempts {
		p.qrMu.Unlock()
		return nil, biz.ErrDingTalkLoginFailed
	}
	// gate 在创建结束时释放，轮询不能读取尚未返回的二维码标识。
	p.qrAttempts[state] = attempt
	p.qrMu.Unlock()
	defer func() { attempt.gate <- struct{}{} }()
	ready := false
	defer func() {
		if !ready {
			attempt.finished = true
			p.removeQRAttempt(state, attempt)
		}
	}()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, challenge, nil)
	if err != nil {
		return nil, biz.ErrDingTalkLoginFailed
	}
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response, err := attempt.client.Do(request)
	if err != nil {
		return nil, dingTalkQRError(ctx)
	}
	_, readErr := io.Copy(io.Discard, io.LimitReader(response.Body, dingTalkQRBodyLimit))
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || readErr != nil {
		return nil, biz.ErrDingTalkLoginFailed
	}
	result, err := attempt.post(ctx, "/oauth2/generate_qrcode", params)
	if err != nil {
		return nil, err
	}
	var link string
	if !result.Success || json.Unmarshal(result.Result, &link) != nil {
		return nil, biz.ErrDingTalkLoginFailed
	}
	qrURL, err := url.Parse(link)
	if err != nil || qrURL.Scheme != "https" || qrURL.Host != "login.dingtalk.com" || qrURL.Path != "/oauth2/qr_confirm.htm" || qrURL.User != nil || qrURL.Fragment != "" {
		return nil, biz.ErrDingTalkLoginFailed
	}
	query, err = url.ParseQuery(qrURL.RawQuery)
	if err != nil || len(query) != 1 || len(query["code"]) != 1 || strings.TrimSpace(query.Get("code")) == "" || len(query.Get("code")) > 4096 {
		return nil, biz.ErrDingTalkLoginFailed
	}
	attempt.code = query.Get("code")
	// 与上游当前页面一致，二维码内容必须是真实确认链接，而非普通授权 URL。
	query.Set("dd_mini_app_id", "5000000004998565")
	query.Set("showmenu", "false")
	qrURL.RawQuery = query.Encode()
	if !expiresAt.After(time.Now()) {
		return nil, biz.ErrDingTalkLoginFailed
	}
	ready = true
	return &biz.DingTalkQRCode{URL: qrURL.String()}, nil
}

func (p *dingTalkIdentityProvider) PollQRCode(ctx context.Context, state string) (*biz.DingTalkQRCodePollResult, error) {
	p.qrMu.Lock()
	attempt := p.qrAttempts[state]
	p.qrMu.Unlock()
	if attempt == nil {
		return &biz.DingTalkQRCodePollResult{Status: biz.DingTalkQRCodeExpired}, nil
	}
	// 同次长轮询串行；取消后立即重挂载可以等待旧请求释放，不能误报登录失败。
	// 排队和上游请求共用有界期限，防止重复请求无限等待。
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	select {
	case <-attempt.gate:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	defer func() { attempt.gate <- struct{}{} }()
	if attempt.finished || !attempt.expiresAt.After(time.Now()) {
		p.removeQRAttempt(state, attempt)
		return &biz.DingTalkQRCodePollResult{Status: biz.DingTalkQRCodeExpired}, nil
	}
	params := maps.Clone(attempt.params)
	params.Set("code", attempt.code)
	params.Set("exclusiveCorpId", "")
	params.Set("stayLogin", "false")
	result, err := attempt.post(ctx, "/oauth2/login_with_qr", params)
	if err != nil {
		return nil, err
	}
	if !attempt.expiresAt.After(time.Now()) {
		attempt.finished = true
		p.removeQRAttempt(state, attempt)
		return &biz.DingTalkQRCodePollResult{Status: biz.DingTalkQRCodeExpired}, nil
	}
	if !result.Success {
		switch result.ErrorCode {
		case "11021":
			return &biz.DingTalkQRCodePollResult{Status: biz.DingTalkQRCodeWaiting}, nil
		case "11041":
			return &biz.DingTalkQRCodePollResult{Status: biz.DingTalkQRCodeScanned}, nil
		case "11019":
			attempt.finished = true
			p.removeQRAttempt(state, attempt)
			return &biz.DingTalkQRCodePollResult{Status: biz.DingTalkQRCodeExpired}, nil
		default:
			attempt.finished = true
			p.removeQRAttempt(state, attempt)
			return nil, biz.ErrDingTalkLoginFailed
		}
	}
	attempt.finished = true
	p.removeQRAttempt(state, attempt)
	var confirmed struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(result.Result, &confirmed) != nil {
		return nil, biz.ErrDingTalkLoginFailed
	}
	authCode, err := p.validatedQRAuthCode(confirmed.URL, state)
	if err != nil {
		return nil, err
	}
	return &biz.DingTalkQRCodePollResult{Status: biz.DingTalkQRCodeConfirmed, AuthCode: authCode}, nil
}

func (p *dingTalkIdentityProvider) removeQRAttempt(state string, attempt *dingTalkQRAttempt) {
	p.qrMu.Lock()
	if p.qrAttempts[state] == attempt {
		delete(p.qrAttempts, state)
	}
	p.qrMu.Unlock()
}

func (a *dingTalkQRAttempt) post(ctx context.Context, path string, values url.Values) (*dingTalkQRResponse, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, dingTalkLoginOrigin+path, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, biz.ErrDingTalkLoginFailed
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("sec_umidToken", "FAKE_UMID")
	request.Header.Set("Origin", dingTalkLoginOrigin)
	request.Header.Set("Referer", a.referer)
	request.Header.Set("User-Agent", "Mozilla/5.0")
	response, err := a.client.Do(request)
	if err != nil {
		return nil, dingTalkQRError(ctx)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, biz.ErrDingTalkLoginFailed
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, dingTalkQRBodyLimit+1))
	if err != nil || len(body) > dingTalkQRBodyLimit {
		return nil, biz.ErrDingTalkLoginFailed
	}
	var result dingTalkQRResponse
	if json.Unmarshal(body, &result) != nil {
		return nil, biz.ErrDingTalkLoginFailed
	}
	return &result, nil
}

func (p *dingTalkIdentityProvider) validatedQRAuthCode(rawURL, state string) (string, error) {
	callback, err := url.Parse(p.redirectURI)
	if err != nil {
		return "", biz.ErrDingTalkLoginFailed
	}
	redirect, err := url.Parse(rawURL)
	if err != nil || redirect.Scheme != callback.Scheme || redirect.Host != callback.Host || redirect.Path != callback.Path || redirect.User != nil || redirect.Fragment != "" {
		return "", biz.ErrDingTalkLoginFailed
	}
	values, err := url.ParseQuery(redirect.RawQuery)
	if err != nil || len(values["state"]) != 1 || subtle.ConstantTimeCompare([]byte(values.Get("state")), []byte(state)) != 1 || len(values["authCode"]) != 1 || strings.TrimSpace(values.Get("authCode")) == "" || len(values.Get("authCode")) > 4096 {
		return "", biz.ErrDingTalkLoginFailed
	}
	return values.Get("authCode"), nil
}

func dingTalkQRError(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return biz.ErrDingTalkLoginFailed
}
