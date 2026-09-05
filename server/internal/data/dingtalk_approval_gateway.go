package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

const dingTalkWorkflowProcessInstancesURL = "https://api.dingtalk.com/v1.0/workflow/processInstances"

type dingTalkEnterpriseTokenProvider interface {
	Enabled() bool
	enterpriseAccessToken(context.Context) (string, error)
}

type dingTalkApprovalGateway struct {
	tokens   dingTalkEnterpriseTokenProvider
	client   *http.Client
	endpoint string
}

func NewDingTalkApprovalGateway(provider *dingTalkIdentityProvider) biz.DingTalkApprovalGateway {
	return &dingTalkApprovalGateway{
		tokens:   provider,
		client:   &http.Client{Timeout: 15 * time.Second},
		endpoint: dingTalkWorkflowProcessInstancesURL,
	}
}

type dingTalkStartProcessRequest struct {
	ProcessCode         string                              `json:"processCode"`
	OriginatorUserID    string                              `json:"originatorUserId"`
	Approvers           []dingTalkStartProcessApprover      `json:"approvers"`
	FormComponentValues []dingTalkStartProcessFormComponent `json:"formComponentValues"`
}

type dingTalkStartProcessApprover struct {
	ActionType string   `json:"actionType"`
	UserIDs    []string `json:"userIds"`
}

type dingTalkStartProcessFormComponent struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type dingTalkStartProcessResponse struct {
	InstanceID string `json:"instanceId"`
}

type dingTalkAPIErrorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestid"`
}

type dingTalkGetProcessResponse struct {
	Result *dingTalkGetProcessResult `json:"result"`
}

type dingTalkGetProcessResult struct {
	Status           string                     `json:"status"`
	Result           string                     `json:"result"`
	FinishTime       json.RawMessage            `json:"finishTime"`
	OperationRecords []dingTalkProcessOperation `json:"operationRecords"`
}

type dingTalkProcessOperation struct {
	UserID string          `json:"userId"`
	Result string          `json:"result"`
	Date   json.RawMessage `json:"date"`
}

