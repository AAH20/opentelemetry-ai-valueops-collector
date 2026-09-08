package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/AAH20/opentelemetry-ai-valueops-collector/internal/value"
)

func main() {
	apiKey := os.Getenv("VALUEOPS_API_KEY")
	if apiKey == "" {
		log.Fatal("VALUEOPS_API_KEY is required")
	}
	var rates []value.RateCard
	file, err := os.Open(env("VALUEOPS_RATE_CARDS", "config/rate-cards.json"))
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	if err := json.NewDecoder(file).Decode(&rates); err != nil {
		log.Fatal(err)
	}
	engine, err := value.NewEngine(rates)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: env("VALUEOPS_LISTEN", ":8080"), Handler: value.API{Engine: engine, APIKey: apiKey}.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("AI ValueOps collector listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
