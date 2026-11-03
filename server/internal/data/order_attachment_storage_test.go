package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

type attachmentOSSFixture struct {
	storage    *orderAttachmentStorage
	body       []byte
	headers    http.Header
	versioning string
	deleted    bool
}

func newAttachmentOSSFixture(t *testing.T) *attachmentOSSFixture {
	t.Helper()
	f := &attachmentOSSFixture{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.URL.Query()["versioning"]; ok {
			w.Header().Set("Content-Type", "application/xml")
			io.WriteString(w, "<VersioningConfiguration><Status>"+f.versioning+"</Status></VersioningConfiguration>")
			return
		}
		switch r.Method {
		case http.MethodPut:
			if r.Header.Get("X-Oss-Forbid-Overwrite") != "true" || r.Header.Get("X-Oss-Object-Acl") != "private" {
				t.Error("上传请求缺少禁止覆盖或私有ACL约束")
			}
			if f.body != nil {
				w.WriteHeader(http.StatusConflict)
				return
			}
			f.body, _ = io.ReadAll(r.Body)
			f.headers = r.Header.Clone()
		case http.MethodHead, http.MethodGet:
			if f.body == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			for k, v := range f.headers {
				if strings.HasPrefix(strings.ToLower(k), "x-oss-meta-") {
					w.Header()[k] = v
				}
			}
			w.Header().Set("Content-Type", f.headers.Get("Content-Type"))
			w.Header().Set("Content-Length", stringSize(len(f.body)))
			w.Header().Set("ETag", "\"test-etag\"")
			if r.Method == http.MethodGet {
				w.Write(f.body)
			}
		case http.MethodDelete:
			f.deleted = true
			f.body = nil
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)
	f.storage = &orderAttachmentStorage{enabled: true, bucket: "test-bucket", signingKey: []byte("isolated-test-key"), now: time.Now,
		client: oss.NewClient(oss.LoadDefaultConfig().WithRegion("cn-beijing").WithEndpoint(server.URL).WithUsePathStyle(true).WithRetryMaxAttempts(1).WithCredentialsProvider(credentials.NewStaticCredentialsProvider("test-id", "test-secret")))}
	return f
}

func stringSize(n int) string { return fmt.Sprint(n) }

func uploadFixtureAttachment(t *testing.T, f *attachmentOSSFixture, org, actor, order uuid.UUID, payload []byte) *biz.OrderAttachmentUpload {
	t.Helper()
	sum := sha256.Sum256(payload)
	upload, err := f.storage.PrepareUpload(context.Background(), org, actor, order, &biz.OrderAttachment{DocType: "BL", FileName: "中文提单.pdf", MIMEType: "application/pdf", FileSize: int64(len(payload)), Checksum: hex.EncodeToString(sum[:])})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequest(http.MethodPut, upload.UploadURL, bytes.NewReader(payload))
	if err != nil {
		t.Fatal("创建上传请求失败")
	}
	for k, v := range upload.Headers {
		request.Header.Set(k, v)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal("直传请求失败")
	}
	response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("直传状态=%d", response.StatusCode)
	}
	return upload
}

func TestOrderAttachmentOSSUploadVerifyAccessDelete(t *testing.T) {
	f := newAttachmentOSSFixture(t)
	org, actor, order := uuid.New(), uuid.New(), uuid.New()
	ctx := context.Background()
	upload := uploadFixtureAttachment(t, f, org, actor, order, []byte("actual document"))
	attachment, err := f.storage.VerifyUpload(ctx, org, actor, order, upload.UploadToken)
	if err != nil {
		t.Fatal(err)
	}
	if !validOrderAttachmentKey(org, attachment.ObjectKey) {
		t.Fatal("对象键不属于当前组织")
	}
	attachment.OrderID = uuid.New() // 拆票共享引用在新票读取时保留原资产键。
	access, err := f.storage.Access(ctx, org, attachment, true)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(access.URL)
	if parsed.Query().Has("response-content-type") {
		t.Fatal("OSS禁止覆盖响应Content-Type")
	}
	if !strings.Contains(parsed.Query().Get("response-content-disposition"), "attachment") {
		t.Fatal("下载必须携带原文件名")
	}
	if _, err := f.storage.Access(ctx, uuid.New(), attachment, false); err != biz.ErrOrderAttachmentInvalidArgument {
		t.Fatal("跨组织访问未拒绝")
	}
	request, _ := http.NewRequest(http.MethodPut, upload.UploadURL, bytes.NewReader([]byte("overwrite")))
	for k, v := range upload.Headers {
		request.Header.Set(k, v)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal("覆盖请求失败")
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatal("覆盖应拒绝")
	}
	if err := f.storage.Delete(ctx, org, attachment.ObjectKey); err != nil || !f.deleted {
		t.Fatal("删除未完成")
	}
}

