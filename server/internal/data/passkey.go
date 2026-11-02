package data

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	wa "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	pk "github.com/roncin/roncin-go-admin/server/internal/data/ent/passkeycredential"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"golang.org/x/net/publicsuffix"
)

const passkeyCeremonyTTL = 5 * time.Minute
const passkeyCeremonyCapacity = 2048

type passkeySession struct {
	purpose string
	userID  uuid.UUID
	data    *wa.SessionData
	expires time.Time
}
type passkeyRepo struct {
	data     *Data
	settings biz.PasskeySettings
	engine   *wa.WebAuthn
	mu       sync.Mutex
	sessions map[string]passkeySession
}

func NewPasskeyRepo(data *Data, security *conf.Security) (biz.PasskeyRepo, error) {
	settings := biz.PasskeySettings{}
	if security != nil && security.Passkey != nil {
		c := security.Passkey
		settings = biz.PasskeySettings{Enabled: c.Enabled, RPID: c.RpId, RPName: c.RpName, Origins: append([]string(nil), c.Origins...)}
	}
	repo := &passkeyRepo{data: data, settings: settings, sessions: make(map[string]passkeySession)}
	if !settings.Enabled {
		return repo, nil
	}
	if err := validatePasskeySettings(settings); err != nil {
		return nil, err
	}
	engine, err := wa.New(&wa.Config{RPID: settings.RPID, RPDisplayName: settings.RPName, RPOrigins: settings.Origins,
		AuthenticatorSelection: protocol.AuthenticatorSelection{ResidentKey: protocol.ResidentKeyRequirementRequired, RequireResidentKey: protocol.ResidentKeyRequired(), UserVerification: protocol.VerificationRequired},
		AttestationPreference:  protocol.PreferNoAttestation,
		Timeouts:               wa.TimeoutsConfig{Login: wa.TimeoutConfig{Enforce: true, Timeout: passkeyCeremonyTTL, TimeoutUVD: passkeyCeremonyTTL}, Registration: wa.TimeoutConfig{Enforce: true, Timeout: passkeyCeremonyTTL, TimeoutUVD: passkeyCeremonyTTL}},
	})
	if err != nil {
		return nil, fmt.Errorf("通行密钥配置无效: %w", err)
	}
	repo.engine = engine
	return repo, nil
}
func validatePasskeySettings(s biz.PasskeySettings) error {
	if s.RPID == "" || s.RPID != strings.ToLower(s.RPID) || net.ParseIP(s.RPID) != nil || protocol.ValidateRPID(s.RPID) != nil {
		return fmt.Errorf("通行密钥 rp_id 必须是有效的小写域名")
	}
	if suffix, _ := publicsuffix.PublicSuffix(s.RPID); suffix == s.RPID && s.RPID != "localhost" {
		return fmt.Errorf("通行密钥 rp_id 不能是公共后缀")
	}
	if strings.TrimSpace(s.RPName) == "" || len(s.Origins) == 0 {
		return fmt.Errorf("通行密钥必须配置 rp_name 和明确 origins")
	}
	seen := map[string]bool{}
	for _, origin := range s.Origins {
		parsed, err := url.Parse(origin)
		if err != nil || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host == "" || origin != parsed.Scheme+"://"+parsed.Host || parsed.Hostname() != strings.ToLower(parsed.Hostname()) || protocol.ValidateRPID(parsed.Hostname()) != nil {
			return fmt.Errorf("通行密钥 origin 必须是无路径的完整来源地址")
		}
		host := parsed.Hostname()
		if net.ParseIP(host) != nil || (host != s.RPID && !strings.HasSuffix(host, "."+s.RPID)) {
			return fmt.Errorf("通行密钥 origin 必须属于配置的 rp_id")
		}
		if parsed.Scheme != "https" && !(parsed.Scheme == "http" && s.RPID == "localhost" && host == "localhost") {
			return fmt.Errorf("通行密钥 origin 必须使用 HTTPS，仅 localhost 允许 HTTP")
		}
		if seen[origin] {
			return fmt.Errorf("通行密钥 origins 不得重复")
		}
		seen[origin] = true
	}
	return nil
}
func (r *passkeyRepo) Settings() biz.PasskeySettings {
	s := r.settings
	s.Origins = append([]string(nil), s.Origins...)
	return s
}
func (r *passkeyRepo) DiscardCeremony(id string) { r.mu.Lock(); delete(r.sessions, id); r.mu.Unlock() }
func (r *passkeyRepo) saveCeremony(purpose string, userID uuid.UUID, session *wa.SessionData, options any) (*biz.PasskeyCeremony, error) {
	body, err := json.Marshal(options)
	if err != nil {
		return nil, biz.ErrPasskeyInvalid
	}
	var token [32]byte
	if _, err = rand.Read(token[:]); err != nil {
		return nil, biz.ErrPasskeyInvalid
	}
	id := base64.RawURLEncoding.EncodeToString(token[:])
	expires := time.Now().Add(passkeyCeremonyTTL)
	if !session.Expires.IsZero() && session.Expires.Before(expires) {
		expires = session.Expires
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for key, item := range r.sessions {
		if !item.expires.After(time.Now()) {
			delete(r.sessions, key)
		}
	}
	if len(r.sessions) >= passkeyCeremonyCapacity {
		return nil, biz.ErrPasskeyBusy
	}
	r.sessions[id] = passkeySession{purpose: purpose, userID: userID, data: session, expires: expires}
	return &biz.PasskeyCeremony{ID: id, OptionsJSON: string(body), ExpiresAt: expires}, nil
}
func (r *passkeyRepo) takeCeremony(id, purpose string, userID uuid.UUID) (*wa.SessionData, error) {
	r.mu.Lock()
	session, ok := r.sessions[id]
	delete(r.sessions, id)
	r.mu.Unlock()
	if !ok || session.purpose != purpose || session.userID != userID || !session.expires.After(time.Now()) {
		return nil, biz.ErrPasskeyInvalid
	}
	return session.data, nil
}
func (r *passkeyRepo) BeginLogin(_ context.Context, origin string) (*biz.PasskeyCeremony, error) {
	if r.engine == nil {
		return nil, biz.ErrPasskeyDisabled
	}
	options, session, err := r.engine.BeginDiscoverableLogin(wa.WithLoginOrigin(origin))
	if err != nil {
		return nil, biz.ErrPasskeyInvalid
	}
	return r.saveCeremony("login", uuid.Nil, session, options.Response)
}
func (r *passkeyRepo) BeginRegistration(ctx context.Context, userID uuid.UUID, origin string) (*biz.PasskeyCeremony, error) {
	if r.engine == nil {
		return nil, biz.ErrPasskeyDisabled
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	account, err := client.User.Query().Where(user.IDEQ(userID), user.EnabledEQ(true), user.PendingAssignmentEQ(false)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrInvalidCredentials, nil)
	}
	stored, err := client.PasskeyCredential.Query().Where(pk.UserIDEQ(userID), pk.RpIDEQ(r.settings.RPID)).All(ctx)
	if err != nil {
		return nil, err
	}
	credentials := make([]wa.Credential, 0, len(stored))
	for _, row := range stored {
		var credential wa.Credential
		if json.Unmarshal(row.CredentialJSON, &credential) != nil {
			return nil, biz.ErrPasskeyInvalid
		}
		credentials = append(credentials, credential)
	}
	subject := &passkeyUser{account: account, credentials: credentials}
	exclusions := make([]protocol.CredentialDescriptor, 0, len(credentials))
	for _, credential := range credentials {
		exclusions = append(exclusions, credential.Descriptor())
	}
	options, session, err := r.engine.BeginRegistration(subject, wa.WithExclusions(exclusions), wa.WithResidentKeyRequirement(protocol.ResidentKeyRequirementRequired), wa.WithRegistrationOrigin(origin))
	if err != nil {
		return nil, biz.ErrPasskeyInvalid
	}
	return r.saveCeremony("registration", userID, session, options.Response)
}
func (r *passkeyRepo) FinishRegistration(ctx context.Context, userID uuid.UUID, id, response, name string, audit *biz.AuditEvent) (record *biz.PasskeyCredentialRecord, err error) {
	if r.engine == nil {
		return nil, biz.ErrPasskeyDisabled
	}
	session, err := r.takeCeremony(id, "registration", userID)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBytes([]byte(response))
	if err != nil {
		return nil, biz.ErrPasskeyInvalid
	}
	err = r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		account, err := client.User.Query().Where(user.IDEQ(userID)).ForShare().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrInvalidCredentials, nil)
		}
		if !account.Enabled || account.PendingAssignment {
			return biz.ErrInvalidCredentials
		}
		credential, err := r.engine.CreateCredential(&passkeyUser{account: account}, *session, parsed)
		if err != nil {
			return biz.ErrPasskeyInvalid
		}
		encoded, err := json.Marshal(credential)
		if err != nil {
			return biz.ErrPasskeyInvalid
		}
		row, err := client.PasskeyCredential.Create().SetUserID(userID).SetRpID(r.settings.RPID).SetCredentialID(base64.RawURLEncoding.EncodeToString(credential.ID)).SetCredentialJSON(encoded).SetName(name).Save(txCtx)
		if err != nil {
			if ent.IsConstraintError(err) {
				return biz.ErrPasskeyExists
			}
			return err
		}
		if err := writeAudit(txCtx, client.AuditLog, audit); err != nil {
			return err
		}
		record = passkeyRecord(row)
		return nil
	})
	return record, err
}
func (r *passkeyRepo) FinishLogin(ctx context.Context, id, response string) (identity *biz.Credential, err error) {
	if r.engine == nil {
		return nil, biz.ErrPasskeyDisabled
	}
	session, err := r.takeCeremony(id, "login", uuid.Nil)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBytes([]byte(response))
	if err != nil {
		return nil, biz.ErrPasskeyInvalid
	}
	userID, err := uuid.FromBytes(parsed.Response.UserHandle)
	if err != nil {
		return nil, biz.ErrPasskeyInvalid
	}
	err = r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		// 所有认证、删除和注册先锁用户，避免与账号停用/成员维护形成反向锁序。
		account, err := client.User.Query().Where(user.IDEQ(userID)).ForShare().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrPasskeyInvalid, nil)
		}
		if !account.Enabled || account.PendingAssignment {
			return biz.ErrInvalidCredentials
		}
		row, err := client.PasskeyCredential.Query().Where(pk.UserIDEQ(userID), pk.RpIDEQ(r.settings.RPID), pk.CredentialIDEQ(base64.RawURLEncoding.EncodeToString(parsed.RawID))).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrPasskeyInvalid, nil)
		}
		var stored wa.Credential
		if json.Unmarshal(row.CredentialJSON, &stored) != nil {
			return biz.ErrPasskeyInvalid
		}
		subject := &passkeyUser{account: account, credentials: []wa.Credential{stored}}
		_, verified, err := r.engine.ValidatePasskeyLogin(func(rawID, handle []byte) (wa.User, error) {
			if !bytes.Equal(rawID, stored.ID) || !bytes.Equal(handle, subject.WebAuthnID()) {
				return nil, biz.ErrPasskeyInvalid
			}
			return subject, nil
		}, *session, parsed)
		if err != nil || verified.Authenticator.CloneWarning {
			return biz.ErrPasskeyInvalid
		}
		encoded, err := json.Marshal(verified)
		if err != nil {
			return biz.ErrPasskeyInvalid
		}
		if _, err = client.PasskeyCredential.UpdateOneID(row.ID).SetCredentialJSON(encoded).SetLastUsedAt(time.Now()).Save(txCtx); err != nil {
			return err
		}
		identity = &biz.Credential{UserID: account.ID, Username: account.Username, DisplayName: account.DisplayName, Enabled: account.Enabled, PendingAssignment: account.PendingAssignment}
		return nil
	})
	return identity, err
}
func (r *passkeyRepo) List(ctx context.Context, userID uuid.UUID) ([]biz.PasskeyCredentialRecord, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := client.PasskeyCredential.Query().Where(pk.UserIDEQ(userID), pk.RpIDEQ(r.settings.RPID)).Order(ent.Asc(pk.FieldCreatedAt), ent.Asc(pk.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	records := make([]biz.PasskeyCredentialRecord, 0, len(rows))
	for _, row := range rows {
		records = append(records, *passkeyRecord(row))
	}
	return records, nil
}
func (r *passkeyRepo) Delete(ctx context.Context, userID, id uuid.UUID, audit *biz.AuditEvent) error {
	return r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, err := r.data.client(txCtx)
		if err != nil {
			return err
		}
		account, err := client.User.Query().Where(user.IDEQ(userID)).ForShare().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrInvalidCredentials, nil)
		}
		if !account.Enabled || account.PendingAssignment {
			return biz.ErrInvalidCredentials
		}
		row, err := client.PasskeyCredential.Query().Where(pk.IDEQ(id), pk.UserIDEQ(userID), pk.RpIDEQ(r.settings.RPID)).ForUpdate().Only(txCtx)
		if err != nil {
			return mapEntError(err, biz.ErrPasskeyNotFound, nil)
		}
		if err = client.PasskeyCredential.DeleteOneID(row.ID).Exec(txCtx); err != nil {
			return err
		}
		return writeAudit(txCtx, client.AuditLog, audit)
	})
}
func passkeyRecord(row *ent.PasskeyCredential) *biz.PasskeyCredentialRecord {
	return &biz.PasskeyCredentialRecord{ID: row.ID, Name: row.Name, CreatedAt: row.CreatedAt, LastUsedAt: row.LastUsedAt}
}

type passkeyUser struct {
	account     *ent.User
	credentials []wa.Credential
}

func (u *passkeyUser) WebAuthnID() []byte { return u.account.ID[:] }
func (u *passkeyUser) WebAuthnName() string {
	if u.account.Username != "" {
		return u.account.Username
	}
	return u.account.DisplayName
}
func (u *passkeyUser) WebAuthnDisplayName() string          { return u.account.DisplayName }
func (u *passkeyUser) WebAuthnCredentials() []wa.Credential { return u.credentials }

var _ biz.PasskeyRepo = (*passkeyRepo)(nil)
