package db

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/gratten/ownpath/internal/models" // Adjust import path
	_ "github.com/mattn/go-sqlite3"              // SQLite driver
)

// DB is a global handle for the database connection.
var DB *sql.DB

// InitDB initializes the SQLite database.
// It creates the file if it doesn't exist and sets up the schema.
func InitDB(dbPath string) error {
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Create the activities table if it doesn't exist.
	schema := `
    CREATE TABLE IF NOT EXISTS activities (
        id TEXT PRIMARY KEY,          -- Unique ID (e.g., UUID or hash from FIT file)
        timestamp DATETIME NOT NULL,  -- Activity start time
        type TEXT NOT NULL,           -- e.g., 'run', 'hike', 'bike'
        stats_json TEXT NOT NULL,     -- Serialized JSON of stats (distance, elevation, etc.)
        gpx_data TEXT                 -- GPX XML string for map rendering (optional for MVP)
    );`
	_, err = DB.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	log.Println("Database initialized successfully")
	return nil
}

// CloseDB closes the database connection.
func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

// InsertActivity inserts a new activity into the database.
func InsertActivity(act models.Activity) error {
	stmt := `INSERT INTO activities (id, timestamp, type, stats_json, gpx_data) VALUES (?, ?, ?, ?, ?)`
	_, err := DB.Exec(stmt, act.ID, act.Timestamp, act.Type, act.StatsJSON, act.GPXData)
	if err != nil {
		return fmt.Errorf("failed to insert activity: %w", err)
	}
	return nil
}

// GetActivities returns a list of all activities (for dashboard).
func GetActivities() ([]models.Activity, error) {
	rows, err := DB.Query(`SELECT id, timestamp, type, stats_json, gpx_data FROM activities ORDER BY timestamp DESC`)
	if err != nil {
		return nil, fmt.Errorf("failed to query activities: %w", err)
	}
	defer rows.Close()

	var activities []models.Activity
	for rows.Next() {
		var act models.Activity
		var ts string // Temporary for timestamp
		if err := rows.Scan(&act.ID, &ts, &act.Type, &act.StatsJSON, &act.GPXData); err != nil {
			return nil, fmt.Errorf("failed to scan activity: %w", err)
		}
		act.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts) // Adjust format if needed
		activities = append(activities, act)
	}
	return activities, nil
}

// GetActivityByID returns a single activity by ID (for detail view).
func GetActivityByID(id string) (*models.Activity, error) {
	row := DB.QueryRow(`SELECT id, timestamp, type, stats_json, gpx_data FROM activities WHERE id = ?`, id)

	var act models.Activity
	var ts string
	err := row.Scan(&act.ID, &ts, &act.Type, &act.StatsJSON, &act.GPXData)
	if err == sql.ErrNoRows {
		return nil, nil // Not found
	} else if err != nil {
		return nil, fmt.Errorf("failed to get activity: %w", err)
	}
	act.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
	return &act, nil
}
