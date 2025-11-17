package models

import "time"

type Activity struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	StatsJSON string    `json:"stats_json"` // e.g., '{"distance": 10.5, "elevation": 200, ...}'
	GPXData   string    `json:"gpx_data"`   // GPX XML string
}
