resource "aws_cloudfront_distribution" "cdn" {
  enabled             = true
  is_ipv6_enabled     = true
  comment             = var.cf_comment
  default_root_object = "index.html"
  price_class         = var.cf_price_class

  origin {
    domain_name              = "${aws_s3_bucket.site.bucket}.s3.${var.aws_region}.amazonaws.com"
    origin_id                = var.origin_id
    origin_access_control_id = aws_cloudfront_origin_access_control.oac.id
    connection_attempts      = 3
    connection_timeout       = 10
  }

  default_cache_behavior {
    target_origin_id       = var.origin_id
    viewer_protocol_policy = "redirect-to-https"
    allowed_methods        = ["GET", "HEAD"]
    cached_methods         = ["GET", "HEAD"]
    cache_policy_id        = var.cache_policy_id
    compress               = true
  }

  restrictions {
    geo_restriction { restriction_type = "none" }
  }

  viewer_certificate {
    cloudfront_default_certificate = true
    minimum_protocol_version       = "TLSv1"
  }

  custom_error_response {
    error_caching_min_ttl = 0
    error_code            = 403
    response_code         = 200
    response_page_path    = "/index.html"
  }

  custom_error_response {
    error_caching_min_ttl = 0
    error_code            = 404
    response_code         = 200
    response_page_path    = "/index.html"
  }

  tags = {
    Name = "myslack-web"
  }
}

# 既存 OAC を state 管理するための定義（差分は無視）
resource "aws_cloudfront_origin_access_control" "oac" {
  name                              = var.oac_name # 既存の名称に合わせる
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
  description                       = "Created by CloudFront"
}

