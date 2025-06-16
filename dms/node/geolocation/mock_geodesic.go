package geolocation

import (
	"fmt"

	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// MockGeoLocator is a mock implementation of LocationProvider for testing
type MockGeoLocator struct {
	coordinates map[string]map[string]Coordinate
}

// NewMockGeoLocator creates a new mock GeoLocator
func NewMockGeoLocator() *MockGeoLocator {
	return &MockGeoLocator{
		coordinates: make(map[string]map[string]Coordinate),
	}
}

// AddLocation adds a location to the mock GeoLocator
func (m *MockGeoLocator) AddLocation(country, city string, lat, long float64) {
	if _, ok := m.coordinates[country]; !ok {
		m.coordinates[country] = make(map[string]Coordinate)
	}
	m.coordinates[country][city] = Coordinate{lat: lat, long: long}
}

// Coordinate returns the coordinate for a location
func (m *MockGeoLocator) Coordinate(loc jtypes.Location) (Coordinate, error) {
	if loc.Country == "" || loc.City == "" {
		return Coordinate{}, fmt.Errorf("no city in location")
	}

	coord, ok := m.coordinates[loc.Country][loc.City]
	if !ok {
		return Coordinate{}, fmt.Errorf("unknown city")
	}

	return coord, nil
}

func (m *MockGeoLocator) Empty() bool {
	return len(m.coordinates) == 0
}
