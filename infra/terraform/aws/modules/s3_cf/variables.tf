variable "aws_region" { type = string }
variable "site_bucket_name" { type = string }
variable "uploads_prefix" { type = string }
variable "cf_comment" { type = string }
variable "cf_price_class" { type = string }
variable "oac_name" {
  type        = string
  description = "既存 CloudFront OAC の name（コンソールと一致させる）"
}
variable "allowed_origin" {
  type = string
}

variable "cache_policy_id" {
  type = string
}

variable "origin_id" {
  type = string
}

