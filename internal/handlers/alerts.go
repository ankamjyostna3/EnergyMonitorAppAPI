package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var svc *dynamodb.DynamoDB

func init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))
	svc = dynamodb.New(sess)
}

type Alert struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

type UserThreshold struct {
	UserID    string  `json:"UserID"`
	Threshold float64 `json:"Threshold"`
}

func GetAlertsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := validateTokenAndGetUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	result, err := svc.GetItem(&dynamodb.GetItemInput{
		TableName: aws.String("UserThresholds"),
		Key: map[string]*dynamodb.AttributeValue{
			"UserID": {
				S: aws.String(userID),
			},
		},
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var userThreshold UserThreshold
	err = dynamodbattribute.UnmarshalMap(result.Item, &userThreshold)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(userThreshold)
}

func UpdateAlertsHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := validateTokenAndGetUserID(r)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	var input struct {
		Threshold float64 `json:"threshold"`
	}
	err = json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	userThreshold := UserThreshold{
		UserID:    userID,
		Threshold: input.Threshold,
	}

	av, err := dynamodbattribute.MarshalMap(userThreshold)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = svc.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("UserThresholds"),
		Item:      av,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
