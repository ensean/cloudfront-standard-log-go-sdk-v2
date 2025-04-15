package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront"
	"github.com/aws/aws-sdk-go-v2/service/cloudfront/types"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	cwltypes "github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func main() {
	// Define command line flags
	originHost := flag.String("origin-host", "", "Origin host (required)")
	cachePolicyID := flag.String("cache-policy-id", "", "Cache policy ID (required)")
	originRequestPolicyID := flag.String("origin-request-policy-id", "", "Origin request policy ID (required)")
	responseHeadersPolicyID := flag.String("response-headers-policy-id", "", "Response headers policy ID (required)")
	bucketName := flag.String("bucket-name", "", "S3 bucket name for logs (required)")
	region := flag.String("region", "us-east-1", "AWS region")
	flag.Parse()

	// Validate required parameters
	if *originHost == "" || *cachePolicyID == "" || *originRequestPolicyID == "" || 
	   *responseHeadersPolicyID == "" || *bucketName == "" {
		fmt.Println("Error: All parameters are required")
		flag.Usage()
		os.Exit(1)
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(*region))
	if err != nil {
		log.Fatalf("Unable to load SDK config: %v", err)
	}

	// Create CloudFront client
	cfClient := cloudfront.NewFromConfig(cfg)

	// Get account ID
	accountID := getAccountID(cfg)

	// Create CloudFront distribution
	distributionID, err := createCloudFrontDistribution(cfClient, *originHost, *cachePolicyID, 
		*originRequestPolicyID, *responseHeadersPolicyID)
	if err != nil {
		log.Fatalf("Failed to create CloudFront distribution: %v", err)
	}
	fmt.Printf("Successfully created CloudFront distribution with ID: %s\n", distributionID)

	// Configure CloudFront Access Logs V2
	err = configureCloudFrontLogsV2(cfg, distributionID, *bucketName, *region, accountID)
	if err != nil {
		log.Fatalf("Failed to configure CloudFront Access Logs V2: %v", err)
	}

	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("- Distribution ID: %s\n", distributionID)
	fmt.Printf("- Origin Host: %s\n", *originHost)
	fmt.Printf("- Cache Policy ID: %s\n", *cachePolicyID)
	fmt.Printf("- Origin Request Policy ID: %s\n", *originRequestPolicyID)
	fmt.Printf("- Response Headers Policy ID: %s\n", *responseHeadersPolicyID)
	fmt.Printf("- S3 Bucket for Logs: %s\n", *bucketName)
	fmt.Printf("- Region: %s\n", *region)
}

// Helper function to get AWS account ID using STS GetCallerIdentity
func getAccountID(cfg aws.Config) string {
	stsClient := sts.NewFromConfig(cfg)
	result, err := stsClient.GetCallerIdentity(context.TODO(), &sts.GetCallerIdentityInput{})
	if err != nil {
		log.Fatalf("Failed to get account ID: %v", err)
	}
	return *result.Account
}

// Create CloudFront distribution with specified parameters
func createCloudFrontDistribution(cfClient *cloudfront.Client, originHost, cachePolicyID, 
	originRequestPolicyID, responseHeadersPolicyID string) (string, error) {
	
	// Create distribution configuration
	distributionConfig := &types.DistributionConfig{
		CallerReference: aws.String(fmt.Sprintf("cli-reference-%d", time.Now().Unix())),
		Comment:         aws.String("Distribution created with CloudFront Access Logs V2"),
		Enabled:         aws.Bool(true),
		DefaultCacheBehavior: &types.DefaultCacheBehavior{
			TargetOriginId:         aws.String("primary-origin"),
			ViewerProtocolPolicy:   types.ViewerProtocolPolicyRedirectToHttps,
			CachePolicyId:          aws.String(cachePolicyID),
			OriginRequestPolicyId:  aws.String(originRequestPolicyID),
			ResponseHeadersPolicyId: aws.String(responseHeadersPolicyID),
		},
		Origins: &types.Origins{
			Quantity: aws.Int32(1),
			Items: []types.Origin{
				{
					Id:         aws.String("primary-origin"),
					DomainName: aws.String(originHost),
					CustomOriginConfig: &types.CustomOriginConfig{
						HTTPPort:             aws.Int32(80),
						HTTPSPort:            aws.Int32(443),
						OriginProtocolPolicy: types.OriginProtocolPolicyHttpsOnly,
						OriginSslProtocols: &types.OriginSslProtocols{
							Quantity: aws.Int32(1),
							Items:    []types.SslProtocol{types.SslProtocolTLSv12},
						},
					},
				},
			},
		},
		PriceClass: types.PriceClassPriceClass100,
	}

	// Create the distribution
	createDistInput := &cloudfront.CreateDistributionInput{
		DistributionConfig: distributionConfig,
	}

	result, err := cfClient.CreateDistribution(context.TODO(), createDistInput)
	if err != nil {
		return "", fmt.Errorf("failed to create distribution: %w", err)
	}

	return *result.Distribution.Id, nil
}

