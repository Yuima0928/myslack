variable "instance_id" { type = string }
variable "public_subnet_id" { type = string }
variable "security_group_id" { type = string }
variable "iam_instance_profile" { type = string }
variable "eip_allocation_id" { type = string }
variable "ami" {
  type = string
}
