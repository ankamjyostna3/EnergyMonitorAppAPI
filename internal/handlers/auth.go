package handlers

import (
	"encoding/json"
	"net/http"

	"EnergyMonitorAppAPI/internal/services"
)

type Response struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	Token   string `json:"token,omitempty"`
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
func HandleLogout(writer http.ResponseWriter, request *http.Request) {
	// Here, you would clear the session or cookie if stored.
	http.Redirect(writer, request, "/", http.StatusFound)
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
