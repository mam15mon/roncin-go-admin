package data

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- 钉钉回调协议固定使用 SHA-1 签名。
	"crypto/subtle"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

type dingTalkApprovalCallbackCodec struct {
	enabled  bool
	token    string
	aesKey   []byte
	ownerKey string
}

func NewDingTalkApprovalCallbackCodec(security *conf.Security) (biz.DingTalkApprovalCallbackCodec, error) {
	codec := &dingTalkApprovalCallbackCodec{}
	if security == nil || security.Dingtalk == nil || !security.Dingtalk.Enabled {
		return codec, nil
	}
	config := security.Dingtalk
	codec.token = strings.TrimSpace(config.EventToken)
	codec.ownerKey = strings.TrimSpace(config.CorpId)
	aesKey := strings.TrimSpace(config.EventAesKey)
	// 钉钉登录/机器人启用不代表订单解锁审批已经完成配置。三个回调字段均为空时
	// 保持回调关闭，让角色直解和其他钉钉能力仍可启动；普通申请会在创建本地事实时
	// 明确进入 CONFIGURATION_FAILED。只配置一部分则视为部署错误并阻止启动。
	if codec.token == "" && aesKey == "" {
		return codec, nil
	}
	if codec.token == "" || codec.ownerKey == "" || aesKey == "" {
		return nil, fmt.Errorf("钉钉审批回调已启用，但 event_token、event_aes_key 或 corp_id 未配置")
	}
	decoded, err := base64.StdEncoding.DecodeString(aesKey + "=")
	if err != nil || len(decoded) != 32 {
		return nil, fmt.Errorf("钉钉 event_aes_key 必须是 43 位有效密钥")
	}
	codec.aesKey = decoded
	codec.enabled = true
	return codec, nil
}

func (c *dingTalkApprovalCallbackCodec) Enabled() bool { return c.enabled }

type dingTalkCallbackPayload struct {
	EventID           string `json:"EventId"`
	EventIDLower      string `json:"eventId"`
	EventType         string `json:"EventType"`
	EventTypeLower    string `json:"eventType"`
	CorpID            string `json:"CorpId"`
	CorpIDLower       string `json:"corpId"`
	ProcessInstanceID string `json:"processInstanceId"`
}

func (c *dingTalkApprovalCallbackCodec) Decode(signature, timestamp, nonce, encrypted string) (*biz.DingTalkApprovalCallbackEvent, error) {
	if !c.enabled {
		return nil, fmt.Errorf("钉钉审批回调未启用")
	}
	encrypted = strings.TrimSpace(encrypted)
	if !verifyDingTalkCallbackSignature(c.token, timestamp, nonce, encrypted, signature) {
		return nil, fmt.Errorf("钉钉审批回调签名无效")
	}
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil || len(ciphertext) == 0 || len(ciphertext)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("钉钉审批回调密文无效")
	}
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return nil, fmt.Errorf("初始化钉钉审批回调解密器: %w", err)
	}
	plain := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, c.aesKey[:aes.BlockSize]).CryptBlocks(plain, ciphertext)
	plain, err = unpadDingTalkCallback(plain)
	if err != nil || len(plain) < 20 {
		return nil, fmt.Errorf("钉钉审批回调明文无效")
	}
	messageLength := int(binary.BigEndian.Uint32(plain[16:20]))
	if messageLength < 0 || 20+messageLength > len(plain) {
		return nil, fmt.Errorf("钉钉审批回调消息长度无效")
	}
	message := plain[20 : 20+messageLength]
	ownerKey := string(plain[20+messageLength:])
	if subtle.ConstantTimeCompare([]byte(ownerKey), []byte(c.ownerKey)) != 1 {
		return nil, fmt.Errorf("钉钉审批回调企业不匹配")
	}
	var payload dingTalkCallbackPayload
	if err := json.Unmarshal(message, &payload); err != nil {
		return nil, fmt.Errorf("解析钉钉审批回调: %w", err)
	}
	eventType := firstApprovalValue(payload.EventType, payload.EventTypeLower)
	corpID := firstApprovalValue(payload.CorpID, payload.CorpIDLower)
	if corpID != "" && subtle.ConstantTimeCompare([]byte(corpID), []byte(c.ownerKey)) != 1 {
		return nil, fmt.Errorf("钉钉审批回调 corpId 不匹配")
	}
	if eventType != "bpms_instance_change" && eventType != "check_url" {
		return nil, fmt.Errorf("不支持的钉钉审批回调事件")
	}
	processInstanceID := strings.TrimSpace(payload.ProcessInstanceID)
	if eventType == "bpms_instance_change" && processInstanceID == "" {
		return nil, fmt.Errorf("钉钉审批回调缺少实例 ID")
	}
	payloadHash := digestBytes([]byte(encrypted))
	eventID := firstApprovalValue(payload.EventID, payload.EventIDLower)
	if eventID == "" {
		eventID = payloadHash
	}
	return &biz.DingTalkApprovalCallbackEvent{
		EventID:              eventID,
		CorpID:               c.ownerKey,
		EventType:            eventType,
		ProcessInstanceID:    processInstanceID,
		EncryptedPayloadHash: payloadHash,
	}, nil
}

