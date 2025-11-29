package main

import (
	"context"
	"encoding/json"
	"github.com/redis/go-redis/v9"
	"log"
	"net/http"
	"os"
	"time"
)

type response struct {
	Visits int64  `json:"visits"`
	Pod    string `json:"pod"`
}

var (
	rdb   *redis.Client
	podID string
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	hostname, err := os.Hostname()
	if err != nil {
		podID = "unknown"
	} else {
		podID = hostname
	}

	addr := getenv("REDIS_ADDR", "localhost:6379")
	password := os.Getenv("REDIS_PASSWORD")
	appPort := getenv("APP_PORT", "8080")

	log.Printf("pod=%s redis_addr=%s app_port=%s\n", podID, addr, appPort)

	rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to ping redis: %v", err)
	}

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/favicon.ico", handleFavicon)

	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Starting HTTP server on port", appPort)

	if err := http.ListenAndServe(":"+appPort, nil); err != nil {
		log.Fatal(err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	visits, err := rdb.Incr(ctx, "visits_total").Result()
	if err != nil {
		log.Printf("failed to INCR visits_total: %v", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response{
		Visits: visits,
		Pod:    podID,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		http.Error(w, "redis not available", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func handleFavicon(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
