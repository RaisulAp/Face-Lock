package geo

import "github.com/google/uuid"

// OfficeLocation represents an office location used in geofence calculations.
type OfficeLocation struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	RadiusMeter int       `json:"radius_meter"`
	IsActive    bool      `json:"is_active"`
	Address     string    `json:"address,omitempty"`
}

// GeofenceResult holds the result of evaluating employee coordinates against office locations.
type GeofenceResult struct {
	Inside        bool       `json:"inside"`
	DistanceMeter float64    `json:"distance_meter"`
	OfficeID      *uuid.UUID `json:"office_id"`
	OfficeName    string     `json:"office_name"`
}

// Evaluate evaluates employee coordinates against all active office locations.
// If the employee is within the radius of one or more locations, Inside=true and
// the covering location with the smallest distance is selected.
// If outside all locations, Inside=false and the closest location overall is selected.
// If locations is empty or has no active locations, Inside=false, DistanceMeter=0, OfficeID=nil.
func Evaluate(lat, lon float64, locations []OfficeLocation) GeofenceResult {
	var (
		coveringFound   bool
		minCoveringDist float64
		bestCovering    OfficeLocation

		closestDist float64
		bestClosest OfficeLocation
		first       = true
	)

	for _, loc := range locations {
		if !loc.IsActive {
			continue
		}

		d := Haversine(lat, lon, loc.Latitude, loc.Longitude)

		if first || d < closestDist {
			closestDist = d
			bestClosest = loc
			first = false
		}

		if d <= float64(loc.RadiusMeter) {
			if !coveringFound || d < minCoveringDist {
				coveringFound = true
				minCoveringDist = d
				bestCovering = loc
			}
		}
	}

	if first {
		// No active office locations
		return GeofenceResult{
			Inside:        false,
			DistanceMeter: 0,
			OfficeID:      nil,
			OfficeName:    "",
		}
	}

	if coveringFound {
		return GeofenceResult{
			Inside:        true,
			DistanceMeter: minCoveringDist,
			OfficeID:      &bestCovering.ID,
			OfficeName:    bestCovering.Name,
		}
	}

	return GeofenceResult{
		Inside:        false,
		DistanceMeter: closestDist,
		OfficeID:      &bestClosest.ID,
		OfficeName:    bestClosest.Name,
	}
}