func (c *dingTalkApprovalCallbackCodec) EncodeSuccess(timestamp, nonce string) (string, string, error) {
	if !c.enabled {
		return "", "", fmt.Errorf("钉钉审批回调未启用")
	}
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", "", fmt.Errorf("生成钉钉回调响应随机数: %w", err)
	}
	message := []byte("success")
	plain := make([]byte, 0, 20+len(message)+len(c.ownerKey))
	plain = append(plain, randomBytes...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(message)))
	plain = append(plain, length...)
	plain = append(plain, message...)
	plain = append(plain, c.ownerKey...)
	plain = padDingTalkCallback(plain)
	block, err := aes.NewCipher(c.aesKey)
	if err != nil {
		return "", "", err
	}
	ciphertext := make([]byte, len(plain))
	cipher.NewCBCEncrypter(block, c.aesKey[:aes.BlockSize]).CryptBlocks(ciphertext, plain)
	encrypted := base64.StdEncoding.EncodeToString(ciphertext)
	return encrypted, signDingTalkCallback(c.token, timestamp, nonce, encrypted), nil
}

func signDingTalkCallback(token, timestamp, nonce, encrypted string) string {
	values := []string{token, timestamp, nonce, encrypted}
	sort.Strings(values)
	sum := sha1.Sum([]byte(strings.Join(values, ""))) // #nosec G401 -- 钉钉回调协议固定使用 SHA-1。
	return hex.EncodeToString(sum[:])
}

func verifyDingTalkCallbackSignature(token, timestamp, nonce, encrypted, signature string) bool {
	expected := signDingTalkCallback(token, timestamp, nonce, encrypted)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(strings.TrimSpace(signature)))) == 1
}

func padDingTalkCallback(value []byte) []byte {
	const blockSize = 32
	padding := blockSize - len(value)%blockSize
	return append(value, bytes.Repeat([]byte{byte(padding)}, padding)...)
}

func unpadDingTalkCallback(value []byte) ([]byte, error) {
	if len(value) == 0 {
		return nil, fmt.Errorf("空明文")
	}
	padding := int(value[len(value)-1])
	if padding < 1 || padding > 32 || padding > len(value) {
		return nil, fmt.Errorf("填充无效")
	}
	for _, current := range value[len(value)-padding:] {
		if int(current) != padding {
			return nil, fmt.Errorf("填充无效")
		}
	}
	return value[:len(value)-padding], nil
}

func firstApprovalValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

var _ biz.DingTalkApprovalCallbackCodec = (*dingTalkApprovalCallbackCodec)(nil)
