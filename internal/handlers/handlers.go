package handlers // Adjust if your package name differs

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid" // For unique IDs

	"github.com/muktihari/fit/decoder"                 // For decoding FIT files
	"github.com/muktihari/fit/profile/mesgdef"         // For typed messages (e.g., NewFileId, NewSession)
	"github.com/muktihari/fit/profile/untyped/mesgnum" // For message numbers (e.g., MesgNumFileId)
	// For core proto types (e.g., Fit, Mesg)
)

// getSportFormatted is a helper to convert FIT sport ID to a string.
func getSportFormatted(sport byte) string {
	switch sport {
	case 1:
		return "Running"
	case 2:
		return "Cycling"
	case 5:
		return "Walking" // Or Hiking; adjust based on your needs
	case 17:
		return "Swimming"
		// Add more from FIT Profile (e.g., 0=Generic, 11=Hiking if needed)
	default:
		return "Unknown"
	}
}

// UploadHandler handles FIT file uploads, parsing, and basic processing.
func UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit upload size to 10MB
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20) // 10 MB
	err := r.ParseMultipartForm(10 << 20)           // Parse form with same limit
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Get the uploaded file (assuming form field name "fit_file")
	file, header, err := r.FormFile("fit_file")
	if err != nil {
		http.Error(w, "Failed to get file: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Use header for validation and logging
	if !strings.HasSuffix(header.Filename, ".fit") {
		http.Error(w, "Only .fit files are allowed", http.StatusBadRequest)
		return
	}
	log.Printf("Uploaded file: %s (size: %d bytes)", header.Filename, header.Size)

	// Read the file into a buffer for parsing
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		http.Error(w, "Failed to read file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Parse the FIT file
	dec := decoder.New(buf)
	fit, err := dec.Decode()
	if err != nil {
		http.Error(w, "Failed to decode FIT file: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Extract key messages (loop through all messages)
	var fileID *mesgdef.FileId
	var session *mesgdef.Session
	var recordCount int
	for i := range fit.Messages {
		mesg := &fit.Messages[i] // Reference to the message
		switch mesg.Num {
		case mesgnum.FileId:
			fileID = mesgdef.NewFileId(mesg)
		case mesgnum.Session:
			session = mesgdef.NewSession(mesg)
		case mesgnum.Record:
			recordCount++
		}
		// Note: If developer fields are present (e.g., in mesg.DeveloperFields), you can handle them here for future expansion.
	}

	// Validate required messages
	if fileID == nil {
		http.Error(w, "No FileId message found in activity", http.StatusBadRequest)
		return
	}
	if session == nil {
		http.Error(w, "No Session message found in activity", http.StatusBadRequest)
		return
	}

	// Generate a unique ID (e.g., UUID)
	activityID := uuid.New().String()

	// Extract key data (customize this based on what you need)
	parsedData := struct {
		ID          string
		Type        string // e.g., "running", "hiking"
		Timestamp   int64
		Distance    float64 // in meters
		Elevation   float64 // total ascent in meters
		RecordCount int     // Number of data points (for GPX-like tracks)
		// Add more fields as needed, e.g., for full GPX: extract Record messages with lat/long/alt/time
	}{
		ID:          activityID,
		Type:        getSportFormatted(byte(session.Sport)),
		Timestamp:   fileID.TimeCreated.Unix(),              // FIT timestamp (uint32); convert to int64 for Unix time if needed
		Distance:    float64(session.TotalDistance) / 100.0, // FIT scale: uint32 value / 100 = meters
		Elevation:   float64(session.TotalAscent),           // uint16 value is already in meters
		RecordCount: recordCount,
	}

	// TODO: Store in DB (Stub for now - replace with Step 4 implementation)
	// For MVP testing, just log it
	log.Printf("Parsed Activity: %+v", parsedData)
	// Example stub: Save to a simple in-memory map or file if you want to test without DB
	// In Step 4, you'll insert into SQLite, e.g., serialize to JSON for the `stats_json` column

	// Respond with success (e.g., JSON with ID for frontend to use)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "success", "activity_id": "%s", "message": "Activity uploaded and parsed"}`, activityID)
}
