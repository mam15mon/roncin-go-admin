package data

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type approvalTokenStub struct{}

func (approvalTokenStub) Enabled() bool { return true }
func (approvalTokenStub) enterpriseAccessToken(context.Context) (string, error) {
	return "test-token", nil
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
			_, err := gateway.Create(context.Background(), &biz.DingTalkApprovalCreateCommand{ProcessCode: "PROC", ApplicantUserID: "applicant", ApproverUserIDs: []string{"approver"}, OrderNo: "SE-001"})
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
		body := make([]byte, request.ContentLength)
		_, _ = request.Body.Read(body)
		payload := string(body)
		for _, required := range []string{`"processCode":"PROC"`, `"originatorUserId":"applicant"`, `"actionType":"OR"`, `"操作票号"`, `"SE-001"`} {
			if !strings.Contains(payload, required) {
				t.Fatalf("payload missing %s: %s", required, payload)
			}
		}
		for _, forbidden := range []string{"accessToken", "clientSecret", "candidateSnapshot", "organizationId"} {
			if strings.Contains(payload, forbidden) {
				t.Fatalf("payload unexpectedly contains %s", forbidden)
			}
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"instanceId":"instance-1"}`))
	}))
	defer server.Close()
	gateway := &dingTalkApprovalGateway{tokens: approvalTokenStub{}, client: server.Client(), endpoint: server.URL}
	result, err := gateway.Create(context.Background(), &biz.DingTalkApprovalCreateCommand{ProcessCode: "PROC", ApplicantUserID: "applicant", ApproverUserIDs: []string{"approver"}, OrderNo: "SE-001"})
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
