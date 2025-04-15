# CloudFront Access Logs V2 Configuration Tool

This Go application configures CloudFront Access Logs V2 using the AWS SDK Go v2. It automates the process of setting up CloudFront logs to be delivered to an S3 bucket using the new CloudWatch Logs delivery framework.

## Prerequisites

- Go 1.21 or later
- AWS credentials configured (via environment variables, AWS credentials file, or IAM role)
- Permissions to create CloudWatch Logs delivery sources, destinations, and deliveries
- An existing CloudFront distribution
- An existing S3 bucket with appropriate permissions

## Installation

```bash
# Clone the repository
git clone <repository-url>
cd <repository-directory>

# Install dependencies
go mod tidy
```

## Usage

```bash
go run main.go --distribution-id=YOUR_DISTRIBUTION_ID --bucket-name=YOUR_BUCKET_NAME [--region=us-east-1]
```

### Parameters

- `--distribution-id`: (Required) The ID of your CloudFront distribution
- `--bucket-name`: (Required) The name of the S3 bucket where logs will be stored
- `--region`: (Optional) AWS region (default: us-east-1)

## How It Works

The application performs the following steps:

1. Creates a delivery source pointing to your CloudFront distribution
2. Creates a delivery destination pointing to your S3 bucket
3. Creates a delivery that connects the source to the destination with appropriate configuration

## Example

```bash
go run main.go --distribution-id=E1DOYPS2VYJZXX --bucket-name=cloudfrong-sdk-324256
```

## Important Notes

- The account ID is currently a placeholder. In a production environment, you should implement the `getAccountID` function to retrieve your actual AWS account ID using the STS service.
- Make sure your S3 bucket has the appropriate permissions to receive logs from CloudWatch Logs.

## Athena operations

### create Athena table
```
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
LOCATION 's3://cloudfront-sdk-logs-12345265/AWSLogs/aws-account-id=123456789012/CloudFront/DistributionId=E1YJRG2ECJVHXX/'
TBLPROPERTIES ("skip.header.line.count"="1")

```

### load partitions
```
MSCK REPAIR TABLE `cloudfront_logs`;

```

### Query data
```
select count(*) FROM "default"."cloudfront_logs" where year='2025' and month='04' and day='14' and hour='14'
```
