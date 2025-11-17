package utils

import (
	"fmt"
	"strings"
)

// generateGPXFromFIT generates a GPX XML string from parsed FIT data.
// Assumes parsedData has a Points slice as shown above.
func GenerateGPXFromFIT(parsedData struct {
	ID          string
	Type        string
	Timestamp   int64
	Distance    float64
	Elevation   float64
	RecordCount int
	Points      []struct {
		Lat  float64
		Long float64
		Ele  float64
		Time int64
	}
}) string {
	if len(parsedData.Points) == 0 {
		return "" // Or handle as empty GPX
	}

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString(`<gpx version="1.1" creator="OwnPath">`)
	sb.WriteString(`<trk><trkseg>`)

	for _, pt := range parsedData.Points {
		sb.WriteString(fmt.Sprintf(
			`<trkpt lat="%f" lon="%f"><ele>%f</ele></trkpt>`,
			pt.Lat, pt.Long, pt.Ele,
		))
	}

	sb.WriteString(`</trkseg></trk></gpx>`)
	return sb.String()
}
