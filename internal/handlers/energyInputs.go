package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"EnergyMonitorAppAPI/pkg/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/aws/aws-sdk-go/service/lambda"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/jwk"
)

type EnergyData struct {
	UserID string  `json:"UserID"`
	Energy float64 `json:"Energy"`
	Date   string  `json:"Date"`
}

type UploadRequest struct {
	S3URL string `json:"s3_url"`
}

var (
	db           *dynamodb.DynamoDB
	s3Client     *s3.S3
	lambdaClient *lambda.Lambda
)

func init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(config.AppConfig.AWS.Region),
	}))
	db = dynamodb.New(sess)

	s3Client = s3.New(sess)
	lambdaClient = lambda.New(sess)
}

func validateTokenAndGetUserID(r *http.Request) (string, error) {
	tokenString := r.Header.Get("Authorization")
	if tokenString == "" {
		return "", fmt.Errorf("missing token")
	}
	// Remove 'Bearer ' prefix if it exists
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return getKey(token, r)
	})
	if err != nil {
		return "", fmt.Errorf("invalid token: %v", err)
	}

	// Extract the user ID from the token claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid token claims")
	}

	userID, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("invalid token claims: missing sub")
	}

	return userID, nil
}

func getKey(token *jwt.Token, r *http.Request) (interface{}, error) {
	jwksURL := "https://cognito-idp.us-west-2.amazonaws.com/" + config.AppConfig.AWS.Cognito.UserPoolID + "/.well-known/jwks.json"
	set, err := jwk.Fetch(r.Context(), jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWK: %v", err)
	}

	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, fmt.Errorf("missing kid in token header")
	}

	key, ok := set.LookupKeyID(keyID)
	if !ok {
		return nil, fmt.Errorf("unable to find key with kid: %v", keyID)
	}

	var rawKey interface{}
	err = key.Raw(&rawKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get raw key: %v", err)
	}

	return rawKey, nil
}

func SaveEnergyDataHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := validateTokenAndGetUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var data EnergyData
	err = json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Set the UserID from the token
	data.UserID = userID

	// Set current date if Date is not provided
	if data.Date == "" {
		data.Date = time.Now().Format(time.RFC3339)
	}

	// Insert the data into the EnergyData table
	err = putEnergyData(data)

	if err != nil {
		response := Response{}
		response.Success = false
		response.Error = fmt.Sprintf("Failed to put item: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	// Check if data.Date is the current date
	currentDate := time.Now().Format("2006-01-02")
	if data.Date == currentDate {
		// Invoke the thresholdAlert Lambda function
		invokeThresholdAlertsLambda(data)
	}

	response := Response{
		Success: true,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func invokeThresholdAlertsLambda(data EnergyData) {
	// Prepare the payload
	payload, err := json.Marshal(map[string]string{"UserID": data.UserID})
	if err != nil {
		log.Printf("Failed to marshal payload: %v", err)
		return
	}

	// Invoke the Lambda function
	input := &lambda.InvokeInput{
		FunctionName: aws.String(config.AppConfig.AWS.Lambda.ThresholdAlertLambda),
		Payload:      payload,
	}

	result, err := lambdaClient.Invoke(input)
	if err != nil {
		log.Printf("Failed to invoke Lambda function: %v", err)
		return
	}

	// Log the result
	log.Printf("Lambda function invoked successfully: %s", result.Payload)
}

func putEnergyData(data EnergyData) error {
	av, err := dynamodbattribute.MarshalMap(data)
	if err != nil {
		return err
	}

	_, err = db.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("EnergyData"),
		Item:      av,
	})
	return err
}

func GetPresignedURLHandler(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	fileName := r.URL.Query().Get("fileName")
	fileType := r.URL.Query().Get("fileType")

	// Add timestamp to the file name
	timestamp := time.Now().Format("20060102150405")
	fileNameWithTimestamp := fmt.Sprintf("%s_%s", timestamp, fileName)

	// Generate a pre-signed URL for file upload
	req, _ := s3Client.PutObjectRequest(&s3.PutObjectInput{
		Bucket:      aws.String(config.AppConfig.AWS.S3.BucketName),
		Key:         aws.String(fileNameWithTimestamp),
		ContentType: aws.String(fileType),
	})

	urlStr, err := req.Presign(15 * time.Minute)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate pre-signed URL: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"url": urlStr,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func UploadEnergyDataHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := validateTokenAndGetUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var req UploadRequest
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if req.S3URL == "" {
		http.Error(w, "Missing s3_url in request payload", http.StatusBadRequest)
		return
	}
	fmt.Println("before invoke req.S3URL", req.S3URL)
	fmt.Println("before invoke userID", userID)

	// Invoke the Lambda function with the S3 URL and userID
	payload := map[string]string{
		"s3_url":  req.S3URL,
		"user_id": userID,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to marshal payload: %v", err), http.StatusInternalServerError)
		return
	}

	input := &lambda.InvokeInput{
		FunctionName: aws.String(config.AppConfig.AWS.Lambda.CSVProcessorLambda),
		Payload:      payloadBytes,
	}

	result, err := lambdaClient.Invoke(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to invoke Lambda function: %v", err), http.StatusInternalServerError)
		return
	}

	// Check the result of the Lambda invocation
	if *result.StatusCode != 200 || result.FunctionError != nil {
		http.Error(w, fmt.Sprintf("Lambda function error: %v", string(result.Payload)), http.StatusInternalServerError)
		return
	}
	fmt.Println("after invoke result", result)
	fmt.Println("after invoke result.Payload", string(result.Payload))
	fmt.Println("after invoke result.StatusCode", result.StatusCode)
	fmt.Println("after invoke result.FunctionError", result.FunctionError)

	response := map[string]interface{}{
		"statusCode": result.StatusCode,
		"payload":    string(result.Payload),
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
