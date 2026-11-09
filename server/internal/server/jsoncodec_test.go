package server

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/go-kratos/kratos/v3/encoding"
	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	financev1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	masterdatav1 "github.com/roncin/roncin-go-admin/server/api/masterdata/v1"
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

func TestProtoJSONCodecUsesEnumNumbers(t *testing.T) {
	codec := encoding.GetCodec("json")
	if codec == nil {
		t.Fatal("JSON 编解码器未注册")
	}

	data, err := codec.Marshal(&adminv1.AdminUser{Status: adminv1.AdminUserStatus_ADMIN_USER_STATUS_PENDING_AUTHORIZATION})
	if err != nil {
		t.Fatalf("枚举响应编码失败: %v", err)
	}
	if string(data) != `{"status":2}` {
		t.Fatalf("枚举响应 = %s，期望使用 OpenAPI 数字枚举", data)
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

// 主数据响应的标量零值省略是实际 wire 契约，前端必须按字段 presence 适配。
func TestProtoJSONCodecMasterDataResponseZeroValues(t *testing.T) {
	codec := encoding.GetCodec("json")
	tests := []struct {
		name    string
		message proto.Message
	}{
		{"通用主数据", &masterdatav1.MasterDataItem{Id: "record-1", Source: "seed"}},
		{"港口", &masterdatav1.Port{Id: "record-1", Source: "seed"}},
		{"机场", &masterdatav1.Airport{Id: "record-1", Source: "seed"}},
		{"航司", &masterdatav1.Airline{Id: "record-1", Source: "seed"}},
		{"船司", &masterdatav1.ShippingLine{Id: "record-1", Source: "seed"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, err := codec.Marshal(test.message)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(payload, &fields); err != nil {
				t.Fatal(err)
			}
			for _, key := range []string{"enabled", "sortOrder"} {
				if _, exists := fields[key]; exists {
					t.Fatalf("零值字段 %s 应被省略，实际 %s", key, payload)
				}
			}
			if string(fields["id"]) != `"record-1"` || string(fields["source"]) != `"seed"` {
				t.Fatalf("非零字段丢失：%s", payload)
			}
		})
	}
	payload, err := codec.Marshal(&masterdatav1.MasterDataItem{Enabled: true, SortOrder: 25})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["enabled"]) != "true" || string(fields["sortOrder"]) != "25" {
		t.Fatalf("非零字段未正确输出：%s", payload)
	}
}

func TestProtoJSONCodecMasterDataOptionalPresence(t *testing.T) {
	codec := encoding.GetCodec("json")
	for _, enabled := range []*bool{nil, proto.Bool(false), proto.Bool(true)} {
		request := &masterdatav1.ListPortsRequest{Enabled: enabled}
		payload, err := codec.Marshal(request)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(payload, &fields); err != nil {
			t.Fatal(err)
		}
		value, exists := fields["enabled"]
		if exists != (enabled != nil) {
			t.Fatalf("optional 筛选存在性丢失：%s", payload)
		}
		if enabled != nil && string(value) != fmt.Sprint(*enabled) {
			t.Fatalf("optional 筛选值丢失：%s", payload)
		}
		var decoded masterdatav1.ListPortsRequest
		if err := codec.Unmarshal(payload, &decoded); err != nil {
			t.Fatal(err)
		}
		if (decoded.Enabled == nil) != (enabled == nil) {
			t.Fatalf("请求往返后筛选存在性丢失：%s", payload)
		}
	}
	for _, name := range []*string{nil, proto.String("")} {
		payload, err := codec.Marshal(&masterdatav1.Port{NameZh: name})
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(payload, &fields); err != nil {
			t.Fatal(err)
		}
		value, exists := fields["nameZh"]
		if exists != (name != nil) || (exists && string(value) != `""`) {
			t.Fatalf("optional 响应空字符串存在性丢失：%s", payload)
		}
	}
}
