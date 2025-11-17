package main

import (
	"log"
	"net/http"

	"github.com/gratten/ownpath/internal/db"
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

// // Helper to detect if this is a redirected request (prevents loops; optional)
// func isRedirected(r *http.Request) bool {
// 	// Simple check: look for a query param set during redirect
// 	return r.URL.Query().Get("redirected") == "true"
// }

func main() {
	// Initialize the database (this creates/opens ownpath.db and sets up the schema)
	if err := db.InitDB("./ownpath.db"); err != nil { // Adjust path if needed (e.g., for Docker/Start9)
		log.Fatalf("Failed to initialize database: %v", err) // Crash if init fails
	}
	defer db.CloseDB() // Ensure the DB closes cleanly on exit

	// Serve static files from /web (no redirect; directly serve index.html at root)
	fs := http.FileServer(http.Dir("./web"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			// Directly serve index.html for root or explicit /index.html (avoids any loop)
			http.ServeFile(w, r, "./web/index.html")
			return
		}
		// Serve other files (strips leading /)
		http.StripPrefix("/", fs).ServeHTTP(w, r)
	})

	// Set up routes with logging and error handling
	// http.HandleFunc("/health", withLoggingAndErrorHandling(handlers.HealthHandler))
	http.HandleFunc("/api/activities", withLoggingAndErrorHandling(handlers.ActivitiesHandler))
	// http.HandleFunc("/api/sync", withLoggingAndErrorHandling(handlers.SyncHandler))
	http.HandleFunc("/api/upload", handlers.UploadHandler)

	// Start the server
	port := ":8080" // Default port; can be configured later for Start9
	log.Printf("Starting server on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
