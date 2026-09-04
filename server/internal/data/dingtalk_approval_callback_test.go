package data

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

func TestDingTalkApprovalCallbackVerifyDecryptAndCorpValidation(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	encodedKey := strings.TrimSuffix(base64.StdEncoding.EncodeToString(key), "=")
	codecInterface, err := NewDingTalkApprovalCallbackCodec(&conf.Security{Dingtalk: &conf.Security_DingTalk{
		Enabled:     true,
		CorpId:      "corp-test",
		EventToken:  "token-test",
		EventAesKey: encodedKey,
	}})
	if err != nil {
		t.Fatalf("create codec: %v", err)
	}
	codec := codecInterface.(*dingTalkApprovalCallbackCodec)
	timestamp, nonce := "1788537600", "nonce-test"
	encrypted := encryptApprovalCallbackForTest(t, codec, map[string]string{
		"EventId":           "event-1",
		"EventType":         "bpms_instance_change",
		"corpId":            "corp-test",
		"processInstanceId": "instance-1",
	}, "corp-test")
	signature := signDingTalkCallback(codec.token, timestamp, nonce, encrypted)
	event, err := codec.Decode(signature, timestamp, nonce, encrypted)
	if err != nil || event.EventID != "event-1" || event.ProcessInstanceID != "instance-1" || len(event.EncryptedPayloadHash) != 64 {
		t.Fatalf("event=%#v err=%v", event, err)
	}
	if _, err := codec.Decode("bad-signature", timestamp, nonce, encrypted); err == nil {
		t.Fatal("wrong signature must be rejected")
	}

	wrongCorp := encryptApprovalCallbackForTest(t, codec, map[string]string{
		"EventType":         "bpms_instance_change",
		"corpId":            "another-corp",
		"processInstanceId": "instance-1",
	}, "corp-test")
	if _, err := codec.Decode(signDingTalkCallback(codec.token, timestamp, nonce, wrongCorp), timestamp, nonce, wrongCorp); err == nil {
		t.Fatal("payload corpId mismatch must be rejected")
	}

	wrongOwner := encryptApprovalCallbackForTest(t, codec, map[string]string{
		"EventType":         "bpms_instance_change",
		"corpId":            "corp-test",
		"processInstanceId": "instance-1",
	}, "another-corp")
	if _, err := codec.Decode(signDingTalkCallback(codec.token, timestamp, nonce, wrongOwner), timestamp, nonce, wrongOwner); err == nil {
		t.Fatal("encrypted owner key mismatch must be rejected")
	}
}

func TestDingTalkApprovalCallbackEncryptedSuccessResponse(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	codecInterface, err := NewDingTalkApprovalCallbackCodec(&conf.Security{Dingtalk: &conf.Security_DingTalk{
		Enabled:     true,
		CorpId:      "corp-test",
		EventToken:  "token-test",
		EventAesKey: strings.TrimSuffix(base64.StdEncoding.EncodeToString(key), "="),
	}})
	if err != nil {
		t.Fatal(err)
	}
	codec := codecInterface.(*dingTalkApprovalCallbackCodec)
	encrypted, signature, err := codec.EncodeSuccess("1788537600", "nonce-test")
	if err != nil || !verifyDingTalkCallbackSignature(codec.token, "1788537600", "nonce-test", encrypted, signature) {
		t.Fatalf("encrypted=%q signature=%q err=%v", encrypted, signature, err)
	}
}

func TestDingTalkApprovalCallbackCanRemainDisabledWhenOnlyDingTalkLoginIsEnabled(t *testing.T) {
	codec, err := NewDingTalkApprovalCallbackCodec(&conf.Security{Dingtalk: &conf.Security_DingTalk{
		Enabled: true,
		CorpId:  "corp-test",
	}})
	if err != nil {
		t.Fatalf("未配置审批回调不应阻止钉钉登录或角色直解启动: %v", err)
	}
	if codec.Enabled() {
		t.Fatal("审批回调字段为空时不应注册为启用")
	}

	if _, err := NewDingTalkApprovalCallbackCodec(&conf.Security{Dingtalk: &conf.Security_DingTalk{
		Enabled:    true,
		CorpId:     "corp-test",
		EventToken: "partial-token",
	}}); err == nil {
		t.Fatal("审批回调只配置一部分时必须显式报错")
	}
}

func encryptApprovalCallbackForTest(t *testing.T, codec *dingTalkApprovalCallbackCodec, payload map[string]string, ownerKey string) string {
	t.Helper()
	message, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	plain := make([]byte, 16)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(message)))
	plain = append(plain, length...)
	plain = append(plain, message...)
	plain = append(plain, ownerKey...)
	plain = padDingTalkCallback(plain)
	block, err := aes.NewCipher(codec.aesKey)
	if err != nil {
		t.Fatal(err)
	}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, codec.aesKey[:aes.BlockSize]).CryptBlocks(ciphertext, plain)
	return base64.StdEncoding.EncodeToString(ciphertext)
}
