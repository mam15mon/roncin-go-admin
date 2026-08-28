package data

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

const (
	dingTalkRegistrationTokenSecretMinLength = 32
	dingTalkRegistrationTokenMaxLength       = 3800
)

var dingTalkRegistrationTokenAdditionalData = []byte("roncin:dingtalk-registration:v1")

type dingTalkRegistrationTokenCodec struct {
	aead   cipher.AEAD
	corpID string
}

type dingTalkRegistrationTokenPayload struct {
	UnionID   string `json:"u"`
	UserID    string `json:"i"`
	CorpID    string `json:"c"`
	Name      string `json:"n"`
	Email     string `json:"e,omitempty"`
	AvatarURL string `json:"a,omitempty"`
	ExpiresAt int64  `json:"x"`
}

func NewDingTalkRegistrationTokenCodec(security *conf.Security) (*dingTalkRegistrationTokenCodec, error) {
	codec := &dingTalkRegistrationTokenCodec{}
	if security == nil || security.Dingtalk == nil || !security.Dingtalk.Enabled {
		return codec, nil
	}
	secret := strings.TrimSpace(security.Dingtalk.RegistrationTokenSecret)
	if len(secret) < dingTalkRegistrationTokenSecretMinLength {
		return nil, fmt.Errorf("钉钉认证已启用，但 registration_token_secret 少于 32 个字符")
	}
	digest := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(digest[:])
	if err != nil {
		return nil, fmt.Errorf("初始化钉钉注册凭证加密器: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("初始化钉钉注册凭证认证加密器: %w", err)
	}
	codec.aead = aead
	codec.corpID = strings.TrimSpace(security.Dingtalk.CorpId)
	return codec, nil
}

func (c *dingTalkRegistrationTokenCodec) Seal(identity *biz.DingTalkIdentity, expiresAt time.Time) (string, error) {
	if c.aead == nil {
		return "", biz.ErrDingTalkDisabled
	}
	if identity == nil || strings.TrimSpace(identity.UnionID) == "" || strings.TrimSpace(identity.UserID) == "" || strings.TrimSpace(identity.CorpID) == "" || strings.TrimSpace(identity.Name) == "" || !expiresAt.After(time.Now()) {
		return "", biz.ErrDingTalkLoginFailed
	}
	payload := dingTalkRegistrationTokenPayload{
		UnionID:   strings.TrimSpace(identity.UnionID),
		UserID:    strings.TrimSpace(identity.UserID),
		CorpID:    strings.TrimSpace(identity.CorpID),
		Name:      strings.TrimSpace(identity.Name),
		ExpiresAt: expiresAt.Unix(),
	}
	if identity.Email != nil {
		payload.Email = strings.TrimSpace(*identity.Email)
	}
	if identity.AvatarURL != nil {
		payload.AvatarURL = strings.TrimSpace(*identity.AvatarURL)
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("编码钉钉注册凭证: %w", err)
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("生成钉钉注册凭证随机数: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, plaintext, dingTalkRegistrationTokenAdditionalData)
	token := base64.RawURLEncoding.EncodeToString(sealed)
	if len(token) > dingTalkRegistrationTokenMaxLength {
		return "", fmt.Errorf("钉钉注册凭证超过 Cookie 长度限制")
	}
	return token, nil
}

func (c *dingTalkRegistrationTokenCodec) Open(token string, now time.Time) (*biz.DingTalkIdentity, error) {
	if c.aead == nil || strings.TrimSpace(token) == "" {
		return nil, biz.ErrDingTalkRegistrationExpired
	}
	sealed, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(token))
	if err != nil || len(sealed) <= c.aead.NonceSize() {
		return nil, biz.ErrDingTalkRegistrationExpired
	}
	nonce, ciphertext := sealed[:c.aead.NonceSize()], sealed[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, dingTalkRegistrationTokenAdditionalData)
	if err != nil {
		return nil, biz.ErrDingTalkRegistrationExpired
	}
	var payload dingTalkRegistrationTokenPayload
	if err := json.Unmarshal(plaintext, &payload); err != nil {
		return nil, biz.ErrDingTalkRegistrationExpired
	}
	if payload.ExpiresAt <= now.Unix() || strings.TrimSpace(payload.CorpID) != c.corpID || strings.TrimSpace(payload.UnionID) == "" || strings.TrimSpace(payload.UserID) == "" || strings.TrimSpace(payload.Name) == "" {
		return nil, biz.ErrDingTalkRegistrationExpired
	}
	identity := &biz.DingTalkIdentity{
		UnionID: strings.TrimSpace(payload.UnionID),
		UserID:  strings.TrimSpace(payload.UserID),
		CorpID:  strings.TrimSpace(payload.CorpID),
		Name:    strings.TrimSpace(payload.Name),
	}
	if email := strings.TrimSpace(payload.Email); email != "" {
		identity.Email = &email
	}
	if avatarURL := strings.TrimSpace(payload.AvatarURL); avatarURL != "" {
		identity.AvatarURL = &avatarURL
	}
	return identity, nil
}

var _ biz.DingTalkRegistrationTokenCodec = (*dingTalkRegistrationTokenCodec)(nil)
