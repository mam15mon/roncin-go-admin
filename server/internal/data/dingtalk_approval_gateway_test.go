package data

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type approvalTokenStub struct{}

func (approvalTokenStub) Enabled() bool { return true }
func (approvalTokenStub) enterpriseAccessToken(context.Context) (string, error) {
	return "test-token", nil
}

func testApprovalCreateCommand() *biz.DingTalkApprovalCreateCommand {
	return &biz.DingTalkApprovalCreateCommand{
		ProcessCode:          "PROC",
		ApplicantUserID:      "applicant",
		ApproverUserIDs:      []string{"approver"},
		BusinessType:         biz.OrderBusinessAI,
		OrderNo:              "AI-001",
		ApplicantDisplayName: "申请人甲",
		LockGeneration:       2,
	}
}

func TestDingTalkApprovalGatewayCreateFailureClassification(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantKind biz.DingTalkApprovalDispatchFailureKind
	}{
		{name: "明确参数拒绝", status: http.StatusBadRequest, body: `{"code":"InvalidParameter","message":"bad"}`, wantKind: biz.DingTalkApprovalDispatchFailureRejected},
		{name: "明确限流未受理", status: http.StatusTooManyRequests, body: `{"code":"Throttling","message":"slow"}`, wantKind: biz.DingTalkApprovalDispatchFailureRetryable},
		{name: "服务端错误结果未知", status: http.StatusInternalServerError, body: `{"code":"InternalError"}`, wantKind: biz.DingTalkApprovalDispatchFailureUnknown},
		{name: "成功响应解析失败结果未知", status: http.StatusOK, body: `{`, wantKind: biz.DingTalkApprovalDispatchFailureUnknown},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				if request.Header.Get("x-acs-dingtalk-access-token") != "test-token" {
					t.Fatal("未使用钉钉企业令牌请求头")
				}
				response.WriteHeader(test.status)
				_, _ = response.Write([]byte(test.body))
			}))
			defer server.Close()
			gateway := &dingTalkApprovalGateway{tokens: approvalTokenStub{}, client: server.Client(), endpoint: server.URL}
			_, err := gateway.Create(context.Background(), testApprovalCreateCommand())
			var classified *biz.DingTalkApprovalDispatchError
			if !errors.As(err, &classified) || classified.Kind != test.wantKind {
				t.Fatalf("failure = %#v, want kind %s", err, test.wantKind)
			}
			if strings.Contains(classified.Message, "bad") || strings.Contains(classified.Message, "slow") {
				t.Fatalf("持久化错误不得包含钉钉返回的原始 message: %q", classified.Message)
			}
		})
	}
}

func TestDingTalkApprovalGatewayCreateUsesMinimalNativePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Fatalf("method = %s", request.Method)
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatalf("读取审批请求失败: %v", err)
		}
		payload := string(body)
		for _, required := range []string{`"processCode":"PROC"`, `"originatorUserId":"applicant"`, `"actionType":"OR"`, `"业务类型"`, `"空运进口（AI）"`, `"操作票号"`, `"AI-001"`, `"申请人"`, `"申请人甲"`, `"锁定代次"`, `"2"`} {
			if !strings.Contains(payload, required) {
				t.Fatalf("payload missing %s: %s", required, payload)
			}
		}
		for _, forbidden := range []string{"accessToken", "clientSecret", "candidateSnapshot", "organizationId"} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("payload unexpectedly contains %s", forbidden)
			}
		}
		var decoded dingTalkStartProcessRequest
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("解析审批请求失败: %v", err)
		}
		wantForms := []dingTalkStartProcessFormComponent{
			{Name: "业务类型", Value: "空运进口（AI）"},
			{Name: "操作票号", Value: "AI-001"},
			{Name: "申请人", Value: "申请人甲"},
			{Name: "锁定代次", Value: "2"},
			{Name: "解锁原因", Value: "修订单证"},
		}
		if len(decoded.FormComponentValues) != len(wantForms) {
			t.Fatalf("表单字段 = %#v，期望 %#v", decoded.FormComponentValues, wantForms)
		}
		for i := range wantForms {
			if decoded.FormComponentValues[i] != wantForms[i] {
				t.Fatalf("第 %d 个表单字段 = %#v，期望 %#v", i, decoded.FormComponentValues[i], wantForms[i])
			}
		}
		if len(decoded.Approvers) != 1 || decoded.Approvers[0].ActionType != "OR" {
			t.Fatalf("审批节点 = %#v，期望单个 OR 节点", decoded.Approvers)
		}
		wantApprovers := []string{"approver", "approver-2"}
		if !slices.Equal(decoded.Approvers[0].UserIDs, wantApprovers) {
			t.Fatalf("审批候选人 = %v，期望去重后 %v", decoded.Approvers[0].UserIDs, wantApprovers)
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"instanceId":"instance-1"}`))
	}))
	defer server.Close()
	gateway := &dingTalkApprovalGateway{tokens: approvalTokenStub{}, client: server.Client(), endpoint: server.URL}
	command := testApprovalCreateCommand()
	command.ApproverUserIDs = []string{"approver", "approver-2", "approver"}
	reason := "修订单证"
	command.Reason = &reason
	result, err := gateway.Create(context.Background(), command)
	if err != nil || result.ProcessInstanceID != "instance-1" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestDingTalkApprovalGatewayQueryAuthoritativeResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("processInstanceId") != "instance-1" {
			t.Fatalf("processInstanceId = %q", request.URL.Query().Get("processInstanceId"))
		}
		_, _ = response.Write([]byte(`{"result":{"status":"COMPLETED","result":"agree","finishTime":1788537600000,"operationRecords":[{"userId":"approver","result":"AGREE","date":1788537600000}]}}`))
	}))
	defer server.Close()
	gateway := &dingTalkApprovalGateway{tokens: approvalTokenStub{}, client: server.Client(), endpoint: server.URL}
	result, err := gateway.Query(context.Background(), "instance-1")
	if err != nil || result.Decision != biz.DingTalkApprovalDecisionApproved || result.ApproverUserID != "approver" || result.DecidedAt == nil {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestDingTalkSafeDialFailureIsOnlyProvablyUnsentNetworkClass(t *testing.T) {
	dialFailure := &url.Error{Err: &net.OpError{Op: "dial", Err: errors.New("refused")}}
	if !isDingTalkSafeDialFailure(dialFailure) {
		t.Fatal("dial failure should be safely retryable")
	}
	timeout := &url.Error{Err: &net.OpError{Op: "read", Err: errors.New("timeout")}}
	if isDingTalkSafeDialFailure(timeout) {
		t.Fatal("read/timeout failure must remain unknown")
	}
}