func TestOrderAttachmentOSSRejectsForgedExpiredAndMismatchedUpload(t *testing.T) {
	f := newAttachmentOSSFixture(t)
	org, actor, order := uuid.New(), uuid.New(), uuid.New()
	ctx := context.Background()
	upload := uploadFixtureAttachment(t, f, org, actor, order, []byte("original"))
	for _, args := range [][3]uuid.UUID{{uuid.New(), actor, order}, {org, uuid.New(), order}, {org, actor, uuid.New()}} {
		if _, err := f.storage.VerifyUpload(ctx, args[0], args[1], args[2], upload.UploadToken); err != biz.ErrOrderAttachmentUploadInvalid {
			t.Fatal("跨范围授权未拒绝")
		}
	}
	if _, err := f.storage.VerifyUpload(ctx, org, actor, order, upload.UploadToken+"x"); err != biz.ErrOrderAttachmentUploadInvalid {
		t.Fatal("篡改授权未拒绝")
	}
	f.body = []byte("modified") // 保留同样长度与全部正确 metadata，必须校验真实内容。
	if _, err := f.storage.VerifyUpload(ctx, org, actor, order, upload.UploadToken); err != biz.ErrOrderAttachmentUploadInvalid {
		t.Fatal("只核验metadata而未识别内容篡改")
	}
	f.body = []byte("short")
	if _, err := f.storage.VerifyUpload(ctx, org, actor, order, upload.UploadToken); err != biz.ErrOrderAttachmentUploadInvalid {
		t.Fatal("大小不一致未拒绝")
	}
	f.storage.now = func() time.Time { return time.Now().Add(11 * time.Minute) }
	if _, err := f.storage.VerifyUpload(ctx, org, actor, order, upload.UploadToken); err != biz.ErrOrderAttachmentUploadInvalid {
		t.Fatal("过期授权未拒绝")
	}
}

func TestOrderAttachmentOSSRejectsVersionedBucketAndMissingObject(t *testing.T) {
	f := newAttachmentOSSFixture(t)
	ctx := context.Background()
	org, actor, order := uuid.New(), uuid.New(), uuid.New()
	for _, status := range []string{"Enabled", "Suspended"} {
		f.versioning = status
		if _, err := f.storage.PrepareUpload(ctx, org, actor, order, &biz.OrderAttachment{}); err != biz.ErrOrderAttachmentVersioning {
			t.Fatalf("状态 %s 未拒绝", status)
		}
	}
	f.versioning = ""
	upload := uploadFixtureAttachment(t, f, org, actor, order, []byte("test"))
	f.body = nil
	if _, err := f.storage.VerifyUpload(ctx, org, actor, order, upload.UploadToken); err != biz.ErrOrderAttachmentUploadInvalid {
		t.Fatal("不存在对象未拒绝")
	}
}

func TestOrderAttachmentOSSConfiguration(t *testing.T) {
	disabled, err := NewOrderAttachmentStorage(&conf.Data{})
	if err != nil || disabled.Enabled() {
		t.Fatal("默认应关闭")
	}
	if _, err := disabled.PrepareUpload(context.Background(), uuid.New(), uuid.New(), uuid.New(), nil); err != biz.ErrOrderAttachmentStorageUnavailable {
		t.Fatal("关闭时不应上传")
	}
	if _, err := NewOrderAttachmentStorage(&conf.Data{OrderAttachmentOss: &conf.Data_OrderAttachmentOSS{Enabled: true}}); err == nil {
		t.Fatal("缺失配置应拒绝启动")
	}
	settings := &conf.Data_OrderAttachmentOSS{Enabled: true, Region: "cn-beijing", Bucket: "test", AccessKey: "test-id", SecretKey: "test-secret", Endpoint: "http://oss-cn-beijing.aliyuncs.com"}
	if _, err := NewOrderAttachmentStorage(&conf.Data{OrderAttachmentOss: settings}); err == nil {
		t.Fatal("不安全endpoint应拒绝")
	}
	settings.Endpoint = "https://oss-cn-beijing.aliyuncs.com"
	if _, err := NewOrderAttachmentStorage(&conf.Data{OrderAttachmentOss: settings}); err != nil {
		t.Fatal("有效配置不应拒绝")
	}
}
