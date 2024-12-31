package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"EnergyMonitorAppAPI/internal/services"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Token   string `json:"token,omitempty"`
}

var (
	cognitoClient *cognitoidentityprovider.CognitoIdentityProvider
)

func init() {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("us-west-2"),
	}))
	cognitoClient = cognitoidentityprovider.New(sess)
}

func HandleSignup(writer http.ResponseWriter, request *http.Request) {
	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	err := json.NewDecoder(request.Body).Decode(&user)
	if err != nil {
		http.Error(writer, "Invalid request payload", http.StatusBadRequest)
		return
	}

	err = services.SignUpUser(user.Username, user.Password, user.Email)
	if err != nil {
		http.Error(writer, "Failed to sign up user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]string{"message": "Signup successful for user " + user.Username}
	json.NewEncoder(writer).Encode(response)
}

func HandleLogout(w http.ResponseWriter, r *http.Request) {
	_, err := validateTokenAndGetUserID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	tokenString := r.Header.Get("Authorization")

	// Remove 'Bearer ' prefix
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	// Sign out the user by invalidating the access token
	input := &cognitoidentityprovider.GlobalSignOutInput{
		AccessToken: aws.String(tokenString),
	}

	_, err = cognitoClient.GlobalSignOut(input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to sign out: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"message": "Successfully signed out",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func HandleLogin(writer http.ResponseWriter, request *http.Request) {
	var user struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	err := json.NewDecoder(request.Body).Decode(&user)
	if err != nil {
		response := Response{}
		response.Success = false
		response.Error = "Invalid request payload"
		writer.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(writer).Encode(response)
		return
	}

	token, err := services.SignInUser(user.Username, user.Password)
	if err != nil {
		response := Response{}
		response.Success = false
		response.Error = "Failed to sign in user: " + err.Error()
		writer.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(writer).Encode(response)
		return
	}

	response := Response{}
	response.Success = true
	response.Message = "Login successful"
	response.Token = token
	writer.WriteHeader(http.StatusOK)
	json.NewEncoder(writer).Encode(response)
}
