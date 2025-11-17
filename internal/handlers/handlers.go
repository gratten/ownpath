package handlers // Assuming this is internal/handlers; adjust if needed

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid" // For unique IDs
	"github.com/gratten/ownpath/internal/db"
	"github.com/gratten/ownpath/internal/models" // Adjust import path
	"github.com/gratten/ownpath/internal/utils"
	"github.com/muktihari/fit/decoder"                 // For decoding FIT files
	"github.com/muktihari/fit/profile/mesgdef"         // For typed messages (e.g., NewFileId, NewSession)
	"github.com/muktihari/fit/profile/untyped/mesgnum" // For message numbers (e.g., MesgNumFileId)
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
	// NEW: Slice to hold track points for GPX generation
	var points []struct {
		Lat  float64
		Long float64
		Ele  float64
		Time int64
	}
	for i := range fit.Messages {
		mesg := &fit.Messages[i] // Reference to the message
		switch mesg.Num {
		case mesgnum.FileId:
			fileID = mesgdef.NewFileId(mesg)
		case mesgnum.Session:
			session = mesgdef.NewSession(mesg)
		case mesgnum.Record:
			record := mesgdef.NewRecord(mesg)
			// Check for invalid position values (per FIT spec: 0x7FFFFFFF for signed int32)
			if record.PositionLat == 0x7FFFFFFF || record.PositionLong == 0x7FFFFFFF {
				continue // Skip invalid points
			}
			// Convert semicircles to degrees (standard FIT conversion)
			lat := float64(record.PositionLat) / 11930465.0
			long := float64(record.PositionLong) / 11930465.0
			var ele float64
			// Check for invalid altitude (per FIT spec: 0xFFFF for uint16)
			if record.Altitude != 0xFFFF {
				ele = (float64(record.Altitude) / 5.0) - 500.0 // FIT altitude encoding
			}
			// Timestamp is time.Time; convert to Unix int64
			ts := record.Timestamp.Unix()
			points = append(points, struct {
				Lat  float64
				Long float64
				Ele  float64
				Time int64
			}{lat, long, ele, ts})
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
		Points      []struct {
			Lat  float64
			Long float64
			Ele  float64
			Time int64
		} // NEW: Added for track points
	}{
		ID:          activityID,
		Type:        getSportFormatted(byte(session.Sport)),
		Timestamp:   fileID.TimeCreated.Unix(),              // FIT timestamp (uint32); convert to int64 for Unix time if needed
		Distance:    float64(session.TotalDistance) / 100.0, // FIT scale: uint32 value / 100 = meters
		Elevation:   float64(session.TotalAscent),           // uint16 value is already in meters
		RecordCount: recordCount,
		Points:      points, // NEW: Assign extracted points
	}
	// TODO: Store in DB (Stub for now - replace with Step 4 implementation)
	// For MVP testing, just log it
	log.Printf("Parsed Activity: %+v", parsedData)
	// Example stub: Save to a simple in-memory map or file if you want to test without DB
	// In Step 4, you'll insert into SQLite, e.g., serialize to JSON for the `stats_json` column
	// Example stats serialization (adjust based on your actual parsedData fields)
	statsMap := map[string]any{
		"distance":    parsedData.Distance,
		"elevation":   parsedData.Elevation,
		"recordCount": parsedData.RecordCount,
		// Add more fields as needed
	}
	stats, err := json.Marshal(statsMap)
	if err != nil {
		// Handle error (e.g., log and return HTTP 500)
		http.Error(w, "Failed to serialize stats", http.StatusInternalServerError)
		return
	}
	activity := models.Activity{
		ID:        parsedData.ID,                      // Or generate a new one if needed: uuid.New().String()
		Timestamp: time.Unix(parsedData.Timestamp, 0), // Convert int64 Unix timestamp to time.Time
		Type:      parsedData.Type,
		StatsJSON: string(stats),
		GPXData:   utils.GenerateGPXFromFIT(parsedData), // This now works with the updated parsedData
	}
	if err := db.InsertActivity(activity); err != nil {
		// Handle error (e.g., log and return HTTP 500)
		http.Error(w, "Failed to store activity", http.StatusInternalServerError)
		return
	}
	// Respond with success (e.g., JSON with ID for frontend to use)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status": "success", "activity_id": "%s", "message": "Activity uploaded and parsed"}`, activityID)
}
