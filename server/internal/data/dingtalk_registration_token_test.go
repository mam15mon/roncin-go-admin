package data

import (
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

func TestDingTalkRegistrationTokenRoundTrip(t *testing.T) {
	codec := newDingTalkRegistrationTokenCodecForTest(t, "ding-corp")
	email := "zhangsan@example.com"
	avatarURL := "https://example.com/avatar.png"
	identity := &biz.DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三", Email: &email, AvatarURL: &avatarURL}
	expiresAt := time.Now().UTC().Add(time.Minute)

	token, err := codec.Seal(identity, expiresAt)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	decoded, err := codec.Open(token, expiresAt.Add(-time.Second))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if decoded.UnionID != identity.UnionID || decoded.UserID != identity.UserID || decoded.CorpID != identity.CorpID || decoded.Name != identity.Name || decoded.Email == nil || *decoded.Email != email || decoded.AvatarURL == nil || *decoded.AvatarURL != avatarURL {
		t.Fatalf("Open() identity = %#v", decoded)
	}
}

func TestDingTalkRegistrationTokenRejectsTamperingAndExpiry(t *testing.T) {
	codec := newDingTalkRegistrationTokenCodecForTest(t, "ding-corp")
	expiresAt := time.Now().UTC().Add(time.Minute)
	token, err := codec.Seal(&biz.DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}, expiresAt)
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}

	tampered := token[:len(token)-1] + map[bool]string{true: "A", false: "B"}[strings.HasSuffix(token, "A")]
	if _, err := codec.Open(tampered, time.Now().UTC()); err != biz.ErrDingTalkRegistrationExpired {
		t.Fatalf("Open(tampered) error = %v", err)
	}
	if _, err := codec.Open(token, expiresAt.Add(time.Second)); err != biz.ErrDingTalkRegistrationExpired {
		t.Fatalf("Open(expired) error = %v", err)
	}
}

func TestDingTalkRegistrationTokenRejectsOtherOrganization(t *testing.T) {
	issuer := newDingTalkRegistrationTokenCodecForTest(t, "ding-corp")
	verifier := newDingTalkRegistrationTokenCodecForTest(t, "other-corp")
	token, err := issuer.Seal(&biz.DingTalkIdentity{UnionID: "union-id", UserID: "user-id", CorpID: "ding-corp", Name: "张三"}, time.Now().UTC().Add(time.Minute))
	if err != nil {
		t.Fatalf("Seal() error = %v", err)
	}
	if _, err := verifier.Open(token, time.Now().UTC()); err != biz.ErrDingTalkRegistrationExpired {
		t.Fatalf("Open(other corp) error = %v", err)
	}
}

func TestNewDingTalkRegistrationTokenCodecRequiresDedicatedSecret(t *testing.T) {
	_, err := NewDingTalkRegistrationTokenCodec(&conf.Security{Dingtalk: &conf.Security_DingTalk{Enabled: true, CorpId: "ding-corp", RegistrationTokenSecret: "short"}})
	if err == nil {
		t.Fatal("启用钉钉认证时必须拒绝过短的注册凭证密钥")
	}
}

func newDingTalkRegistrationTokenCodecForTest(t *testing.T, corpID string) *dingTalkRegistrationTokenCodec {
	t.Helper()
	codec, err := NewDingTalkRegistrationTokenCodec(&conf.Security{Dingtalk: &conf.Security_DingTalk{
		Enabled:                 true,
		CorpId:                  corpID,
		RegistrationTokenSecret: "test-registration-token-secret-32-bytes",
	}})
	if err != nil {
		t.Fatalf("NewDingTalkRegistrationTokenCodec() error = %v", err)
	}
	return codec
}
