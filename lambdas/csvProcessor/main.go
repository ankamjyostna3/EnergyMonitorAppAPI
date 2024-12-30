package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var (
	db       *dynamodb.DynamoDB
	s3Client *s3.S3
)

func init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))
	db = dynamodb.New(sess)
	s3Client = s3.New(sess)
}

type InputPayload struct {
	S3URL  string `json:"s3_url"`
	UserID string `json:"user_id"`
}

type Response struct {
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

func handler(ctx context.Context, payload InputPayload) (Response, error) {
	log.Println("Handler started with payload:", payload)

	// Parse the S3 URL
	s3URL, err := url.Parse(payload.S3URL)
	if err != nil {
		log.Printf("Failed to parse S3 URL: %v", err)
		return Response{Message: "Failed to parse S3 URL", Error: err.Error()}, err
	}

	// Extract the bucket name from the host
	hostParts := strings.Split(s3URL.Host, ".")
	if len(hostParts) < 3 {
		log.Printf("Invalid S3 URL: %v", s3URL)
		return Response{Message: "Invalid S3 URL", Error: "Invalid S3 URL"}, fmt.Errorf("invalid S3 URL")
	}
	bucket := hostParts[0]
	key := s3URL.Path[1:] // Remove the leading '/'

	log.Printf("Bucket: %s, Key: %s", bucket, key)

	// Get the object from S3
	buff := &aws.WriteAtBuffer{}
	downloader := s3manager.NewDownloaderWithClient(s3Client)
	_, err = downloader.Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		return Response{Message: "Failed to download file", Error: err.Error()}, err
	}

	log.Println("Successfully downloaded file from S3:", payload.S3URL)

	// Process the CSV file
	r := csv.NewReader(bytes.NewReader(buff.Bytes()))
	records, err := r.ReadAll()
	if err != nil {
		log.Printf("Failed to read CSV file: %v", err)
		return Response{Message: "Failed to read CSV file", Error: err.Error()}, err
	}

	for _, record := range records {
		// Assuming the CSV columns are: date, usage
		date := record[0]
		energy, err := strconv.ParseFloat(record[1], 64)
		if err != nil || energy < 0 {
			log.Printf("Failed to convert energy to non-negative float: %v", err)
			return Response{Message: "Failed to convert energy to non-negative float", Error: err.Error()}, err
		}

		// Save to DynamoDB
		item := map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(payload.UserID),
			},
			"Date": {
				S: aws.String(date),
			},
			"Energy": {
				N: aws.String(fmt.Sprintf("%f", energy)),
			},
		}
		_, err = db.PutItem(&dynamodb.PutItemInput{
			TableName: aws.String("EnergyData"),
			Item:      item,
		})
		if err != nil {
			log.Printf("Failed to save record to DynamoDB: %v", err)
			return Response{Message: "Failed to save record to DynamoDB", Error: err.Error()}, err
		}

		log.Println("Successfully saved record to DynamoDB:", item)
	}

	log.Println("Handler completed successfully")
	return Response{Message: "Handler completed successfully"}, nil
}

func main() {
	lambda.Start(handler)
}