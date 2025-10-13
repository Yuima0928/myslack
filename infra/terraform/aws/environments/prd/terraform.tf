terraform {
  backend "s3" {
    bucket = "myslack-tfstate-prod" # tfstateバケット（既存を推奨）
    key    = "myslack/prod/terraform.tfstate"
    region = "ap-northeast-1"
  }
}
