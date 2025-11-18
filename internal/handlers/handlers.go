package handlers // Assuming this is internal/handlers; adjust if needed

import (
	"bytes"
	"database/sql"
	"encoding/base64"
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

// ActivityHandler handles GET requests to /api/activity?id=<uuid>
func ActivityHandler(w http.ResponseWriter, r *http.Request) {
	// Extract ID from query params
	id := r.URL.Query().Get("id")
	log.Printf("ActivityHandler called with ID: '%s' (length: %d)", id, len(id)) // Log even if empty
	if id == "" {
		http.Error(w, "Missing ID parameter", http.StatusBadRequest)
		log.Printf("Error: Missing ID in /api/activity request")
		return
	}

	// // Optional: Validate UUID format
	// if _, err := uuid.Parse(id); err != nil {
	// 	http.Error(w, "Invalid ID format", http.StatusBadRequest)
	// 	log.Printf("Error: Invalid UUID %s: %v", id, err)
	// 	return
	// }

	// Query the DB for the activity
	var activity models.Activity
	row := db.DB.QueryRow("SELECT id, timestamp, type, stats_json, gpx_data FROM activities WHERE id = ?", id)
	err := row.Scan(&activity.ID, &activity.Timestamp, &activity.Type, &activity.StatsJSON, &activity.GPXData)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Activity not found", http.StatusNotFound)
			log.Printf("Error: Activity %s not found", id)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		log.Printf("Error querying activity %s: %v", id, err)
		return
	}

	// Unmarshal StatsJSON for display (assuming it's JSON like {"distance": 10.5, "elevation": 200})
	var stats map[string]interface{}
	if err := json.Unmarshal([]byte(activity.StatsJSON), &stats); err != nil {
		log.Printf("Warning: Failed to unmarshal stats for %s: %v", id, err)
		// Fallback: Display raw JSON
		stats = map[string]interface{}{"raw": activity.StatsJSON}
	}

	// ... other code in ActivityHandler ...

	// After querying DB
	log.Printf("Queried GPX length from DB: %d for ID %s", len(activity.GPXData), id)

	// Base64 encode GPX for safe HTML embedding (avoids escaping issues)
	gpxEncoded := base64.StdEncoding.EncodeToString([]byte(activity.GPXData))
	log.Printf("Encoded GPX length: %d", len(gpxEncoded))

	// Build HTML partial
	html := `<div id="activity-details">
		<h2>Activity: ` + activity.Type + ` on ` + activity.Timestamp.Format("2006-01-02 15:04:05") + `</h2>
		<ul>`
	for key, val := range stats {
		html += fmt.Sprintf("<li><strong>%s:</strong> %v</li>", key, val)
	}
	html += `</ul>
		<div id="map" style="height: 400px; width: 100%;"></div>
		<div id="gpx-data" style="display: none;" data-encoded="true">` + gpxEncoded + `</div> <!-- Base64 encoded -->
	</div>`

	log.Printf("Returning HTML length: %d (with encoded GPX length: %d)", len(html), len(gpxEncoded))

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, html)
	log.Printf("Successfully served activity %s", id)

	// // Build HTML partial for HTMX
	// html := `<div id="activity-details">
	//     <h2>Activity: ` + activity.Type + ` on ` + activity.Timestamp.Format("2006-01-02 15:04:05") + `</h2>
	//     <ul>`
	// for key, val := range stats {
	// 	html += fmt.Sprintf("<li><strong>%s:</strong> %v</li>", key, val)
	// }
	// html += `</ul>
	//     <div id="map" style="height: 400px;"></div> <!-- Map container -->
	//     <div id="gpx-data" style="display: none;">` + activity.GPXData + `</div> <!-- Hidden GPX for JS -->
	// </div>`

	// // Return the HTML partial
	// w.Header().Set("Content-Type", "text/html")
	// fmt.Fprint(w, html)
	// log.Printf("Successfully served activity %s", id)
}

// ActivitiesHandler returns an HTML partial (table rows) for HTMX
func ActivitiesHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("ActivitiesHandler called")
	// Access the DB from the db package (assumes db.DB is exported; adjust if needed, e.g., db.GetDB())
	rows, err := db.DB.Query("SELECT id, timestamp, type, stats_json FROM activities ORDER BY timestamp DESC")
	if err != nil {
		log.Printf("Error querying activities: %v", err)
		w.Header().Set("Content-Type", "text/html")
		http.Error(w, "<tr><td colspan='5'>Error loading activities</td></tr>", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var activities []models.Activity
	for rows.Next() {
		var id, timestampStr, activityType, statsJSON string
		if err := rows.Scan(&id, &timestampStr, &activityType, &statsJSON); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Parse timestamp string to time.Time (assumes RFC3339/ISO format like "2006-01-02T15:04:05Z")
		timestamp, err := time.Parse(time.RFC3339, timestampStr)
		if err != nil {
			log.Printf("Error parsing timestamp '%s' for ID %s: %v", timestampStr, id, err)
			continue // Skip rows with invalid timestamps
		}

		activities = append(activities, models.Activity{
			ID:        id,
			Timestamp: timestamp, // Now assigns the parsed time.Time
			Type:      activityType,
			StatsJSON: statsJSON, // Assign the raw string (as per your model)
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error after scanning rows: %v", err)
	}

	// Build HTML table rows
	var html string
	if len(activities) == 0 {
		html = "<tr><td colspan='5'>No activities yet</td></tr>"
	} else {
		for _, act := range activities {
			// Unmarshal StatsJSON on the fly for display
			var stats map[string]float64
			if err := json.Unmarshal([]byte(act.StatsJSON), &stats); err != nil {
				log.Printf("Error unmarshaling stats_json for ID %s: %v", act.ID, err)
				continue // Skip rows with invalid stats
			}

			distance := stats["distance"] // Default to 0 if missing
			elevation := stats["elevation"]

			// Format Timestamp for display (e.g., "2006-01-02T15:04:05Z")
			timestampFormatted := act.Timestamp.Format(time.RFC3339)

			html += fmt.Sprintf(
				`<tr>
                    <td>%s</td>
                    <td>%s</td>
                    <td>%.1f</td>
                    <td>%.0f</td>
                    <td><a href="/detail.html?id=%s">View</a></td>
                </tr>`,
				timestampFormatted, act.Type, distance, elevation, act.ID,
			)
		}
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, html)
}

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
