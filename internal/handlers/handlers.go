package handlers

import (
	"fmt"
	"log"
	"net/http"
)

// Placeholder handler for /health (returns a simple status for Start8 checks)
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Received request: %s %s", r.Method, r.URL.Path)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, `{"status": "ok"}`)
}

// Placeholder handler for /api/sync (will trigger Garmin pull later)
func SyncHandler(w http.ResponseWriter, r *http.Request) {
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

// Placeholder handler for /api/activities (will list/view data later)
func ActivitiesHandler(w http.ResponseWriter, r *http.Request) {
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
