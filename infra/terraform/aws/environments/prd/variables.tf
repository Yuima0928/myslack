variable "aws_region" {
  type    = string
  default = "ap-northeast-1"
}
variable "aws_profile" {
  type    = string
  default = "myslack-prod"
}

# 既存リソースID（Consoleの値を入れる）
variable "vpc_id" { type = string }            # 例: vpc-xxxxxxxx
variable "public_subnet_id" { type = string }  # 例: subnet-0288c4c0ddc4d5f37
variable "security_group_id" { type = string } # 例: sg-0d1bcd7cfa099e8be
variable "instance_id" { type = string }       # 例: i-0c83db1c81434e17d
variable "eip_allocation_id" { type = string } # 例: eipalloc-085cca51fd2853b7f

variable "site_bucket_name" {
  type    = string
  default = "myslack-web-prod"
}
variable "uploads_prefix" {
  type    = string
  default = "uploads"
}

variable "cf_comment" {
  type    = string
  default = "myslack prod"
}
variable "cf_price_class" {
  type    = string
  default = "PriceClass_200"
}

variable "oac_name" {
  type = string
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

variable "ami" {
  type = string
}

variable "role_name" {
  type = string
}
variable "policy_name" {
  type = string
}
variable "instance_profile_name" {
  type = string
}
