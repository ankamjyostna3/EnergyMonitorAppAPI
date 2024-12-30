package router

import (
	"EnergyMonitorAppAPI/internal/handlers"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

func SetupRouter() http.Handler {
	router := mux.NewRouter()

	//handles login
	router.HandleFunc("/login", handlers.HandleLogin).Methods("POST")
	router.HandleFunc("/logout", handlers.HandleLogout).Methods("GET")
	router.HandleFunc("/signup", handlers.HandleSignup).Methods("POST")

	//handle Energy inputs
	router.HandleFunc("/energy/input", handlers.SaveEnergyDataHandler).Methods("POST")
	router.HandleFunc("/energy/upload", handlers.UploadEnergyDataHandler).Methods("POST")
	router.HandleFunc("/energy/upload", handlers.GetPresignedURLHandler).Methods("GET")

	// Handle Energy history
	router.HandleFunc("/energy/history", handlers.GetEnergyHistoryHandler).Methods("GET")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081"}, // Allow your frontend's origin
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	return c.Handler(router)
}
