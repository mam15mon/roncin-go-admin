package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestEnterprisePartnerRelationChangesPreservesExistingDefaultLink(t *testing.T) {
	keptPartnerID := uuid.New()
	removedPartnerID := uuid.New()
	addedPartnerID := uuid.New()
	keptLinkID := uuid.New()
	removedLinkID := uuid.New()
	existing := []*ent.EnterpriseResourcePartner{
		{ID: keptLinkID, PartnerID: keptPartnerID, IsDefault: true},
		{ID: removedLinkID, PartnerID: removedPartnerID},
	}

	removedIDs, addedIDs := enterprisePartnerRelationChanges(existing, []uuid.UUID{keptPartnerID, addedPartnerID})
	if len(removedIDs) != 1 || removedIDs[0] != removedLinkID {
		t.Fatalf("移除关联计算错误: %v", removedIDs)
	}
	if len(addedIDs) != 1 || addedIDs[0] != addedPartnerID {
		t.Fatalf("新增关联计算错误: %v", addedIDs)
	}
	for _, id := range removedIDs {
		if id == keptLinkID {
			t.Fatal("保留的默认关联不应被删除重建")
		}
	}
}
