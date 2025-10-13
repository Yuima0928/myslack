# 既存バケットを import して管理
# terraform import 'module.s3_cf.aws_s3_bucket.site' myslack-web-prod
resource "aws_s3_bucket" "site" {
  bucket = var.site_bucket_name
}

# CORS（まずは緩め。必要に応じて CF ドメインに絞ってOK）
resource "aws_s3_bucket_cors_configuration" "site_cors" {
  bucket = aws_s3_bucket.site.id
  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "HEAD", "PUT"]
    allowed_origins = [var.allowed_origin]
    expose_headers  = ["ETag", "x-amz-request-id", "x-amz-id-2"]
    max_age_seconds = 3000
  }
}

# CloudFront からの GetObject を許可（OAC/SourceArn 基準）
data "aws_caller_identity" "this" {}

data "aws_iam_policy_document" "site_policy" {
  statement {
    sid       = "AllowCloudFrontServicePrincipal"
    effect    = "Allow"
    actions   = ["s3:GetObject"]
    resources = ["${aws_s3_bucket.site.arn}/*"]
    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [aws_cloudfront_distribution.cdn.arn]
    }
  }
}

# 既存ポリシーとの差分は初期は無視（安定したら外す）
resource "aws_s3_bucket_policy" "site" {
  bucket = aws_s3_bucket.site.id
  policy = data.aws_iam_policy_document.site_policy.json
}

output "bucket_name" { value = aws_s3_bucket.site.bucket }
