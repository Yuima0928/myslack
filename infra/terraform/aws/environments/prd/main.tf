module "network" {
  source            = "../../modules/network"
  vpc_id            = var.vpc_id
  public_subnet_id  = var.public_subnet_id
  security_group_id = var.security_group_id
}

module "role" {
  source                = "../../modules/role"
  site_bucket_arn       = "arn:aws:s3:::${var.site_bucket_name}"
  uploads_prefix        = var.uploads_prefix
  role_name             = var.role_name
  policy_name           = var.policy_name
  instance_profile_name = var.instance_profile_name
}

module "ec2" {
  source               = "../../modules/ec2"
  instance_id          = var.instance_id
  public_subnet_id     = var.public_subnet_id
  security_group_id    = var.security_group_id
  iam_instance_profile = module.role.instance_profile_name
  eip_allocation_id    = var.eip_allocation_id
  ami                  = var.ami
}

module "s3_cf" {
  source           = "../../modules/s3_cf"
  aws_region       = var.aws_region
  site_bucket_name = var.site_bucket_name
  uploads_prefix   = var.uploads_prefix
  cf_comment       = var.cf_comment
  cf_price_class   = var.cf_price_class
  oac_name         = var.oac_name
  allowed_origin   = var.allowed_origin
  cache_policy_id  = var.cache_policy_id
  origin_id        = var.origin_id
}
