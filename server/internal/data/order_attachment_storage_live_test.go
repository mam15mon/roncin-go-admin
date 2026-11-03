package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

// TestOrderAttachmentOSSLive 只在显式开启时向专用 OSS Bucket 写入随机测试文件并清理。
// 禁止把签名地址、SDK 原始错误或环境凭据输出到测试日志。
func TestOrderAttachmentOSSLive(t *testing.T) {
	if os.Getenv("RONCIN_OSS_LIVE_TEST") != "1" {
		t.Skip("未开启真实 OSS 验证")
	}
	storage, err := NewOrderAttachmentStorage(&conf.Data{OrderAttachmentOss: &conf.Data_OrderAttachmentOSS{Enabled: true, Endpoint: os.Getenv("ORDER_ATTACHMENT_OSS_ENDPOINT"), Region: os.Getenv("ORDER_ATTACHMENT_OSS_REGION"), Bucket: os.Getenv("ORDER_ATTACHMENT_OSS_BUCKET"), AccessKey: os.Getenv("ORDER_ATTACHMENT_OSS_ACCESS_KEY"), SecretKey: os.Getenv("ORDER_ATTACHMENT_OSS_SECRET_KEY")}})
	if err != nil {
		t.Fatal("真实 OSS 配置无效")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	org, actor, order := uuid.New(), uuid.New(), uuid.New()
	payload := []byte("Roncin OSS integration " + uuid.NewString())
	sum := sha256.Sum256(payload)
	upload, err := storage.PrepareUpload(ctx, org, actor, order, &biz.OrderAttachment{DocType: "TEST", FileName: "OSS测试.txt", MIMEType: "text/plain", FileSize: int64(len(payload)), Checksum: hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal("真实 OSS 准备上传失败")
	}
	grant, err := storage.(*orderAttachmentStorage).decodeGrant(upload.UploadToken)
	if err != nil {
		t.Fatal("真实 OSS 凭证解析失败")
	}
	t.Cleanup(func() {
		if err := storage.Delete(context.Background(), org, grant.Attachment.ObjectKey); err != nil {
			t.Error("真实 OSS 清理测试对象失败")
		}
	})
	put := func() int {
		request, err := http.NewRequestWithContext(ctx, http.MethodPut, upload.UploadURL, bytes.NewReader(payload))
		if err != nil {
			t.Fatal("创建直传请求失败")
		}
		for k, v := range upload.Headers {
			request.Header.Set(k, v)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal("真实 OSS 直传失败")
		}
		defer response.Body.Close()
		return response.StatusCode
	}
	if code := put(); code != http.StatusOK {
		t.Fatalf("真实 OSS PUT 状态=%d", code)
	}
	if code := put(); code != http.StatusConflict {
		t.Fatalf("真实 OSS 覆盖防护状态=%d", code)
	}
	attachment, err := storage.VerifyUpload(ctx, org, actor, order, upload.UploadToken)
	if err != nil {
		t.Fatal("真实 OSS 内容核验失败")
	}
	access, err := storage.Access(ctx, org, attachment, true)
	if err != nil {
		t.Fatal("真实 OSS 获取下载授权失败")
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, access.URL, nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal("真实 OSS GET 失败")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, int64(len(payload)+1)))
	if err != nil || response.StatusCode != http.StatusOK || !bytes.Equal(body, payload) {
		t.Fatal("真实 OSS 下载内容不一致")
	}
}
