package main

import (
    "fmt"
    "log"
    "net/http"
)

// Placeholder handler for /health (returns a simple status for Start9 checks)
func healthHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Received request: %s %s", r.Method, r.URL.Path)
    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, `{"status": "ok"}`)
}

// Placeholder handler for /api/activities (will list/view data later)
func activitiesHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Received request: %s %s", r.Method, r.URL.Path)
    // For now, just return a placeholder JSON response
    // In later steps, this will query the database
    if r.Method != http.MethodGet {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintln(w, `{"activities": [], "message": "Placeholder - no activities yet"}`)
}

// Placeholder handler for /api/sync (will trigger Garmin pull later)
func syncHandler(w http.ResponseWriter, r *http.Request) {
    log.Printf("Received request: %s %s", r.Method, r.URL.Path)
    // For now, just log and return a success message
    // In Step 3, this will handle OAuth and data ingestion
    if r.Method != http.MethodPost {
        http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
        return
    }
    w.WriteHeader(http.StatusOK)
    fmt.Fprintln(w, `{"message": "Sync triggered (placeholder)"}`)
}

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
    http.HandleFunc("/health", withLoggingAndErrorHandling(healthHandler))
    http.HandleFunc("/api/activities", withLoggingAndErrorHandling(activitiesHandler))
    http.HandleFunc("/api/sync", withLoggingAndErrorHandling(syncHandler))

    // Start the server
    port := ":8080" // Default port; can be configured later for Start9
    log.Printf("Starting server on %s", port)
    if err := http.ListenAndServe(port, nil); err != nil {
        log.Fatalf("Server failed to start: %v", err)
    }
}