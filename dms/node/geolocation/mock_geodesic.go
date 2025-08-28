// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
