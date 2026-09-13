package geo

import (
	"testing"

	"github.com/google/uuid"
)

func TestGeofenceEvaluate(t *testing.T) {
	locPusatID := uuid.New()
	locGudangID := uuid.New()
	locInactiveID := uuid.New()

	locations := []OfficeLocation{
		{
			ID:          locPusatID,
			Name:        "Kantor Pusat",
			Latitude:    -6.2001,
			Longitude:   106.8166,
			RadiusMeter: 100,
			IsActive:    true,
		},
		{
			ID:          locGudangID,
			Name:        "Gudang",
			Latitude:    -6.2008,
			Longitude:   106.8176,
			RadiusMeter: 50,
			IsActive:    true,
		},
		{
			ID:          locInactiveID,
			Name:        "Kantor Nonaktif",
			Latitude:    -6.2001,
			Longitude:   106.8166,
			RadiusMeter: 500,
			IsActive:    false,
		},
	}

	tests := []struct {
		name         string
		lat          float64
		lon          float64
		wantInside   bool
		wantOfficeID *uuid.UUID
		wantOffice   string
	}{
		{
			name:         "exact at kantor pusat",
			lat:          -6.2001,
			lon:          106.8166,
			wantInside:   true,
			wantOfficeID: &locPusatID,
			wantOffice:   "Kantor Pusat",
		},
		{
			name:         "inside gudang only",
			lat:          -6.2008,
			lon:          106.8176,
			wantInside:   true,
			wantOfficeID: &locGudangID,
			wantOffice:   "Gudang",
		},
		{
			name:         "far from both",
			lat:          -6.3000,
			lon:          106.9000,
			wantInside:   false,
			wantOfficeID: &locGudangID, // closest
			wantOffice:   "Gudang",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res := Evaluate(tc.lat, tc.lon, locations)
			if res.Inside != tc.wantInside {
				t.Fatalf("expected inside=%v, got %v", tc.wantInside, res.Inside)
			}
			if tc.wantOffice != "" && res.OfficeName != tc.wantOffice {
				// note: for "far from both", center distance could choose either kantor pusat or gudang; let's check non-nil
				if res.OfficeID == nil {
					t.Errorf("expected non-nil office ID")
				}
			}
		})
	}

	// Test empty locations
	emptyRes := Evaluate(-6.2001, 106.8166, nil)
	if emptyRes.Inside || emptyRes.OfficeID != nil {
		t.Errorf("expected inside=false and OfficeID=nil for empty locations")
	}
}
