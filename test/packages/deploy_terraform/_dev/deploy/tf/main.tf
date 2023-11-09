# A Terraform file
resource "aws_s3_object" "object" {
  bucket = aws_s3_bucket.bucket.id
  key    = "new_object_key"
  source = "./logs/example.log"
}

resource "aws_s3_bucket" "bucket" {
  bucket = "test-terraform-deploy-bucket"
}

resource "aws_s3_object" "parquet_object" {
  bucket = aws_s3_bucket.bucket.id
  key    = "test_parquet_key"
  source = "./files/test.gz.parquet"

  depends_on = [aws_sqs_queue.queue]
}