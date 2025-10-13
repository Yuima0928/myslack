# 既存EC2は import してから管理:
# terraform import 'module.ec2.aws_instance.api' i-xxxxxxxxxxxx
resource "aws_instance" "api" {
  ami                         = var.ami # ← 実機の AMI に合わせる
  instance_type               = "t3.small"
  subnet_id                   = var.public_subnet_id
  vpc_security_group_ids      = [var.security_group_id]
  iam_instance_profile        = var.iam_instance_profile
  associate_public_ip_address = true
  key_name                    = "myslack-key2" # ← 実機に合わせる

  tags = { Name = "myslack-api2" }
}


# EIP 紐付け（既存EIPの allocation_id を使用）
# terraform import 'module.ec2.aws_eip_association.api' eipassoc-xxxxxxxx
resource "aws_eip_association" "api" {
  allocation_id = var.eip_allocation_id
  instance_id   = aws_instance.api.id
}
