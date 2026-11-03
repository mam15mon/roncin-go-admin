package data

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/url"
	"strings"
	"time"

	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss"
	"github.com/aliyun/alibabacloud-oss-go-sdk-v2/oss/credentials"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

const orderAttachmentUploadTTL = 10 * time.Minute

type orderAttachmentStorage struct {
	enabled    bool
	bucket     string
	client     *oss.Client
	signingKey []byte
	now        func() time.Time
}

type attachmentUploadGrant struct {
	OrganizationID uuid.UUID
	ActorID        uuid.UUID
	OrderID        uuid.UUID
	Attachment     biz.OrderAttachment
	ExpiresAt      int64
}

func NewOrderAttachmentStorage(config *conf.Data) (biz.OrderAttachmentStorage, error) {
	settings := config.GetOrderAttachmentOss()
	storage := &orderAttachmentStorage{enabled: settings.GetEnabled(), bucket: settings.GetBucket(), now: time.Now}
	if !storage.enabled {
		return storage, nil
	}
	if strings.TrimSpace(settings.GetRegion()) == "" || strings.TrimSpace(storage.bucket) == "" || strings.TrimSpace(settings.GetAccessKey()) == "" || strings.TrimSpace(settings.GetSecretKey()) == "" || strings.TrimSpace(settings.GetEndpoint()) == "" {
		return nil, fmt.Errorf("订单附件 OSS 已启用但 endpoint、region、bucket 或访问密钥不完整")
	}
	endpoint, err := url.Parse(settings.GetEndpoint())
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || (endpoint.Path != "" && endpoint.Path != "/") {
		return nil, fmt.Errorf("订单附件 OSS endpoint 必须是有效的 HTTPS 服务地址")
	}
	configOSS := oss.LoadDefaultConfig().WithRegion(settings.GetRegion()).WithEndpoint(settings.GetEndpoint()).
		WithCredentialsProvider(credentials.NewStaticCredentialsProvider(settings.GetAccessKey(), settings.GetSecretKey())).
		WithConnectTimeout(5 * time.Second).WithReadWriteTimeout(30 * time.Second).WithRetryMaxAttempts(2)
	storage.client = oss.NewClient(configOSS)
	// 使用领域分离的派生密钥，上传凭证既不能用于 OSS，也不能跨 Bucket 使用。
	mac := hmac.New(sha256.New, []byte(settings.GetSecretKey()))
	mac.Write([]byte("roncin:order-attachment-upload:v1:" + settings.GetAccessKey() + ":" + storage.bucket))
	storage.signingKey = mac.Sum(nil)
	return storage, nil
}

func (s *orderAttachmentStorage) Enabled() bool { return s.enabled }

