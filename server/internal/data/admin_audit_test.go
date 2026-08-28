package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestAuditTargetUserID(t *testing.T) {
	targetID := uuid.New()
	tests := []struct {
		name string
		log  *biz.AuditLog
		want *uuid.UUID
	}{
		{name: "用户资源字段", log: &biz.AuditLog{Action: "admin.user.update", ResourceID: stringPointer(targetID.String())}, want: &targetID},
		{name: "历史详情字段", log: &biz.AuditLog{Action: "admin.user.dingtalk.authorize", Details: map[string]string{"resource_id": targetID.String()}}, want: &targetID},
		{name: "非用户操作", log: &biz.AuditLog{Action: "auth.login", Details: map[string]string{"resource_id": targetID.String()}}, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := auditTargetUserID(test.log)
			if test.want == nil {
				if got != nil {
					t.Fatalf("expected nil, got %s", got)
				}
				return
			}
			if got == nil || *got != *test.want {
				t.Fatalf("expected %s, got %v", test.want, got)
			}
		})
	}
}

func stringPointer(value string) *string { return &value }
