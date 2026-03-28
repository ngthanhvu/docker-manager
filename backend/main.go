package main

import (
	"docker-ui/auth"
	"log"
	"net/http"
	"docker-ui/docker"
	"docker-ui/ws"
)

var authService *auth.Service

func main() {
	// Initialize Docker client
	if err := docker.Init(); err != nil {
		log.Fatalf("Failed to initialize Docker client: %v", err)
	}

	var err error
	authService, err = auth.Init("")
	if err != nil {
		log.Fatalf("Failed to initialize auth service: %v", err)
	}
	defer authService.Close()

	ws.RequestAuthorizer = authService.AuthorizeRequest

	docker.GetDashboardMetrics(1)

	router := SetupRouter()

	// Add CORS middleware
	corsRouter := corsMiddleware(router)

	log.Println("Server started on :8080")
	if err := http.ListenAndServe(":8080", corsRouter); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
