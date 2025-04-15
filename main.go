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
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

func main() {
	// Define command line flags
	distributionID := flag.String("distribution-id", "", "CloudFront distribution ID (required)")
	bucketName := flag.String("bucket-name", "", "S3 bucket name for logs (required)")
	region := flag.String("region", "us-east-1", "AWS region")
	flag.Parse()

	// Validate required parameters
	if *distributionID == "" || *bucketName == "" {
		fmt.Println("Error: distribution-id and bucket-name are required")
		flag.Usage()
		os.Exit(1)
	}

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(*region))
	if err != nil {
		log.Fatalf("unable to load SDK config: %v", err)
	}

	// Create CloudWatch Logs client
	logsClient := cloudwatchlogs.NewFromConfig(cfg)

	// Get account ID
	accountID := getAccountID(cfg)

	// Step 1: Create delivery source (CloudFront distribution)
	sourceName := fmt.Sprintf("CloudFront-%s", *distributionID)
	distributionArn := fmt.Sprintf("arn:aws:cloudfront::%s:distribution/%s", accountID, *distributionID)
	sourceInput := &cloudwatchlogs.PutDeliverySourceInput{
		Name:        aws.String(sourceName),
		ResourceArn: aws.String(distributionArn),
		LogType:     aws.String("ACCESS_LOGS"),
	}
	
	_, err = logsClient.PutDeliverySource(context.TODO(), sourceInput)
	if err != nil {
		log.Fatalf("Failed to create delivery source: %v", err)
	}
	fmt.Printf("Successfully created delivery source: %s\n", sourceName)

	// Step 2: Create delivery destination (S3 bucket)
	destinationName := fmt.Sprintf("S3-destination-cloudfrontlogs-%s", *distributionID)
	bucketArn := fmt.Sprintf("arn:aws:s3:::%s", *bucketName)
	
	destinationInput := &cloudwatchlogs.PutDeliveryDestinationInput{
		Name: aws.String(destinationName),
		DeliveryDestinationConfiguration: &types.DeliveryDestinationConfiguration{
			DestinationResourceArn: aws.String(bucketArn),
		},
		OutputFormat: types.OutputFormatPlain,
	}
	
	_, err = logsClient.PutDeliveryDestination(context.TODO(), destinationInput)
	if err != nil {
		log.Fatalf("Failed to create delivery destination: %v", err)
	}
	fmt.Printf("Successfully created delivery destination: %s\n", destinationName)

	// Step 3: Create delivery (connect source to destination)
	destinationArn := fmt.Sprintf("arn:aws:logs:%s:%s:delivery-destination:%s", 
		*region, accountID, destinationName)
	
	// Create the delivery with the proper configuration
	// Now we can use S3DeliveryConfiguration directly with the updated SDK
	createDeliveryInput := &cloudwatchlogs.CreateDeliveryInput{
		DeliverySourceName:     aws.String(sourceName),
		DeliveryDestinationArn: aws.String(destinationArn),
		Tags: map[string]string{
			"Service": "CloudFront",
			"Created": time.Now().Format(time.RFC3339),
		},
		S3DeliveryConfiguration: &types.S3DeliveryConfiguration{
			SuffixPath:            aws.String("{DistributionId}/{yyyy}/{MM}/{dd}/{HH}/"),
			EnableHiveCompatiblePath: aws.Bool(true),
		},
	}
	
	// Create the delivery
	deliveryResp, err := logsClient.CreateDelivery(context.TODO(), createDeliveryInput)
	if err != nil {
		log.Fatalf("Failed to create delivery: %v", err)
	}
	
	deliveryID := *deliveryResp.Delivery.Id
	fmt.Printf("Successfully created delivery with ID: %s\n", deliveryID)
	fmt.Println("CloudFront access logs v2 configured!")
	
	// Print summary for reference
	fmt.Println("\nConfiguration Summary:")
	fmt.Printf("- Distribution ID: %s\n", *distributionID)
	fmt.Printf("- S3 Bucket: %s\n", *bucketName)
	fmt.Printf("- Delivery Source: %s\n", sourceName)
	fmt.Printf("- Delivery Destination: %s\n", destinationName)
	fmt.Printf("- Delivery ID: %s\n", deliveryID)
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
