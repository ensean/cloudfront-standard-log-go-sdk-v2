package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

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
	sourceName := "S3-delivery"
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
	fmt.Println("Successfully created delivery source")

	// Step 2: Create delivery destination (S3 bucket)
	destinationName := "S3-destination"
	bucketArn := fmt.Sprintf("arn:aws:s3:::%s", *bucketName)
	
	destinationInput := &cloudwatchlogs.PutDeliveryDestinationInput{
		Name: aws.String(destinationName),
		DeliveryDestinationConfiguration: &types.DeliveryDestinationConfiguration{
			DestinationResourceArn: aws.String(bucketArn),
		},
		OutputFormat: types.OutputFormatParquet,
	}
	
	_, err = logsClient.PutDeliveryDestination(context.TODO(), destinationInput)
	if err != nil {
		log.Fatalf("Failed to create delivery destination: %v", err)
	}
	fmt.Println("Successfully created delivery destination")

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
		},
		S3DeliveryConfiguration: &types.S3DeliveryConfiguration{
			SuffixPath: aws.String("{DistributionId}/{yyyy}/{MM}/{dd}/{HH}/"),	// change the path as needed
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
