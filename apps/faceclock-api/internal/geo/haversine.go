package geo

import "math"

// EarthRadiusMeters is the mean radius of Earth in meters (WGS84 / IUGG recommendation).
const EarthRadiusMeters = 6371008.8

// Haversine calculates the great-circle distance between two points on Earth in meters.
// Coordinates are in decimal degrees. Longitude wrapping around the 180th meridian is handled.
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	// Zero distance optimization
	if lat1 == lat2 && lon1 == lon2 {
		return 0
	}

	phi1 := lat1 * math.Pi / 180.0
	phi2 := lat2 * math.Pi / 180.0
	deltaPhi := (lat2 - lat1) * math.Pi / 180.0

	// Handle longitude difference wrapping between -180 and 180 degrees
	deltaLambdaDeg := lon2 - lon1
	for deltaLambdaDeg > 180 {
		deltaLambdaDeg -= 360
	}
	for deltaLambdaDeg < -180 {
		deltaLambdaDeg += 360
	}
	deltaLambda := deltaLambdaDeg * math.Pi / 180.0

	a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*
			math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)

	// Clamp 'a' to [0, 1] to avoid math.NaN from floating-point rounding errors
	if a < 0 {
		a = 0
	} else if a > 1 {
		a = 1
	}

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return EarthRadiusMeters * c
}
