package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type authRepoStub struct {
	credential     *Credential
	createdSession *Session
	auditActions   []string
}

func (s *authRepoStub) FindCredential(context.Context, string) (*Credential, error) {
	return nil, ErrInvalidCredentials
}

func (s *authRepoStub) LoginRateLimitExceeded(context.Context, []string, time.Time, time.Duration, int) (bool, error) {
	return false, nil
}

func (s *authRepoStub) RecordLoginFailure(_ context.Context, _ []string, _ time.Time, _ time.Duration, _ int, audit *AuditEvent) (bool, error) {
	s.auditActions = append(s.auditActions, audit.Action)
	return false, nil
}

func (s *authRepoStub) FindDingTalkCredential(context.Context, *DingTalkIdentity) (*Credential, error) {
	if s.credential == nil {
		return nil, ErrDingTalkNotRegistered
	}
	return s.credential, nil
}

func (s *authRepoStub) ResolvePrincipal(_ context.Context, userID, organizationID uuid.UUID) (*Principal, error) {
	return &Principal{UserID: userID, Organization: Organization{ID: organizationID}}, nil
}

func (s *authRepoStub) CreateSession(_ context.Context, session *Session, _ string, audit *AuditEvent) error {
	s.createdSession = session
	s.auditActions = append(s.auditActions, audit.Action)
	return nil
}

func (s *authRepoStub) FindSession(context.Context, string, time.Time) (*Session, error) {
	return nil, ErrSessionExpired
}

func (s *authRepoStub) ListEnabledMembershipOrganizations(context.Context, uuid.UUID) ([]OrganizationChoice, error) {
	return nil, nil
}

func (s *authRepoStub) RotateSession(_ context.Context, _ string, _ *Session, _ time.Time, audit *AuditEvent) error {
	s.auditActions = append(s.auditActions, audit.Action)
	return nil
}

func (s *authRepoStub) RevokeSession(_ context.Context, _ string, _ time.Time, audit *AuditEvent) error {
	s.auditActions = append(s.auditActions, audit.Action)
	return nil
}

var _ AuthRepo = (*authRepoStub)(nil)
