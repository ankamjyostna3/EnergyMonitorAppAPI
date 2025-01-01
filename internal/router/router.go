package router

import (
	"EnergyMonitorAppAPI/internal/handlers"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func SetupRouter() http.Handler {
	router := mux.NewRouter()

	//handle Authentication
	router.HandleFunc("/auth/login", handlers.HandleLogin).Methods("POST")
	router.HandleFunc("/auth/signup", handlers.SignupHandler).Methods("POST")
	router.HandleFunc("/auth/signout", handlers.HandleLogout).Methods("POST")

	//handle Energy inputs
	router.HandleFunc("/energy/input", handlers.SaveEnergyDataHandler).Methods("POST")
	router.HandleFunc("/energy/upload", handlers.UploadEnergyDataHandler).Methods("POST")
	router.HandleFunc("/energy/upload", handlers.GetPresignedURLHandler).Methods("GET")

	// Handle Energy history
	router.HandleFunc("/energy/history", handlers.GetEnergyHistoryHandler).Methods("GET")
	router.HandleFunc("/energy/summary", handlers.GetEnergySummaryHandler).Methods("GET") // New route

	// Handle Alerts
	router.HandleFunc("/alerts", handlers.GetAlertsHandler).Methods("GET")
	router.HandleFunc("/alerts", handlers.UpdateAlertsHandler).Methods("POST")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081"}, // Allow your frontend's origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	return c.Handler(router)
}
