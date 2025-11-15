package main

import (
	"log"
	"net/http"

	"github.com/gratten/ownpath/internal/handlers" // Adjust based on your module name
	// Adjust based on your module name
)

// Global error handler middleware (wraps all handlers)
func withLoggingAndErrorHandling(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Panic recovered: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(w, r)
	}
}

func main() {
	// Set up routes with logging and error handling
	http.HandleFunc("/health", withLoggingAndErrorHandling(handlers.HealthHandler))
	http.HandleFunc("/api/activities", withLoggingAndErrorHandling(handlers.ActivitiesHandler))
	http.HandleFunc("/api/sync", withLoggingAndErrorHandling(handlers.SyncHandler))

	// Start the server
	port := ":8080" // Default port; can be configured later for Start9
	log.Printf("Starting server on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