func (g *dingTalkApprovalGateway) Create(ctx context.Context, command *biz.DingTalkApprovalCreateCommand) (*biz.DingTalkApprovalCreateResult, error) {
	if !g.tokens.Enabled() {
		return nil, &biz.DingTalkApprovalDispatchError{Kind: biz.DingTalkApprovalDispatchFailureRejected, Code: "DINGTALK_APPROVAL_DISABLED", Message: "钉钉审批未启用"}
	}
	if command == nil || strings.TrimSpace(command.ProcessCode) == "" || strings.TrimSpace(command.ApplicantUserID) == "" ||
		!command.BusinessType.Valid() || strings.TrimSpace(command.OrderNo) == "" || strings.TrimSpace(command.ApplicantDisplayName) == "" ||
		command.LockGeneration == 0 || len(command.ApproverUserIDs) == 0 {
		return nil, &biz.DingTalkApprovalDispatchError{
			Kind:    biz.DingTalkApprovalDispatchFailureRejected,
			Code:    "DINGTALK_APPROVAL_ARGUMENT_INVALID",
			Message: "钉钉审批派发参数不完整",
		}
	}
	approvers := make([]string, 0, len(command.ApproverUserIDs))
	seen := make(map[string]struct{}, len(command.ApproverUserIDs))
	for _, raw := range command.ApproverUserIDs {
		userID := strings.TrimSpace(raw)
		if userID == "" {
			return nil, &biz.DingTalkApprovalDispatchError{
				Kind:    biz.DingTalkApprovalDispatchFailureRejected,
				Code:    "DINGTALK_APPROVAL_ARGUMENT_INVALID",
				Message: "钉钉审批候选人配置不完整",
			}
		}
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		approvers = append(approvers, userID)
	}

	formValues := []dingTalkStartProcessFormComponent{
		{Name: "业务类型", Value: command.BusinessType.DisplayName()},
		{Name: "操作票号", Value: strings.TrimSpace(command.OrderNo)},
		{Name: "申请人", Value: strings.TrimSpace(command.ApplicantDisplayName)},
		{Name: "锁定代次", Value: strconv.FormatUint(command.LockGeneration, 10)},
	}
	if command.Reason != nil && strings.TrimSpace(*command.Reason) != "" {
		formValues = append(formValues, dingTalkStartProcessFormComponent{Name: "解锁原因", Value: strings.TrimSpace(*command.Reason)})
	}
	payload := dingTalkStartProcessRequest{
		ProcessCode:      strings.TrimSpace(command.ProcessCode),
		OriginatorUserID: strings.TrimSpace(command.ApplicantUserID),
		Approvers: []dingTalkStartProcessApprover{{
			ActionType: "OR",
			UserIDs:    approvers,
		}},
		FormComponentValues: formValues,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, &biz.DingTalkApprovalDispatchError{Kind: biz.DingTalkApprovalDispatchFailureRetryable, Code: "DINGTALK_APPROVAL_ENCODE_FAILED", Message: "钉钉审批请求编码失败", Cause: err}
	}
	if err := ctx.Err(); err != nil {
		return nil, &biz.DingTalkApprovalDispatchError{Kind: biz.DingTalkApprovalDispatchFailureRetryable, Code: "DINGTALK_APPROVAL_CONTEXT_CANCELED", Message: "钉钉审批请求尚未发送", Cause: err}
	}
	accessToken, err := g.tokens.enterpriseAccessToken(ctx)
	if err != nil {
		return nil, &biz.DingTalkApprovalDispatchError{Kind: biz.DingTalkApprovalDispatchFailureRetryable, Code: "DINGTALK_ACCESS_TOKEN_FAILED", Message: "获取钉钉企业访问令牌失败", Cause: err}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, g.endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, &biz.DingTalkApprovalDispatchError{Kind: biz.DingTalkApprovalDispatchFailureRetryable, Code: "DINGTALK_APPROVAL_REQUEST_INVALID", Message: "钉钉审批请求未发送", Cause: err}
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("x-acs-dingtalk-access-token", accessToken)

	response, err := g.client.Do(request)
	if err != nil {
		// net/http 允许 RoundTripper 在返回错误的同时返回响应；此时调用方仍负责关闭响应体。
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		kind := biz.DingTalkApprovalDispatchFailureUnknown
		code := "DINGTALK_APPROVAL_CREATE_UNKNOWN"
		message := "钉钉审批发起结果未知，请人工核对"
		if isDingTalkSafeDialFailure(err) {
			kind = biz.DingTalkApprovalDispatchFailureRetryable
			code = "DINGTALK_APPROVAL_NOT_SENT"
			message = "连接钉钉失败，审批请求尚未送达"
		}
		return nil, &biz.DingTalkApprovalDispatchError{Kind: kind, Code: code, Message: message, Cause: err}
	}
	defer response.Body.Close()
	body, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	digest := digestBytes(body)
	if readErr != nil {
		return nil, &biz.DingTalkApprovalDispatchError{Kind: biz.DingTalkApprovalDispatchFailureUnknown, Code: "DINGTALK_APPROVAL_RESPONSE_UNKNOWN", Message: "钉钉审批发起结果未知，请人工核对", ResponseDigest: digest, Cause: readErr}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var apiErr dingTalkAPIErrorResponse
		_ = json.Unmarshal(body, &apiErr)
		code := strings.TrimSpace(apiErr.Code)
		if code == "" {
			code = fmt.Sprintf("DINGTALK_HTTP_%d", response.StatusCode)
		}
		// 钉钉 message 可能回显表单值或租户信息，不持久化第三方原文；诊断依赖 code 与响应摘要。
		message := fmt.Sprintf("钉钉审批接口明确拒绝请求（HTTP %d）", response.StatusCode)
		kind := biz.DingTalkApprovalDispatchFailureRejected
		if response.StatusCode == http.StatusTooManyRequests {
			kind = biz.DingTalkApprovalDispatchFailureRetryable
			message = "钉钉审批接口限流，请稍后重试"
		} else if response.StatusCode >= http.StatusInternalServerError || response.StatusCode == http.StatusRequestTimeout {
			kind = biz.DingTalkApprovalDispatchFailureUnknown
			message = "钉钉审批发起结果未知，请人工核对"
		}
		return nil, &biz.DingTalkApprovalDispatchError{Kind: kind, Code: code, Message: message, ResponseDigest: digest}
	}
	var decoded dingTalkStartProcessResponse
	if err := json.Unmarshal(body, &decoded); err != nil || strings.TrimSpace(decoded.InstanceID) == "" {
		return nil, &biz.DingTalkApprovalDispatchError{Kind: biz.DingTalkApprovalDispatchFailureUnknown, Code: "DINGTALK_APPROVAL_RESPONSE_INCOMPLETE", Message: "钉钉审批发起结果未知，请人工核对", ResponseDigest: digest, Cause: err}
	}
	return &biz.DingTalkApprovalCreateResult{ProcessInstanceID: strings.TrimSpace(decoded.InstanceID), ResponseDigest: digest}, nil
}

func (g *dingTalkApprovalGateway) Query(ctx context.Context, processInstanceID string) (*biz.DingTalkApprovalQueryResult, error) {
	if !g.tokens.Enabled() {
		return nil, fmt.Errorf("钉钉审批未启用")
	}
	processInstanceID = strings.TrimSpace(processInstanceID)
	if processInstanceID == "" {
		return nil, fmt.Errorf("钉钉审批实例 ID 不能为空")
	}
	accessToken, err := g.tokens.enterpriseAccessToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取钉钉企业访问令牌: %w", err)
	}
	endpoint, err := url.Parse(g.endpoint)
	if err != nil {
		return nil, fmt.Errorf("解析钉钉审批查询地址: %w", err)
	}
	query := endpoint.Query()
	query.Set("processInstanceId", processInstanceID)
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("创建钉钉审批查询请求: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("x-acs-dingtalk-access-token", accessToken)
	response, err := g.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("查询钉钉审批实例: %w", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取钉钉审批查询响应: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var apiErr dingTalkAPIErrorResponse
		_ = json.Unmarshal(body, &apiErr)
		return nil, fmt.Errorf("钉钉审批查询失败（HTTP %d，code=%s）", response.StatusCode, trimRunes(apiErr.Code, 64))
	}
	var decoded dingTalkGetProcessResponse
	if err := json.Unmarshal(body, &decoded); err != nil || decoded.Result == nil {
		return nil, fmt.Errorf("解析钉钉审批查询响应: %w", err)
	}
	status := strings.ToUpper(strings.TrimSpace(decoded.Result.Status))
	decision := strings.ToUpper(strings.TrimSpace(decoded.Result.Result))
	result := &biz.DingTalkApprovalQueryResult{Decision: biz.DingTalkApprovalDecisionPending}
	if status != "COMPLETED" {
		return result, nil
	}
	switch decision {
	case "AGREE", "APPROVED":
		result.Decision = biz.DingTalkApprovalDecisionApproved
		for _, operation := range decoded.Result.OperationRecords {
			if strings.EqualFold(strings.TrimSpace(operation.Result), "AGREE") || strings.EqualFold(strings.TrimSpace(operation.Result), "APPROVED") {
				if strings.TrimSpace(operation.UserID) != "" {
					result.ApproverUserID = strings.TrimSpace(operation.UserID)
					result.DecidedAt = parseDingTalkTime(operation.Date)
				}
			}
		}
		if result.DecidedAt == nil {
			result.DecidedAt = parseDingTalkTime(decoded.Result.FinishTime)
		}
		if result.ApproverUserID == "" {
			return nil, fmt.Errorf("钉钉审批通过结果缺少审批人")
		}
	case "REFUSE", "REJECTED":
		result.Decision = biz.DingTalkApprovalDecisionRejected
		for _, operation := range decoded.Result.OperationRecords {
			if strings.EqualFold(strings.TrimSpace(operation.Result), "REFUSE") || strings.EqualFold(strings.TrimSpace(operation.Result), "REJECTED") {
				result.ApproverUserID = strings.TrimSpace(operation.UserID)
				result.DecidedAt = parseDingTalkTime(operation.Date)
			}
		}
		if result.DecidedAt == nil {
			result.DecidedAt = parseDingTalkTime(decoded.Result.FinishTime)
		}
	default:
		return nil, fmt.Errorf("钉钉审批已完成但结果无法识别")
	}
	return result, nil
}

func parseDingTalkTime(raw json.RawMessage) *time.Time {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var milliseconds int64
	if err := json.Unmarshal(raw, &milliseconds); err == nil && milliseconds > 0 {
		parsed := time.UnixMilli(milliseconds).UTC()
		return &parsed
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil
	}
	value = strings.TrimSpace(value)
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		parsed = parsed.UTC()
		return &parsed
	}
	if parsed, err := time.Parse("2006-01-02 15:04:05", value); err == nil {
		parsed = parsed.UTC()
		return &parsed
	}
	return nil
}

func isDingTalkSafeDialFailure(err error) bool {
	var urlErr *url.Error
	if !errors.As(err, &urlErr) {
		return false
	}
	var opErr *net.OpError
	return errors.As(urlErr.Err, &opErr) && opErr.Op == "dial"
}

func digestBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

var _ biz.DingTalkApprovalGateway = (*dingTalkApprovalGateway)(nil)
