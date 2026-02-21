package main

import "math"

const (
	earthRadiusMeters = 6371000 // Earth's radius in meters
	metersPerMile     = 1609.34
)

// Haversine calculates the distance between two points on Earth in meters
func Haversine(lat1, lng1, lat2, lng2 float64) float64 {
	lat1Rad := lat1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	deltaLat := (lat2 - lat1) * math.Pi / 180
	deltaLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLng/2)*math.Sin(deltaLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

// IsWithinRadius checks if two points are within the specified radius (in miles)
func IsWithinRadius(lat1, lng1, lat2, lng2, radiusMiles float64) bool {
	distance := Haversine(lat1, lng1, lat2, lng2)
	radiusMeters := radiusMiles * metersPerMile
	return distance <= radiusMeters
}

// DistanceInMiles returns the distance between two points in miles
func DistanceInMiles(lat1, lng1, lat2, lng2 float64) float64 {
	meters := Haversine(lat1, lng1, lat2, lng2)
	return meters / metersPerMile
}
