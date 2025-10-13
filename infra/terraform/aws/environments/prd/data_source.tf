# 既存ネットワーク系を data 参照
data "aws_vpc" "this" { id = var.vpc_id }
data "aws_subnet" "public" { id = var.public_subnet_id }
data "aws_security_group" "api_sg" { id = var.security_group_id }
