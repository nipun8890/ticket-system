package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"ticketsystem/internal/handlers"
	"ticketsystem/internal/middleware"
	"ticketsystem/internal/store"
)

func main() {
	port := getEnv("PORT", "8080")
	jwtSecret := getEnv("JWT_SECRET", "")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-me"
		log.Println("WARNING: JWT_SECRET not set, using an insecure default for local development only")
	}

	s := store.New()
	h := handlers.New(s, jwtSecret)
	authRequired := middleware.RequireAuth(jwtSecret)

	mux := http.NewServeMux()

	// Public endpoints.
	mux.HandleFunc("GET /health", h.Health)
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)

	// Protected endpoints.
	mux.Handle("POST /tickets", authRequired(http.HandlerFunc(h.CreateTicket)))
	mux.Handle("GET /tickets", authRequired(http.HandlerFunc(h.ListTickets)))
	mux.Handle("GET /tickets/{id}", authRequired(http.HandlerFunc(h.GetTicket)))
	mux.Handle("PATCH /tickets/{id}/status", authRequired(http.HandlerFunc(h.UpdateTicketStatus)))

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      logRequests(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("ticket-system listening on :%s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
