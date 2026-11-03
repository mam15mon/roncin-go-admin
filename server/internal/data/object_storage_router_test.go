package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestObjectStorageDeletionRoutesOnlyKnownOrganizationPrefixes(t *testing.T) {
	org := uuid.New()
	image := &deletionIntegrationStorage{}
	orders := newAttachmentOSSFixture(t)
	router := NewObjectStorageDeleter(image, orders.storage)
	if !router.Enabled() {
		t.Fatal("任一存储开启应启动消费者")
	}
	imageKey := "enterprise-resources/" + org.String() + "/image.jpg"
	if err := router.Delete(context.Background(), org, imageKey); err != nil || image.deleteKey != imageKey {
		t.Fatal("企业图片删除路由错误", err)
	}
	orderKey := "orders/" + org.String() + "/" + uuid.NewString() + "/" + uuid.NewString()
	if err := router.Delete(context.Background(), org, orderKey); err != nil || !orders.deleted {
		t.Fatal("订单附件删除路由错误", err)
	}
	for _, key := range []string{"unknown/file", "orders/" + uuid.NewString() + "/" + uuid.NewString() + "/" + uuid.NewString(), "enterprise-resources/" + uuid.NewString() + "/image.jpg"} {
		if err := router.Delete(context.Background(), org, key); err != biz.ErrOrderAttachmentInvalidArgument {
			t.Fatal("未知或跨组织对象未拒绝", err)
		}
	}
	orders.storage.enabled = false
	if err := router.Delete(context.Background(), org, orderKey); err != biz.ErrOrderAttachmentStorageUnavailable {
		t.Fatal("关闭的订单存储不应回退", err)
	}
}
