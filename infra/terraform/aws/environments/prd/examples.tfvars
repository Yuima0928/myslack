# -------- AWS 基本 --------
aws_region  = "ap-northeast-1" # ← ここを埋めてください（例: ap-northeast-1）
aws_profile = "myslack-xxxx"   # ← ここを埋めてください（AWS CLIのプロファイル名）

# 既存VPC/サブネット/SGを流用する場合（IDは環境に合わせて）
vpc_id            = "vpc-0abcde1234567890f"  # ← ここを埋めてください（vpc- で始まるID）
public_subnet_id  = "subnet-0abcde123456789" # ← ここを埋めてください（public subnetのID）
security_group_id = "sg-0abcde1234567890f"   # ← ここを埋めてください（インバウンド80/443許可推奨）

# -------- EC2 --------
# AMIは最新のAmazon Linux 2023などに置換を推奨（data.aws_amiで動的取得でもOK）
ami = "ami-0123456789abcdef0" # ← ここを埋めてください（使用するAMI ID）
# EIPを使わない場合は未設定のまま
# eip_allocation_id = ""

# -------- S3 / CloudFront --------
# バケット名は世界一意。scratch用は衝突しづらい命名にする
site_bucket_name = "myslack-web-scratch-xxxxxx" # ← ここを埋めてください（必ずユニーク）
uploads_prefix   = "uploads"                    # ← 必要に応じて変更

# CloudFront は新規作成（コメントや価格帯）
cf_comment     = "myslack scratch" # ← 任意
cf_price_class = "PriceClass_200"  # ← 例: PriceClass_All / PriceClass_200 / PriceClass_100

# OAC は新規作成名（既存再利用なら名称を一致）
oac_name = "oac-myslack-web-scratch" # ← 任意（被らない名前）

# CORS：最初は緩めでもよいが、本番はCFのFQDNへ絞る
allowed_origin = "*" # ← 例: https://dxxxxxxxxxxxx.cloudfront.net

# キャッシュポリシー（AWSデフォルトのCACHING_OPTIMIZEDのID例）
cache_policy_id = "658327ea-f89d-4fab-a63d-7e88639e58f6" # ← 運用に合わせて変更可

# Origin ID は module.cloudfront と一致させる（モジュール側の期待値に合わせる）
origin_id = "s3-origin" # ← 変更する場合は両側で揃える

# -------- IAM (EC2 → S3 アップロード用など) --------
role_name             = "MyApp-EC2-S3Uploads-scratch"   # ← 任意（被らない名前）
policy_name           = "myslack-s3-uploads-rw-scratch" # ← 任意（被らない名前）
instance_profile_name = "MyApp-EC2-S3Uploads-scratch"   # ← 任意（被らない名前）
