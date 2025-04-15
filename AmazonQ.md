# Setting up CloudFront with Standard Logs V2

This document explains how to use the `create_cloudfront_with_logs.go` script to create a CloudFront distribution with CloudFront Access Logs V2 configured.

## Prerequisites

- Go 1.21 or later
- AWS credentials configured (via environment variables, AWS credentials file, or IAM role)
- Permissions to create CloudFront distributions and CloudWatch Logs delivery sources, destinations, and deliveries
- An existing S3 bucket with appropriate permissions to receive logs

## Usage

```bash
go run create_cloudfront_with_logs.go \
  --origin-host=example.com \
  --cache-policy-id=658327ea-f89d-4fab-a63d-7e88639e58f6 \
  --origin-request-policy-id=88a5eaf4-2fd4-4709-b370-b4c650ea3fcf \
  --response-headers-policy-id=67f7725c-6f97-4210-82d7-5512b31e9d03 \
  --bucket-name=your-logs-bucket-name
```

### Parameters

- `--origin-host`: (Required) The domain name of your origin server
- `--cache-policy-id`: (Required) The ID of the cache policy to use
- `--origin-request-policy-id`: (Required) The ID of the origin request policy to use
- `--response-headers-policy-id`: (Required) The ID of the response headers policy to use
- `--bucket-name`: (Required) The name of the S3 bucket where logs will be stored
- `--region`: (Optional) AWS region (default: us-east-1)

## Common Cache Policy IDs

- **Managed-CachingOptimized** (658327ea-f89d-4fab-a63d-7e88639e58f6): Optimized for caching performance
- **Managed-CachingDisabled** (4135ea2d-6df8-44a3-9df3-4b5a84be39ad): Disables caching
- **Managed-CachingOptimizedForUncompressedObjects** (b2884449-e4de-46a7-ac36-70bc7f1ddd6d): Optimized for uncompressed objects

## Common Origin Request Policy IDs

- **Managed-AllViewer** (216adef6-5c7f-47e4-b989-5492eafa07d3): Forwards all viewer request values
- **Managed-CORS-S3Origin** (88a5eaf4-2fd4-4709-b370-b4c650ea3fcf): Optimized for CORS requests to S3
- **Managed-AllViewerExceptHostHeader** (b689b0a8-53d0-40ab-baf2-68738e2966ac): Forwards all viewer request values except Host header

## Common Response Headers Policy IDs

- **Managed-CORS-With-Preflight** (5cc3b908-e619-4b99-88e5-2cf7f45965bd): CORS headers with preflight support
- **Managed-CORS-with-preflight-and-SecurityHeadersPolicy** (e61eb60c-9c35-4d20-a928-2b84e02af89c): CORS headers with security headers
- **Managed-SecurityHeadersPolicy** (67f7725c-6f97-4210-82d7-5512b31e9d03): Common security headers

## How It Works

The script performs the following steps:

1. Creates a new CloudFront distribution with the specified origin, cache policy, origin request policy, and response headers policy
2. Creates a delivery source pointing to the newly created CloudFront distribution
3. Creates a delivery destination pointing to your S3 bucket
4. Creates a delivery that connects the source to the destination with appropriate configuration for CloudFront Access Logs V2

## Athena Integration

After logs start flowing to your S3 bucket, you can query them using Amazon Athena. Here's how to set up an Athena table:

```sql
CREATE EXTERNAL TABLE `cloudfront_logs`(
  `date` string, 
  `time` string, 
  `x_edge_location` string, 
  `sc_bytes` string, 
  `c_ip` string, 
  `cs_method` string, 
  `cs_host` string, 
  `cs_uri_stem` string, 
  `sc_status` string, 
  `cs_referer` string, 
  `cs_user_agent` string, 
  `cs_uri_query` string, 
  `cs_cookie` string, 
  `x_edge_result_type` string, 
  `x_edge_request_id` string, 
  `x_host_header` string, 
  `cs_protocol` string, 
  `cs_bytes` string, 
  `time_taken` string, 
  `x_forwarded_for` string, 
  `ssl_protocol` string, 
  `ssl_cipher` string, 
  `x_edge_response_result_type` string, 
  `cs_protocol_version` string, 
  `fle_status` string, 
  `fle_encrypted_fields` string, 
  `c_port` string, 
  `time_to_first_byte` string, 
  `x_edge_detailed_result_type` string, 
  `sc_content_type` string, 
  `sc_content_len` string, 
  `sc_range_start` string, 
  `sc_range_end` string)
PARTITIONED BY(
 year string,
 month string,
 day string,
 hour string )
ROW FORMAT DELIMITED FIELDS TERMINATED BY '\t'
LOCATION 's3://your-logs-bucket-name/AWSLogs/aws-account-id=123456789012/CloudFront/DistributionId=EXAMPLEID/'
TBLPROPERTIES ("skip.header.line.count"="1")
```

Load partitions:
```sql
MSCK REPAIR TABLE `cloudfront_logs`;
```

Query data:
```sql
SELECT count(*) FROM "default"."cloudfront_logs" WHERE year='2025' AND month='04' AND day='15' AND hour='14'
```
