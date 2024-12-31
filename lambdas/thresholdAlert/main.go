package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/sns"
)

var (
	svcDynamoDB *dynamodb.DynamoDB
	svcSNS      *sns.SNS
	svcCognito  *cognitoidentityprovider.CognitoIdentityProvider
)

func init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))
	svcDynamoDB = dynamodb.New(sess)
	svcSNS = sns.New(sess)
	svcCognito = cognitoidentityprovider.New(sess)
}

type EnergyData struct {
	UserID string  `json:"UserID"`
	Date   string  `json:"Date"`
	Energy float64 `json:"Energy"`
}

type UserThreshold struct {
	UserID    string  `json:"UserID"`
	Threshold float64 `json:"Threshold"`
}

type Payload struct {
	UserID string `json:"UserID"`
}

func getEmailFromCognito(userID string) (string, error) {
	fmt.Println("In get email from cognito")
	input := &cognitoidentityprovider.AdminGetUserInput{
		UserPoolId: aws.String("us-west-2_QMNqg83TF"),
		Username:   aws.String(userID),
	}
	fmt.Println("input", input)

	result, err := svcCognito.AdminGetUser(input)
	fmt.Println("result", result)
	if err != nil {
		return "", err
	}

	for _, attr := range result.UserAttributes {
		if *attr.Name == "email" {
			return *attr.Value, nil
		}
	}
	fmt.Println("email not found")

	return "", fmt.Errorf("email not found for user %s", userID)
}

func alertUser(UserID string) {
	log.Printf("Alerting user %s...", UserID)
	// Define the current date to check
	currentDate := time.Now().Format("2006-01-02")

	// Query the EnergyData table for the specified date and user
	result, err := svcDynamoDB.Query(&dynamodb.QueryInput{
		TableName:              aws.String("EnergyData"),
		KeyConditionExpression: aws.String("#UserID = :UserID AND #Date = :Date"),
		ExpressionAttributeNames: map[string]*string{
			"#UserID": aws.String("UserID"),
			"#Date":   aws.String("Date"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":UserID": {
				S: aws.String(UserID),
			},
			":Date": {
				S: aws.String(currentDate),
			},
		},
	})
	if err != nil {
		log.Printf("Failed to query EnergyData for user %s: %v", UserID, err)
		return
	}

	if len(result.Items) == 0 {
		log.Printf("No energy data found for user %s on date %s", UserID, currentDate)
		return
	}
	fmt.Println(result.Items[0])
	fmt.Print("going to threshold DB data")

	var energyData EnergyData
	err = dynamodbattribute.UnmarshalMap(result.Items[0], &energyData)
	if err != nil {
		log.Printf("Failed to unmarshal EnergyData for user %s: %v", UserID, err)
		return
	}

	// Get the user's threshold from UserThresholds table
	thresholdResult, err := svcDynamoDB.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("UserThresholds"),
		Key: map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(UserID),
			},
		},
	})
	if err != nil {
		log.Printf("Failed to get UserThreshold for user %s: %v", UserID, err)
		return
	}
	if thresholdResult.Item == nil {
		log.Printf("No threshold found for user %s", UserID)
		return
	}

	var userThreshold UserThreshold
	err = dynamodbattribute.UnmarshalMap(thresholdResult.Item, &userThreshold)
	if err != nil {
		log.Printf("Failed to unmarshal UserThreshold for user %s: %v", UserID, err)
		return
	}
	fmt.Println("user threshold", userThreshold)

	fmt.Print("usage:", energyData.Energy)
	fmt.Print("userThreshold.Threshold:", userThreshold.Threshold)
	// Check if usage exceeds threshold
	if energyData.Energy > userThreshold.Threshold {
		// Get the user's email from Cognito
		email, err := getEmailFromCognito(UserID)
		fmt.Print("email", email)
		if err != nil {
			log.Printf("Failed to get email for user %s: %v", UserID, err)
			return
		}

		// Send notification via SNS
		message := fmt.Sprintf("Dear user, your energy usage on %s was %.2f, which exceeds your threshold of %.2f.", currentDate, energyData.Energy, userThreshold.Threshold)
		_, err = svcSNS.Publish(&sns.PublishInput{
			Message:  aws.String(message),
			Subject:  aws.String("Energy Usage Alert"),
			TopicArn: aws.String("arn:aws:sns:us-west-2:985539767168:thresholdAlert"),
			MessageAttributes: map[string]*sns.MessageAttributeValue{
				"email": {
					DataType:    aws.String("String"),
					StringValue: aws.String(email),
				},
			},
		})
		if err != nil {
			log.Printf("Failed to send SNS notification to user %s: %v", UserID, err)
		}
	}
}

func checkEnergyUsage(ctx context.Context) {
	log.Println("Checking energy usage...")
	// Scan the UserThresholds table to get all users
	result, err := svcDynamoDB.Scan(&dynamodb.ScanInput{
		TableName: aws.String("UserThresholds"),
	})
	if err != nil {
		log.Fatalf("Failed to scan UserThresholds table: %v", err)
	}

	var userThresholds []UserThreshold
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &userThresholds)
	if err != nil {
		log.Fatalf("Failed to unmarshal UserThresholds: %v", err)
	}

	// Check each user's usage against their threshold
	for _, userThreshold := range userThresholds {
		alertUser(userThreshold.UserID)
	}
}

func handler(ctx context.Context, payload map[string]string) error {
	log.Printf("Received payload: %v", payload)

	if payload["UserID"] != "" {
		alertUser(payload["UserID"])
	} else {
		checkEnergyUsage(ctx)
	}

	return nil
}

func main() {
	lambda.Start(handler)
}
