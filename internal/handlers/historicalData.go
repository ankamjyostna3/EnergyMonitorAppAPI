package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

func GetEnergyHistoryHandler(w http.ResponseWriter, r *http.Request) {
	startDate := r.URL.Query().Get("startDate")
	endDate := r.URL.Query().Get("endDate")
	userID, err := validateTokenAndGetUserID(r)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	if startDate == "" || endDate == "" || userID == "" {
		http.Error(w, "Missing startDate, endDate, or userID query parameter", http.StatusBadRequest)
		return
	}

	// Query DynamoDB for the specified date range
	input := &dynamodb.QueryInput{
		TableName:              aws.String("EnergyData"),
		KeyConditionExpression: aws.String("#userID = :userID AND #date BETWEEN :startDate AND :endDate"),
		ExpressionAttributeNames: map[string]*string{
			"#userID": aws.String("UserID"),
			"#date":   aws.String("Date"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userID": {
				S: aws.String(userID),
			},
			":startDate": {
				S: aws.String(startDate),
			},
			":endDate": {
				S: aws.String(endDate),
			},
		},
	}

	result, err := db.Query(input)
	if err != nil {
		log.Printf("Failed to query DynamoDB: %v", err)
		http.Error(w, fmt.Sprintf("Failed to query DynamoDB: %v", err), http.StatusInternalServerError)
		return
	}

	var energyData []EnergyData
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &energyData)
	if err != nil {
		log.Printf("Failed to unmarshal DynamoDB result: %v", err)
		http.Error(w, fmt.Sprintf("Failed to unmarshal DynamoDB result: %v", err), http.StatusInternalServerError)
		return
	}

	responseBody, err := json.Marshal(energyData)
	if err != nil {
		log.Printf("Failed to marshal response body: %v", err)
		http.Error(w, fmt.Sprintf("Failed to marshal response body: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBody)
}
