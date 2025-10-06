// internal/storage/s3client.go
package storage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"slackgo/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Deps struct {
	Client  *s3.Client
	Presign *s3.PresignClient
	Bucket  string
	Prefix  string
	Expire  time.Duration
}

func NewS3Deps(ctx context.Context, c config.Config) (*S3Deps, error) {
	// 1) LoadDefaultConfig に static creds を「必要なときだけ」渡す
	loadOpts := []func(*awscfg.LoadOptions) error{
		awscfg.WithRegion(c.AWSRegion),
	}

	if c.S3AccessKey != "" && c.S3SecretKey != "" {
		loadOpts = append(loadOpts,
			awscfg.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(c.S3AccessKey, c.S3SecretKey, ""),
			),
		)
		// ※ キーが無い場合は渡さない → IAMロール/IMDS 等のデフォルトチェーンを使う
	}

	cfg, err := awscfg.LoadDefaultConfig(ctx, loadOpts...)
	if err != nil {
		return nil, err
	}

	// 2) 内部用クライアント（MinIO等カスタムのときだけ BaseEndpoint/PathStyle を触る）
	internal := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if c.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(c.S3Endpoint)
			o.UsePathStyle = c.S3UsePathStyle
		}
	})

	// 3) 署名用クライアント
	//    - ブラウザが到達する公開側エンドポイントが必要なときだけ BaseEndpoint を指定
	//    - AWS純正S3なら未指定でOK（リージョンとバケットから自動で組み立て）
	signerClient := s3.NewFromConfig(cfg, func(o *s3.Options) {
		publicBase := c.S3PublicEndpoint
		if publicBase == "" {
			publicBase = c.S3Endpoint // フォールバック（MinIO等）
		}
		if publicBase != "" {
			o.BaseEndpoint = aws.String(publicBase)
			o.UsePathStyle = c.S3UsePathStyle
		}
	})

	return &S3Deps{
		Client: internal,
		Presign: s3.NewPresignClient(signerClient, func(po *s3.PresignOptions) {
			po.Expires = time.Duration(c.S3URLExpirySec) * time.Second
		}),
		Bucket: c.S3Bucket,
		Prefix: c.S3Prefix, // 例: "uploads"
		Expire: time.Duration(c.S3URLExpirySec) * time.Second,
	}, nil
}

func (s *S3Deps) SignGetURL(
	ctx context.Context,
	storageKey string,
	filename string,
	contentType string,
	disposition string, // "inline" or "attachment"
) (string, error) {
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	cd := fmt.Sprintf(`%s; filename="%s"`, disposition, url.PathEscape(filename))

	out, err := s.Presign.PresignGetObject(
		ctx,
		&s3.GetObjectInput{
			Bucket:                     aws.String(s.Bucket),
			Key:                        aws.String(storageKey),
			ResponseContentDisposition: aws.String(cd),
			ResponseContentType:        aws.String(contentType),
		},
		func(o *s3.PresignOptions) { o.Expires = s.Expire },
	)
	if err != nil {
		return "", err
	}
	return out.URL, nil
}

func (s *S3Deps) Expiry() time.Duration { return s.Expire }
