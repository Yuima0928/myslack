data "aws_iam_policy_document" "assume_ec2" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "ec2_role" {
  name               = var.role_name
  assume_role_policy = data.aws_iam_policy_document.assume_ec2.json
}

data "aws_iam_policy_document" "s3_uploads_rw" {
  statement {
    sid       = "ObjectRW"
    actions   = ["s3:GetObject", "s3:PutObject", "s3:DeleteObject", "s3:AbortMultipartUpload"]
    resources = ["${var.site_bucket_arn}/${var.uploads_prefix}/*"]
  }
  statement {
    sid       = "ListForPrefixes"
    actions   = ["s3:ListBucket", "s3:ListBucketMultipartUploads"]
    resources = [var.site_bucket_arn]
    condition {
      test     = "StringLike"
      variable = "s3:prefix"
      values   = ["${var.uploads_prefix}/*"]
    }
  }
}

resource "aws_iam_policy" "s3_uploads_rw" {
  name   = var.policy_name
  policy = data.aws_iam_policy_document.s3_uploads_rw.json
}


resource "aws_iam_role_policy_attachment" "attach_s3" {
  role       = aws_iam_role.ec2_role.name
  policy_arn = aws_iam_policy.s3_uploads_rw.arn
}

resource "aws_iam_instance_profile" "ec2_profile" {
  name = var.instance_profile_name
  role = aws_iam_role.ec2_role.name
}

