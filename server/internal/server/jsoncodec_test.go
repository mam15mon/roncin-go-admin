package server

import (
	"testing"

	"github.com/go-kratos/kratos/v3/encoding"
	partnerv1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
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
