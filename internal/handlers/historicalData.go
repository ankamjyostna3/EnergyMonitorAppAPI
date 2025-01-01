package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

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

func GetEnergySummaryHandler(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	UserID, err := validateTokenAndGetUserID(r)
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	if period == "" || UserID == "" {
		http.Error(w, "Missing period or userID query parameter", http.StatusBadRequest)
		return
	}

	// Get the date range based on the period
	startDate, endDate := getDateRange(period)

	// Query DynamoDB for the specified date range
	input := &dynamodb.QueryInput{
		TableName:              aws.String("EnergyData"),
		KeyConditionExpression: aws.String("#UserID = :UserID AND #Date BETWEEN :startDate AND :endDate"),
		ExpressionAttributeNames: map[string]*string{
			"#UserID": aws.String("UserID"),
			"#Date":   aws.String("Date"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":UserID": {
				S: aws.String(UserID),
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
		log.Printf("Failed to query EnergyData: %v", err)
		http.Error(w, "Failed to get energy summary", http.StatusInternalServerError)
		return
	}

	var energyData []EnergyData
	err = dynamodbattribute.UnmarshalListOfMaps(result.Items, &energyData)
	if err != nil {
		log.Printf("Failed to unmarshal EnergyData: %v", err)
		http.Error(w, "Failed to get energy summary", http.StatusInternalServerError)
		return
	}

	// Calculate trends based on the energy data
	trends := calculateTrends(energyData, period)

	response := map[string]interface{}{
		"period": period,
		"trends": trends,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func getDateRange(period string) (string, string) {
	now := time.Now()
	var startDate, endDate string

	switch period {
	case "daily":
		startDate = now.Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	case "weekly":
		startDate = now.AddDate(0, 0, -7).Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	case "monthly":
		startDate = now.AddDate(0, -1, 0).Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	default:
		startDate = now.Format("2006-01-02")
		endDate = now.Format("2006-01-02")
	}

	return startDate, endDate
}

func calculateTrends(data []EnergyData, period string) map[string]interface{} {
	trends := make(map[string]interface{})
	totalEnergy := 0.0
	dailyTrends := make(map[string]float64)

	for _, record := range data {
		totalEnergy += record.Energy
		date := record.Date
		if _, exists := dailyTrends[date]; !exists {
			dailyTrends[date] = 0
		}
		dailyTrends[date] += record.Energy
	}

	trends["totalEnergy"] = totalEnergy
	trends["dailyTrends"] = dailyTrends

	if period == "weekly" || period == "monthly" {
		weeklyTrends := make(map[string]float64)
		monthlyTrends := make(map[string]float64)

		for date, energy := range dailyTrends {
			parsedDate, _ := time.Parse("2006-01-02", date)
			year, week := parsedDate.ISOWeek()
			month := parsedDate.Format("2006-01")

			weekKey := fmt.Sprintf("%d-W%d", year, week)
			monthKey := month

			if _, exists := weeklyTrends[weekKey]; !exists {
				weeklyTrends[weekKey] = 0
			}
			weeklyTrends[weekKey] += energy

			if _, exists := monthlyTrends[monthKey]; !exists {
				monthlyTrends[monthKey] = 0
			}
			monthlyTrends[monthKey] += energy
		}

		trends["weeklyTrends"] = weeklyTrends
		trends["monthlyTrends"] = monthlyTrends
	}

	return trends
}
