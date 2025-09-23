// internal/storage/s3client.go
package storage

import (
	"context"
	"time"

	"slackgo/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awscfg "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Deps struct {
	Client  *s3.Client        // 内部アクセス用（サーバ→MinIO/S3）
	Presign *s3.PresignClient // 署名URL作成用（外部向けホストでサイン）
	Bucket  string
	Prefix  string
	Expire  time.Duration
}

func NewS3Deps(ctx context.Context, c config.Config) (*S3Deps, error) {
	cfg, err := awscfg.LoadDefaultConfig(ctx,
		awscfg.WithRegion(c.AWSRegion),
		awscfg.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			c.S3AccessKey, c.S3SecretKey, "",
		)),
	)
	if err != nil {
		return nil, err
	}

	// 1) 内部用クライアント（コンテナから到達できるエンドポイント）
	internal := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if c.S3Endpoint != "" {
			o.BaseEndpoint = aws.String(c.S3Endpoint)
		} // 例: http://minio:9000
		o.UsePathStyle = c.S3UsePathStyle
	})

	// 2) 署名用クライアント（ブラウザから到達できるエンドポイント）
	publicBase := c.S3PublicEndpoint
	if publicBase == "" {
		publicBase = c.S3Endpoint
	} // フォールバック
	signerClient := s3.NewFromConfig(cfg, func(o *s3.Options) {
		if publicBase != "" {
			o.BaseEndpoint = aws.String(publicBase)
		} // 例: http://localhost:9000
		o.UsePathStyle = c.S3UsePathStyle
	})

	return &S3Deps{
		Client: internal,
		Presign: s3.NewPresignClient(signerClient, func(po *s3.PresignOptions) {
			po.Expires = time.Duration(c.S3URLExpirySec) * time.Second
		}),
		Bucket: c.S3Bucket,
		Prefix: c.S3Prefix,
		Expire: time.Duration(c.S3URLExpirySec) * time.Second,
	}, nil
}
