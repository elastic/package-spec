# A Terraform file
resource "aws_s3_object" "object" {
  bucket = aws_s3_bucket.bucket.id
  key    = "new_object_key"
  source = "./logs/example.log"
}

resource "aws_s3_bucket" "bucket" {
  bucket = "test-terraform-deploy-bucket"
}