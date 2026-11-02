package data

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/fxamacker/cbor/v2"
	wa "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	audit "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	pk "github.com/roncin/roncin-go-admin/server/internal/data/ent/passkeycredential"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

const passkeyFixtureOrigin = "https://ops.roncin.com.cn"

type passkeyVerificationFixture struct {
	base      *authOrganizationSwitchFixture
	repo      *passkeyRepo
	usecase   *biz.PasskeyUsecase
	auth      *biz.AuthUsecase
	principal *biz.Principal
	caller    biz.PasskeyCaller
}

func newPasskeyVerificationFixture(t *testing.T) *passkeyVerificationFixture {
	t.Helper()
	base := newAuthOrganizationSwitchFixture(t)
	source, err := NewPasskeyRepo(base.data, passkeyTestSettings())
	if err != nil {
		t.Fatal(err)
	}
	auth := biz.NewAuthUsecase(base.repo, &biz.SessionPolicy{TTL: 30 * 24 * time.Hour}, nil, nil)
	return &passkeyVerificationFixture{base: base, repo: source.(*passkeyRepo), auth: auth, usecase: biz.NewPasskeyUsecase(source, auth, base.data), principal: &biz.Principal{UserID: base.userID, Organization: biz.Organization{ID: base.branch}}, caller: biz.PasskeyCaller{Host: "ops.roncin.com.cn", Origin: passkeyFixtureOrigin}}
}

type passkeySigningFixture struct {
	key    *ecdsa.PrivateKey
	id     []byte
	handle []byte
}

