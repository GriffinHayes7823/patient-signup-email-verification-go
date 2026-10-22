package main

import (
	"log"
	"net/http"
	"os"

	"verify-healthtech-signup/internal/signup"
)

func main() {
	client, err := signup.NewClient(os.Getenv("INFRAI_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	publicURL := os.Getenv("PUBLIC_BASE_URL")
	if publicURL == "" {
		publicURL = "http://localhost:8080"
	}
	service := signup.NewVerificationService(client, client, publicURL)
	log.Println("signup verification service listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", signup.NewHandler(service)))
}
