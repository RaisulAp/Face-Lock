package geo

import (
	"math"
	"testing"
)

func TestHaversineZeroDistance(t *testing.T) {
	d := Haversine(-6.2088, 106.8456, -6.2088, 106.8456)
	if d != 0 {
		t.Errorf("expected 0, got %f", d)
	}
}

func TestHaversineKnownDistances(t *testing.T) {
	// Jakarta (-6.2088, 106.8456) to Bandung (-6.9175, 107.6191) ≈ 118-120 km
	d := Haversine(-6.2088, 106.8456, -6.9175, 107.6191)
	expectedKm := 119.0
	actualKm := d / 1000.0
	diff := math.Abs(actualKm - expectedKm)
	if diff > 5.0 {
		t.Errorf("Jakarta to Bandung distance expected ~119 km, got %f km", actualKm)
	}

	// 180th meridian crossing: (0, 179.999) to (0, -179.999) should be ~222.4 meters, NOT half the globe
	meridianDist := Haversine(0.0, 179.999, 0.0, -179.999)
	if meridianDist > 300 || meridianDist < 200 {
		t.Errorf("180th meridian crossing expected ~222.4m, got %f meters", meridianDist)
	}

	// Equator: 1 degree longitude at equator (0, 0) to (0, 1)
	equatorDegreeDist := Haversine(0.0, 0.0, 0.0, 1.0)
	expectedDegree := 111195.0 // ~111.195 km
	if math.Abs(equatorDegreeDist-expectedDegree) > 500 {
		t.Errorf("1 degree at equator expected ~111.195km, got %f meters", equatorDegreeDist)
	}

	// Antipodal points: North Pole to South Pole
	polesDist := Haversine(90.0, 0.0, -90.0, 0.0)
	expectedPoles := math.Pi * EarthRadiusMeters
	if math.Abs(polesDist-expectedPoles) > 100 {
		t.Errorf("Poles distance expected %f, got %f", expectedPoles, polesDist)
	}
}