// Configure CloudFront Access Logs V2
func configureCloudFrontLogsV2(cfg aws.Config, distributionID, bucketName, region, accountID string) error {
	// Create CloudWatch Logs client
	logsClient := cloudwatchlogs.NewFromConfig(cfg)

	// Step 1: Create delivery source (CloudFront distribution)
	sourceName := fmt.Sprintf("CloudFront-%s", distributionID)
	distributionArn := fmt.Sprintf("arn:aws:cloudfront::%s:distribution/%s", accountID, distributionID)
	sourceInput := &cloudwatchlogs.PutDeliverySourceInput{
		Name:        aws.String(sourceName),
		ResourceArn: aws.String(distributionArn),
		LogType:     aws.String("ACCESS_LOGS"),
	}
	
	_, err := logsClient.PutDeliverySource(context.TODO(), sourceInput)
	if err != nil {
		return fmt.Errorf("failed to create delivery source: %w", err)
	}
	fmt.Printf("Successfully created delivery source: %s\n", sourceName)

	// Step 2: Create delivery destination (S3 bucket)
	destinationName := fmt.Sprintf("S3-destination-cloudfrontlogs-%s", distributionID)
	bucketArn := fmt.Sprintf("arn:aws:s3:::%s", bucketName)
	
	destinationInput := &cloudwatchlogs.PutDeliveryDestinationInput{
		Name: aws.String(destinationName),
		DeliveryDestinationConfiguration: &cwltypes.DeliveryDestinationConfiguration{
			DestinationResourceArn: aws.String(bucketArn),
		},
		OutputFormat: cwltypes.OutputFormatPlain,
	}
	
	_, err = logsClient.PutDeliveryDestination(context.TODO(), destinationInput)
	if err != nil {
		return fmt.Errorf("failed to create delivery destination: %w", err)
	}
	fmt.Printf("Successfully created delivery destination: %s\n", destinationName)

	// Step 3: Create delivery (connect source to destination)
	destinationArn := fmt.Sprintf("arn:aws:logs:%s:%s:delivery-destination:%s", 
		region, accountID, destinationName)
	
	// Create the delivery with the proper configuration
	createDeliveryInput := &cloudwatchlogs.CreateDeliveryInput{
		DeliverySourceName:     aws.String(sourceName),
		DeliveryDestinationArn: aws.String(destinationArn),
		Tags: map[string]string{
			"Service": "CloudFront",
			"Created": time.Now().Format(time.RFC3339),
		},
		S3DeliveryConfiguration: &cwltypes.S3DeliveryConfiguration{
			SuffixPath:              aws.String("{DistributionId}/{yyyy}/{MM}/{dd}/{HH}/"),
			EnableHiveCompatiblePath: aws.Bool(true),
		},
	}
	
	// Create the delivery
	deliveryResp, err := logsClient.CreateDelivery(context.TODO(), createDeliveryInput)
	if err != nil {
		return fmt.Errorf("failed to create delivery: %w", err)
	}
	
	deliveryID := *deliveryResp.Delivery.Id
	fmt.Printf("Successfully created delivery with ID: %s\n", deliveryID)
	fmt.Println("CloudFront access logs v2 configured!")
	
	return nil
}