func newPasskeySigner(t *testing.T, handle uuid.UUID) *passkeySigningFixture {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	id := make([]byte, 32)
	if _, err = rand.Read(id); err != nil {
		t.Fatal(err)
	}
	return &passkeySigningFixture{key: key, id: id, handle: append([]byte(nil), handle[:]...)}
}
func passkeyOptionChallenge(t *testing.T, options string) string {
	t.Helper()
	var data struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal([]byte(options), &data); err != nil || data.Challenge == "" {
		t.Fatalf("仪式选项缺少challenge: %v", err)
	}
	return data.Challenge
}
func passkeyB64(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }
func passkeyJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func (s *passkeySigningFixture) registration(t *testing.T, ceremony *biz.PasskeyCeremony, origin string, flags byte) string {
	t.Helper()
	publicKey, err := cbor.Marshal(map[int]any{1: 2, 3: -7, -1: 1, -2: s.key.X.FillBytes(make([]byte, 32)), -3: s.key.Y.FillBytes(make([]byte, 32))})
	if err != nil {
		t.Fatal(err)
	}
	rpHash := sha256.Sum256([]byte("roncin.com.cn"))
	authData := append([]byte(nil), rpHash[:]...)
	authData = append(authData, flags|0x40)
	authData = append(authData, 0, 0, 0, 0)
	authData = append(authData, make([]byte, 16)...)
	authData = binary.BigEndian.AppendUint16(authData, uint16(len(s.id)))
	authData = append(authData, s.id...)
	authData = append(authData, publicKey...)
	attestation, err := cbor.Marshal(map[string]any{"fmt": "none", "authData": authData, "attStmt": map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	client := passkeyJSON(t, map[string]any{"type": "webauthn.create", "challenge": passkeyOptionChallenge(t, ceremony.OptionsJSON), "origin": origin})
	return string(passkeyJSON(t, map[string]any{"id": passkeyB64(s.id), "rawId": passkeyB64(s.id), "type": "public-key", "response": map[string]any{"attestationObject": passkeyB64(attestation), "clientDataJSON": passkeyB64(client), "transports": []string{"internal"}}, "clientExtensionResults": map[string]any{}}))
}

type passkeyAssertionSettings struct {
	origin, rpID, challenge string
	flags                   byte
	counter                 uint32
	handle                  []byte
	badSignature            bool
}

func (s *passkeySigningFixture) assertion(t *testing.T, ceremony *biz.PasskeyCeremony, settings passkeyAssertionSettings) string {
	t.Helper()
	if settings.origin == "" {
		settings.origin = passkeyFixtureOrigin
	}
	if settings.rpID == "" {
		settings.rpID = "roncin.com.cn"
	}
	if settings.challenge == "" {
		settings.challenge = passkeyOptionChallenge(t, ceremony.OptionsJSON)
	}
	if settings.handle == nil {
		settings.handle = s.handle
	}
	if settings.flags == 0 {
		settings.flags = 0x05
	}
	client := passkeyJSON(t, map[string]any{"type": "webauthn.get", "challenge": settings.challenge, "origin": settings.origin})
	rpHash := sha256.Sum256([]byte(settings.rpID))
	authData := append([]byte(nil), rpHash[:]...)
	authData = append(authData, settings.flags)
	authData = binary.BigEndian.AppendUint32(authData, settings.counter)
	clientHash := sha256.Sum256(client)
	signed := append(append([]byte(nil), authData...), clientHash[:]...)
	hash := sha256.Sum256(signed)
	signature, err := ecdsa.SignASN1(rand.Reader, s.key, hash[:])
	if err != nil {
		t.Fatal(err)
	}
	if settings.badSignature {
		signature[len(signature)-1] ^= 0xff
	}
	return string(passkeyJSON(t, map[string]any{"id": passkeyB64(s.id), "rawId": passkeyB64(s.id), "type": "public-key", "response": map[string]any{"authenticatorData": passkeyB64(authData), "clientDataJSON": passkeyB64(client), "signature": passkeyB64(signature), "userHandle": passkeyB64(settings.handle)}, "clientExtensionResults": map[string]any{}}))
}
func (f *passkeyVerificationFixture) register(t *testing.T) (*passkeySigningFixture, *biz.PasskeyCredentialRecord) {
	t.Helper()
	signer := newPasskeySigner(t, f.base.userID)
	ceremony, err := f.usecase.BeginRegistration(t.Context(), f.caller, f.principal)
	if err != nil {
		t.Fatal(err)
	}
	var options struct {
		AuthenticatorSelection struct {
			ResidentKey, UserVerification string
			RequireResidentKey            bool
		} `json:"authenticatorSelection"`
		PublicKey any `json:"publicKey"`
	}
	if err = json.Unmarshal([]byte(ceremony.OptionsJSON), &options); err != nil || options.AuthenticatorSelection.ResidentKey != "required" || options.AuthenticatorSelection.UserVerification != "required" || !options.AuthenticatorSelection.RequireResidentKey || options.PublicKey != nil {
		t.Fatalf("注册必须要求可发现凭证及本地用户验证: %+v %v", options, err)
	}
	record, err := f.usecase.FinishRegistration(t.Context(), f.caller, f.principal, ceremony.ID, signer.registration(t, ceremony, passkeyFixtureOrigin, 0x05), "测试通行密钥")
	if err != nil {
		t.Fatal(err)
	}
	return signer, record
}
func (f *passkeyVerificationFixture) storedCredential(t *testing.T, id uuid.UUID) wa.Credential {
	t.Helper()
	row, err := f.base.data.db.PasskeyCredential.Get(t.Context(), id)
	if err != nil {
		t.Fatal(err)
	}
	var credential wa.Credential
	if err = json.Unmarshal(row.CredentialJSON, &credential); err != nil {
		t.Fatal(err)
	}
	return credential
}

func TestPasskeyRegistrationAndLoginRealSignaturesPostgres(t *testing.T) {
	f := newPasskeyVerificationFixture(t)
	signer, record := f.register(t)
	ceremony, err := f.usecase.BeginLogin(t.Context(), f.caller)
	if err != nil {
		t.Fatal(err)
	}
	response := signer.assertion(t, ceremony, passkeyAssertionSettings{counter: 1})
	result, err := f.usecase.FinishLogin(t.Context(), f.caller, ceremony.ID, response, "test-agent")
	if err != nil {
		t.Fatal(err)
	}
	principal, err := f.auth.AuthenticateSession(t.Context(), result.Token)
	if err != nil || principal.UserID != f.base.userID {
		t.Fatalf("验签后未建立真实会话: %v", err)
	}
	stored := f.storedCredential(t, record.ID)
	if stored.Authenticator.SignCount != 1 || stored.Authenticator.CloneWarning {
		t.Fatal("签名计数未正确保存")
	}
	row, err := f.base.data.db.PasskeyCredential.Get(t.Context(), record.ID)
	if err != nil || row.LastUsedAt == nil {
		t.Fatal("未记录最近使用时间")
	}
	count, err := f.base.data.db.AuditLog.Query().Where(audit.ActionEQ("auth.passkey.login")).Count(t.Context())
	if err != nil || count != 1 {
		t.Fatal("登录审计未与会话提交")
	}
	if _, err = f.usecase.FinishLogin(t.Context(), f.caller, ceremony.ID, response, "test-agent"); err != biz.ErrPasskeyInvalid {
		t.Fatal("成功challenge可被重放")
	}
	records, err := f.usecase.List(t.Context(), f.caller, f.principal)
	if err != nil || len(records) != 1 || records[0].ID != record.ID || records[0].Name != "测试通行密钥" {
		t.Fatal("本人凭证列表错误")
	}
}

func TestPasskeyRejectsCryptographicAndIdentityFailuresPostgres(t *testing.T) {
	f := newPasskeyVerificationFixture(t)
	signer, record := f.register(t)
	otherID := uuid.New()
	if _, err := f.base.data.db.User.Create().SetID(otherID).SetUsername("pk-other-" + f.base.suffix).SetDisplayName("另一个用户").Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name     string
		settings passkeyAssertionSettings
	}{
		{"错误origin", passkeyAssertionSettings{origin: "https://evil.example.com", counter: 1}},
		{"错误RPID", passkeyAssertionSettings{rpID: "other.roncin.com.cn", counter: 1}},
		{"错误challenge", passkeyAssertionSettings{challenge: passkeyB64([]byte("another-unrelated-challenge-value")), counter: 1}},
		{"错误签名", passkeyAssertionSettings{badSignature: true, counter: 1}},
		{"跨用户handle", passkeyAssertionSettings{handle: otherID[:], counter: 1}},
		{"缺少UV", passkeyAssertionSettings{flags: 0x01, counter: 1}},
		{"缺少UP", passkeyAssertionSettings{flags: 0x04, counter: 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ceremony, err := f.usecase.BeginLogin(t.Context(), f.caller)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.usecase.FinishLogin(t.Context(), f.caller, ceremony.ID, signer.assertion(t, ceremony, tc.settings), "agent"); err == nil {
				t.Fatal("无效断言建立了会话")
			}
			if _, err = f.repo.takeCeremony(ceremony.ID, "login", uuid.Nil); err != biz.ErrPasskeyInvalid {
				t.Fatal("失败challenge仍可重放")
			}
			if got := f.storedCredential(t, record.ID).Authenticator.SignCount; got != 0 {
				t.Fatal("失败验签改写了凭证")
			}
			if count, err := f.base.data.db.Session.Query().Count(t.Context()); err != nil || count != 0 {
				t.Fatal("失败验签仍建立会话")
			}
		})
	}
}

func TestPasskeyRegistrationOwnershipAndDeletionPostgres(t *testing.T) {
	f := newPasskeyVerificationFixture(t)
	signer, record := f.register(t)
	otherID := uuid.New()
	if _, err := f.base.data.db.User.Create().SetID(otherID).SetUsername("pk-second-" + f.base.suffix).SetDisplayName("另一用户").Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	other := &biz.Principal{UserID: otherID, Organization: f.principal.Organization}
	ceremony, err := f.usecase.BeginRegistration(t.Context(), f.caller, f.principal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.usecase.FinishRegistration(t.Context(), f.caller, other, ceremony.ID, signer.registration(t, ceremony, passkeyFixtureOrigin, 0x05), "跨用户"); err != biz.ErrPasskeyInvalid {
		t.Fatal("注册challenge可跨用户使用")
	}
	if err = f.usecase.Delete(t.Context(), f.caller, other, record.ID); err != biz.ErrPasskeyNotFound {
		t.Fatal("可删除其他用户的凭证")
	}
	login, err := f.usecase.BeginLogin(t.Context(), f.caller)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.usecase.Delete(t.Context(), f.caller, f.principal, record.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = f.usecase.FinishLogin(t.Context(), f.caller, login.ID, signer.assertion(t, login, passkeyAssertionSettings{counter: 1}), "agent"); err == nil {
		t.Fatal("已删除凭证仍可登录")
	}
	if count, err := f.base.data.db.PasskeyCredential.Query().Count(t.Context()); err != nil || count != 0 {
		t.Fatal("本人凭证未删除")
	}
}

func TestPasskeyAccountDisabledAndAtomicSessionRollbackPostgres(t *testing.T) {
	f := newPasskeyVerificationFixture(t)
	signer, record := f.register(t)
	ceremony, err := f.usecase.BeginLogin(t.Context(), f.caller)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.base.data.db.User.UpdateOneID(f.base.userID).SetEnabled(false).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.usecase.FinishLogin(t.Context(), f.caller, ceremony.ID, signer.assertion(t, ceremony, passkeyAssertionSettings{counter: 1}), "agent"); err != biz.ErrInvalidCredentials {
		t.Fatal("停用用户仍可登录")
	}
	if err = f.base.data.db.User.UpdateOneID(f.base.userID).SetEnabled(true).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	ceremony, err = f.usecase.BeginLogin(t.Context(), f.caller)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.base.data.db.Membership.Update().Where(membership.UserIDEQ(f.base.userID)).SetEnabled(false).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.usecase.FinishLogin(t.Context(), f.caller, ceremony.ID, signer.assertion(t, ceremony, passkeyAssertionSettings{counter: 1}), "agent"); err != biz.ErrOrganizationForbidden {
		t.Fatalf("无工作台资格未拒绝: %v", err)
	}
	if stored := f.storedCredential(t, record.ID); stored.Authenticator.SignCount != 0 {
		t.Fatal("建会话失败没有回滚签名计数")
	}
	if count, err := f.base.data.db.Session.Query().Count(t.Context()); err != nil || count != 0 {
		t.Fatal("失败事务建立了会话")
	}
}

func TestPasskeyConcurrentCounterUpdatesAreSerializedPostgres(t *testing.T) {
	f := newPasskeyVerificationFixture(t)
	signer, record := f.register(t)
	first, err := f.usecase.BeginLogin(t.Context(), f.caller)
	if err != nil {
		t.Fatal(err)
	}
	second, err := f.usecase.BeginLogin(t.Context(), f.caller)
	if err != nil {
		t.Fatal(err)
	}
	responses := []string{signer.assertion(t, first, passkeyAssertionSettings{counter: 1}), signer.assertion(t, second, passkeyAssertionSettings{counter: 1})}
	start := make(chan struct{})
	results := make(chan error, 2)
	for i, ceremony := range []*biz.PasskeyCeremony{first, second} {
		go func(ceremony *biz.PasskeyCeremony, response string) {
			<-start
			_, err := f.usecase.FinishLogin(t.Context(), f.caller, ceremony.ID, response, "agent")
			results <- err
		}(ceremony, responses[i])
	}
	close(start)
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		} else if !errors.Is(err, biz.ErrPasskeyInvalid) {
			t.Fatalf("并发登录非预期失败: %v", err)
		}
	}
	if successes != 1 {
		t.Fatalf("相同非零签名计数并发成功%d次", successes)
	}
	if got := f.storedCredential(t, record.ID).Authenticator.SignCount; got != 1 {
		t.Fatal("并发签名计数没有串行持久化")
	}
	if count, err := f.base.data.db.Session.Query().Count(t.Context()); err != nil || count != 1 {
		t.Fatal("并发计数失败仍建立会话")
	}
}

func TestPasskeyRegistrationRejectsOriginAndUVPostgres(t *testing.T) {
	f := newPasskeyVerificationFixture(t)
	for _, tc := range []struct {
		name, origin string
		flags        byte
	}{{"错误来源", "https://evil.example.com", 0x05}, {"缺少UV", passkeyFixtureOrigin, 0x01}, {"缺少UP", passkeyFixtureOrigin, 0x04}} {
		t.Run(tc.name, func(t *testing.T) {
			signer := newPasskeySigner(t, f.base.userID)
			ceremony, err := f.usecase.BeginRegistration(t.Context(), f.caller, f.principal)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = f.usecase.FinishRegistration(t.Context(), f.caller, f.principal, ceremony.ID, signer.registration(t, ceremony, tc.origin, tc.flags), "未通过验证"); err == nil {
				t.Fatal("未通过验证的注册写入数据库")
			}
		})
	}
	if count, err := f.base.data.db.PasskeyCredential.Query().Count(t.Context()); err != nil || count != 0 {
		t.Fatal("错误注册留下凭证")
	}
	if count, err := f.base.data.db.AuditLog.Query().Where(audit.ActionEQ("auth.passkey.register")).Count(t.Context()); err != nil || count != 0 {
		t.Fatal("错误注册写入成功审计")
	}
}

func TestPasskeyDuplicateCredentialsCannotTransferOwnerPostgres(t *testing.T) {
	f := newPasskeyVerificationFixture(t)
	signer, record := f.register(t)
	ceremony, err := f.usecase.BeginRegistration(t.Context(), f.caller, f.principal)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.usecase.FinishRegistration(t.Context(), f.caller, f.principal, ceremony.ID, signer.registration(t, ceremony, passkeyFixtureOrigin, 0x05), "重复凭证"); err != biz.ErrPasskeyExists {
		t.Fatalf("唯一索引未兜底: %v", err)
	}
	row, err := f.base.data.db.PasskeyCredential.Query().Where(pk.IDEQ(record.ID), pk.UserIDEQ(f.base.userID)).Only(t.Context())
	if err != nil || row.Name != "测试通行密钥" {
		t.Fatal("重复注册覆盖了原凭证")
	}
}

func TestPasskeyPendingAccountRejectsLoginPostgres(t *testing.T) {
	f := newPasskeyVerificationFixture(t)
	signer, _ := f.register(t)
	ceremony, err := f.usecase.BeginLogin(t.Context(), f.caller)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.base.data.db.User.Update().Where(user.IDEQ(f.base.userID)).SetEnabled(false).SetPendingAssignment(true).Exec(t.Context()); err != nil {
		t.Fatal(err)
	}
	if _, err = f.usecase.FinishLogin(t.Context(), f.caller, ceremony.ID, signer.assertion(t, ceremony, passkeyAssertionSettings{counter: 1}), "agent"); err != biz.ErrInvalidCredentials {
		t.Fatal("待分配账号可通过通行密钥登录")
	}
}