func (s *orderAttachmentStorage) PrepareUpload(ctx context.Context, organizationID, actorID, orderID uuid.UUID, input *biz.OrderAttachment) (*biz.OrderAttachmentUpload, error) {
	if !s.enabled {
		return nil, biz.ErrOrderAttachmentStorageUnavailable
	}
	// OSS 的禁止覆盖头在开启过版本控制的桶中不生效，必须显式拒绝此配置。
	versioning, err := s.client.GetBucketVersioning(ctx, &oss.GetBucketVersioningRequest{Bucket: oss.Ptr(s.bucket)})
	if err != nil {
		return nil, biz.ErrOrderAttachmentStorageUnavailable
	}
	if oss.ToString(versioning.VersionStatus) != "" {
		return nil, biz.ErrOrderAttachmentVersioning
	}
	attachment := *input
	id := uuid.Must(uuid.NewV7()).String()
	attachment.IdempotencyKey = id
	attachment.ObjectKey = "orders/" + organizationID.String() + "/" + orderID.String() + "/" + id
	expires := s.now().Add(orderAttachmentUploadTTL)
	grant := attachmentUploadGrant{OrganizationID: organizationID, ActorID: actorID, OrderID: orderID, Attachment: attachment, ExpiresAt: expires.Unix()}
	token, err := s.encodeGrant(grant)
	if err != nil {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	signed, err := s.client.Presign(ctx, &oss.PutObjectRequest{
		Bucket: oss.Ptr(s.bucket), Key: oss.Ptr(attachment.ObjectKey), ContentType: oss.Ptr(attachment.MIMEType),
		ContentLength: oss.Ptr(attachment.FileSize), ForbidOverwrite: oss.Ptr("true"), Acl: oss.ObjectACLPrivate,
		Metadata: map[string]string{"organization-id": organizationID.String(), "order-id": orderID.String(), "uploader-id": actorID.String(), "checksum-sha256": attachment.Checksum},
	}, func(options *oss.PresignOptions) { options.Expiration = expires })
	if err != nil {
		return nil, biz.ErrOrderAttachmentStorageUnavailable
	}
	headers := make(map[string]string)
	for key, value := range signed.SignedHeaders {
		if !strings.EqualFold(key, "host") && !strings.EqualFold(key, "content-length") {
			headers[key] = value
		}
	}
	return &biz.OrderAttachmentUpload{UploadURL: signed.URL, UploadToken: token, Headers: headers, ExpiresAt: expires}, nil
}

func (s *orderAttachmentStorage) encodeGrant(grant attachmentUploadGrant) (string, error) {
	payload, err := json.Marshal(grant)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, s.signingKey)
	mac.Write([]byte(encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (s *orderAttachmentStorage) decodeGrant(token string) (*attachmentUploadGrant, error) {
	if len(token) > 8192 {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	mac := hmac.New(sha256.New, s.signingKey)
	mac.Write([]byte(parts[0]))
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	var grant attachmentUploadGrant
	if err := json.Unmarshal(payload, &grant); err != nil || grant.ExpiresAt <= s.now().Unix() {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	return &grant, nil
}

func (s *orderAttachmentStorage) VerifyUpload(ctx context.Context, organizationID, actorID, orderID uuid.UUID, token string) (*biz.OrderAttachment, error) {
	if !s.enabled {
		return nil, biz.ErrOrderAttachmentStorageUnavailable
	}
	grant, err := s.decodeGrant(token)
	if err != nil {
		return nil, err
	}
	if grant.OrganizationID != organizationID || grant.ActorID != actorID || grant.OrderID != orderID || !strings.HasPrefix(grant.Attachment.ObjectKey, "orders/"+organizationID.String()+"/"+orderID.String()+"/") {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	attachment := &grant.Attachment
	head, err := s.client.HeadObject(ctx, &oss.HeadObjectRequest{Bucket: oss.Ptr(s.bucket), Key: oss.Ptr(attachment.ObjectKey)})
	if err != nil {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	if head.ContentLength != attachment.FileSize || oss.ToString(head.ContentType) != attachment.MIMEType || head.Metadata["organization-id"] != organizationID.String() || head.Metadata["order-id"] != orderID.String() || head.Metadata["uploader-id"] != actorID.String() || head.Metadata["checksum-sha256"] != attachment.Checksum || oss.ToString(head.ETag) == "" {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	// metadata 只是客户端声明：读取有限长度的实际内容核算 SHA-256，不能把声明当校验。
	object, err := s.client.GetObject(ctx, &oss.GetObjectRequest{Bucket: oss.Ptr(s.bucket), Key: oss.Ptr(attachment.ObjectKey), IfMatch: head.ETag})
	if err != nil {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	defer object.Body.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(object.Body, attachment.FileSize+1))
	if err != nil || size != attachment.FileSize || hex.EncodeToString(hash.Sum(nil)) != attachment.Checksum {
		return nil, biz.ErrOrderAttachmentUploadInvalid
	}
	return attachment, nil
}

func validOrderAttachmentKey(organizationID uuid.UUID, key string) bool {
	parts := strings.Split(key, "/")
	if len(parts) != 4 || parts[0] != "orders" || parts[1] != organizationID.String() {
		return false
	}
	orderID, orderErr := uuid.Parse(parts[2])
	assetID, assetErr := uuid.Parse(parts[3])
	return orderErr == nil && assetErr == nil && orderID != uuid.Nil && assetID != uuid.Nil
}

func (s *orderAttachmentStorage) Access(ctx context.Context, organizationID uuid.UUID, attachment *biz.OrderAttachment, download bool) (*biz.OrderAttachmentAccess, error) {
	if !s.enabled {
		return nil, biz.ErrOrderAttachmentStorageUnavailable
	}
	if attachment == nil || !validOrderAttachmentKey(organizationID, attachment.ObjectKey) {
		return nil, biz.ErrOrderAttachmentInvalidArgument
	}
	disposition := "attachment"
	if !download && (attachment.MIMEType == "application/pdf" || attachment.MIMEType == "image/png" || attachment.MIMEType == "image/jpeg" || attachment.MIMEType == "image/gif" || attachment.MIMEType == "image/webp" || attachment.MIMEType == "image/bmp") {
		disposition = "inline"
	}
	expires := s.now().Add(5 * time.Minute)
	signed, err := s.client.Presign(ctx, &oss.GetObjectRequest{Bucket: oss.Ptr(s.bucket), Key: oss.Ptr(attachment.ObjectKey),
		ResponseContentDisposition: oss.Ptr(mime.FormatMediaType(disposition, map[string]string{"filename": attachment.FileName})), ResponseCacheControl: oss.Ptr("private, no-store"),
	}, func(options *oss.PresignOptions) { options.Expiration = expires })
	if err != nil {
		return nil, biz.ErrOrderAttachmentStorageUnavailable
	}
	return &biz.OrderAttachmentAccess{URL: signed.URL, ExpiresAt: expires}, nil
}

func (s *orderAttachmentStorage) Delete(ctx context.Context, organizationID uuid.UUID, objectKey string) error {
	if !s.enabled {
		return biz.ErrOrderAttachmentStorageUnavailable
	}
	if !validOrderAttachmentKey(organizationID, objectKey) {
		return biz.ErrOrderAttachmentInvalidArgument
	}
	_, err := s.client.DeleteObject(ctx, &oss.DeleteObjectRequest{Bucket: oss.Ptr(s.bucket), Key: oss.Ptr(objectKey)})
	if err != nil {
		return biz.ErrOrderAttachmentStorageUnavailable
	}
	return nil
}
