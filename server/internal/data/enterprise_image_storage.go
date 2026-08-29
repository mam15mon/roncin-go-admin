package data

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

type enterpriseImageStorage struct {
	enabled bool
	bucket  string
	client  *s3.Client
	presign *s3.PresignClient
}

func NewEnterpriseImageStorage(config *conf.Data) (biz.EnterpriseImageStorage, error) {
	settings := config.GetObjectStorage()
	storage := &enterpriseImageStorage{enabled: settings.GetEnabled(), bucket: settings.GetBucket()}
	if !settings.GetEnabled() {
		return storage, nil
	}
	if settings.GetRegion() == "" || settings.GetBucket() == "" || settings.GetAccessKey() == "" || settings.GetSecretKey() == "" {
		return nil, fmt.Errorf("对象存储已启用但 region、bucket 或访问密钥不完整")
	}
	awsSettings, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(settings.GetRegion()),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(settings.GetAccessKey(), settings.GetSecretKey(), "")),
	)
	if err != nil {
		return nil, fmt.Errorf("加载对象存储配置: %w", err)
	}
	client := s3.NewFromConfig(awsSettings, func(options *s3.Options) {
		options.UsePathStyle = settings.GetPathStyle()
		if settings.GetEndpoint() != "" {
			options.BaseEndpoint = aws.String(settings.GetEndpoint())
		}
	})
	storage.client = client
	storage.presign = s3.NewPresignClient(client)
	return storage, nil
}

func (s *enterpriseImageStorage) Enabled() bool {
	return s.enabled
}

func (s *enterpriseImageStorage) PrepareUpload(ctx context.Context, organizationID uuid.UUID, fileName, mimeType string, fileSize int64, checksum string) (*biz.EnterpriseImageUpload, error) {
	if !s.enabled {
		return nil, biz.ErrEnterpriseImageStorageUnavailable
	}
	extension := strings.ToLower(filepath.Ext(fileName))
	key := fmt.Sprintf("enterprise-resources/%s/%s%s", organizationID.String(), uuid.Must(uuid.NewV7()).String(), extension)
	expires := 10 * time.Minute
	request, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), ContentType: aws.String(mimeType), ContentLength: aws.Int64(fileSize),
		Metadata: map[string]string{"checksum": checksum, "organization-id": organizationID.String()},
	}, s3.WithPresignExpires(expires))
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	for name, values := range request.SignedHeader {
		if len(values) > 0 && !strings.EqualFold(name, "content-length") && !strings.EqualFold(name, "host") {
			headers[name] = values[0]
		}
	}
	return &biz.EnterpriseImageUpload{UploadURL: request.URL, ObjectKey: key, Headers: headers, ExpiresAt: time.Now().Add(expires)}, nil
}

func (s *enterpriseImageStorage) VerifyUpload(ctx context.Context, organizationID uuid.UUID, image *biz.EnterpriseResourceImage) error {
	if !s.enabled {
		return biz.ErrEnterpriseImageStorageUnavailable
	}
	if !strings.HasPrefix(image.ObjectKey, "enterprise-resources/"+organizationID.String()+"/") {
		return biz.ErrEnterpriseResourceInvalidArgument
	}
	result, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(image.ObjectKey)})
	if err != nil {
		return biz.ErrEnterpriseResourceInvalidArgument
	}
	if result.ContentLength == nil || *result.ContentLength != image.FileSize || result.ContentType == nil || *result.ContentType != image.MIMEType || result.Metadata["checksum"] != image.Checksum || result.Metadata["organization-id"] != organizationID.String() {
		return biz.ErrEnterpriseResourceInvalidArgument
	}
	return nil
}

func (s *enterpriseImageStorage) PresignGet(ctx context.Context, organizationID uuid.UUID, objectKey string) (string, time.Time, error) {
	if !s.enabled {
		return "", time.Time{}, biz.ErrEnterpriseImageStorageUnavailable
	}
	if !strings.HasPrefix(objectKey, "enterprise-resources/"+organizationID.String()+"/") {
		return "", time.Time{}, biz.ErrEnterpriseResourceInvalidArgument
	}
	expires := 10 * time.Minute
	request, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(objectKey)}, s3.WithPresignExpires(expires))
	if err != nil {
		return "", time.Time{}, err
	}
	return request.URL, time.Now().Add(expires), nil
}

func (s *enterpriseImageStorage) Delete(ctx context.Context, organizationID uuid.UUID, objectKey string) error {
	if !s.enabled {
		return biz.ErrEnterpriseImageStorageUnavailable
	}
	if !strings.HasPrefix(objectKey, "enterprise-resources/"+organizationID.String()+"/") {
		return biz.ErrEnterpriseResourceInvalidArgument
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(objectKey)})
	return err
}
