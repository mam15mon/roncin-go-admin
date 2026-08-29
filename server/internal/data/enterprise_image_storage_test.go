package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestEnterpriseImageExtensionUsesValidatedMIME(t *testing.T) {
	tests := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/bmp":  ".bmp",
		"image/gif":  ".gif",
	}
	for mimeType, expected := range tests {
		if actual := enterpriseImageExtension(mimeType); actual != expected {
			t.Fatalf("MIME %s 的扩展名为 %s，期望 %s", mimeType, actual, expected)
		}
	}
	if actual := enterpriseImageExtension("application/pdf"); actual != "" {
		t.Fatalf("非图片 MIME 不应生成扩展名，实际为 %s", actual)
	}
}

func TestEnterpriseImageStorageRejectsOtherOrganizationObjectKey(t *testing.T) {
	organizationID := uuid.New()
	storage := &enterpriseImageStorage{enabled: true}
	err := storage.VerifyUpload(t.Context(), organizationID, &biz.EnterpriseResourceImage{
		ObjectKey: "enterprise-resources/" + uuid.New().String() + "/image.png",
	})
	if err != biz.ErrEnterpriseResourceInvalidArgument {
		t.Fatalf("其他组织的对象键应被拒绝，实际为 %v", err)
	}
}
