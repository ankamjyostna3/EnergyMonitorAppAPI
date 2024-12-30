package main

import (
	"log"
	"net/http"

	"EnergyMonitorAppAPI/internal/router"
	"EnergyMonitorAppAPI/pkg/config"
)

func main() {
	err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	r := router.SetupRouter()

	log.Println("Server is running on port 8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
