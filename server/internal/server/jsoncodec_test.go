package server

import (
	"testing"

	"github.com/go-kratos/kratos/v3/encoding"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	partnerv1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"go.einride.tech/aip/fieldbehavior"
	"google.golang.org/protobuf/proto"
)

func TestProtoJSONCodecAcceptsContractFieldNames(t *testing.T) {
	codec := encoding.GetCodec("json")
	if codec == nil || codec.Name() != "json" {
		t.Fatalf("JSON 编解码器未注册: %v", codec)
	}

	request := &partnerv1.CreatePartnerRequest{}
	payload := []byte(`{"code":"CUST001","legalName":"示例客户有限公司"}`)
	if err := codec.Unmarshal(payload, request); err != nil {
		t.Fatalf("按契约 camelCase 字段解码失败: %v", err)
	}
	if request.GetLegalName() != "示例客户有限公司" {
		t.Fatalf("camelCase 字段未解码到目标值: %q", request.GetLegalName())
	}
}

func TestProtoJSONCodecKeepsProtoFieldNames(t *testing.T) {
	codec := encoding.GetCodec("json")
	if codec == nil {
		t.Fatal("JSON 编解码器未注册")
	}

	request := &partnerv1.CreatePartnerRequest{}
	payload := []byte(`{"code":"CUST001","legal_name":"示例客户有限公司"}`)
	if err := codec.Unmarshal(payload, request); err != nil {
		t.Fatalf("按原始 snake_case 字段解码失败: %v", err)
	}
	if request.GetLegalName() != "示例客户有限公司" {
		t.Fatalf("snake_case 字段未解码到目标值: %q", request.GetLegalName())
	}
}

func TestProtoJSONCodecDiscardsUnknownFields(t *testing.T) {
	codec := encoding.GetCodec("json")
	if codec == nil {
		t.Fatal("JSON 编解码器未注册")
	}

	request := &partnerv1.CreatePartnerRequest{}
	payload := []byte(`{"code":"CUST001","legalName":"示例客户有限公司","unexpected":"value"}`)
	if err := codec.Unmarshal(payload, request); err != nil {
		t.Fatalf("未知字段应被忽略而不是报错: %v", err)
	}
}

func TestProtoJSONCodecFallsBackToEncodingJSON(t *testing.T) {
	codec := encoding.GetCodec("json")
	if codec == nil {
		t.Fatal("JSON 编解码器未注册")
	}

	var target struct {
		Name string `json:"name"`
	}
	if err := codec.Unmarshal([]byte(`{"name":"roncin"}`), &target); err != nil {
		t.Fatalf("非 Proto 消息解码失败: %v", err)
	}
	if target.Name != "roncin" {
		t.Fatalf("非 Proto 消息字段未解码: %q", target.Name)
	}
}

func TestProtoJSONCodecKeepsExplicitZeroExpectedVersionPresence(t *testing.T) {
	codec := encoding.GetCodec("json")
	if codec == nil {
		t.Fatal("JSON 编解码器未注册")
	}

	tests := []struct {
		name    string
		payload []byte
		request proto.Message
	}{
		{
			name:    "汇率继承设置",
			payload: []byte(`{"inheritBaseCurrencyRate":true,"expectedVersion":"0"}`),
			request: &financev1.UpdateExchangeRateCustomSettingRequest{},
		},
		{
			name:    "账单费用修改策略",
			payload: []byte(`{"enabled":true,"editableFields":[],"expectedVersion":"0"}`),
			request: &financev1.UpdateBilledFeeEditPolicyRequest{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := codec.Unmarshal(test.payload, test.request); err != nil {
				t.Fatalf("首次保存请求解码失败: %v", err)
			}
			if err := fieldbehavior.ValidateRequiredFields(test.request); err != nil {
				t.Fatalf("明确提交版本 0 的首次保存请求不应被通用校验拒绝: %v", err)
			}
			switch request := test.request.(type) {
			case *financev1.UpdateExchangeRateCustomSettingRequest:
				if request.ExpectedVersion == nil || request.GetExpectedVersion().GetValue() != 0 {
					t.Fatalf("expected_version=0 的存在性丢失: %#v", request.ExpectedVersion)
				}
			case *financev1.UpdateBilledFeeEditPolicyRequest:
				if request.ExpectedVersion == nil || request.GetExpectedVersion().GetValue() != 0 {
					t.Fatalf("expected_version=0 的存在性丢失: %#v", request.ExpectedVersion)
				}
			default:
				t.Fatalf("未覆盖的请求类型 %T", request)
			}
		})
	}
}
