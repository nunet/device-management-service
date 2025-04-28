// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package geolocation

import (
	"math"
	"testing"

	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

func TestCoordinate(t *testing.T) {
	t.Parallel()

	t.Run("must be able to check if a coordinate is empty", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name       string
			coordinate Coordinate
			want       bool
		}{
			{
				name:       "empty coordinate",
				coordinate: Coordinate{},
				want:       true,
			},
			{
				name:       "non-empty coordinate",
				coordinate: Coordinate{lat: 1.0, long: 1.0},
				want:       false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				if got := tt.coordinate.Empty(); got != tt.want {
					t.Errorf("Coordinate.Empty() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("must be able to parse the coordinates", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name    string
			fields  []string
			want    Coordinate
			wantErr bool
		}{
			{
				name:    "valid coordinates",
				fields:  []string{"", "", "", "", "12.34", "56.78"},
				want:    Coordinate{lat: 12.34, long: 56.78},
				wantErr: false,
			},
			{
				name:    "invalid latitude",
				fields:  []string{"", "", "", "", "invalid", "56.78"},
				want:    Coordinate{},
				wantErr: true,
			},
			{
				name:    "invalid longitude",
				fields:  []string{"", "", "", "", "12.34", "invalid"},
				want:    Coordinate{},
				wantErr: true,
			},
			{
				name:    "missing fields",
				fields:  []string{"", "", "", "", "", "12.34"},
				want:    Coordinate{},
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				got, err := parseCoordinate(tt.fields)
				if (err != nil) != tt.wantErr {
					t.Errorf("parseCoordinate() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if err == nil && got != tt.want {
					t.Errorf("parseCoordinate() = %v, want %v", got, tt.want)
				}
			})
		}
	})

	t.Run("must be able to compute geodesic", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			name string
			p1   Coordinate
			p2   Coordinate
			want float64
		}{
			{
				name: "same coordinates",
				p1:   Coordinate{lat: 0, long: 0},
				p2:   Coordinate{lat: 0, long: 0},
				want: 0,
			},
			{
				name: "different coordinates",
				p1:   Coordinate{lat: 36.12, long: -86.67},
				p2:   Coordinate{lat: 33.94, long: -118.40},
				want: 2886.4444428379834, // Expected distance in km
			},
			{
				name: "empty coordinates",
				p1:   Coordinate{},
				p2:   Coordinate{lat: 33.94, long: -118.40},
				want: 0,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()

				got := ComputeGeodesic(tt.p1, tt.p2)
				if math.Abs(got-tt.want) > 0.01 {
					t.Errorf("ComputeGeodesic() = %v, want %v", got, tt.want)
				}
			})
		}
	})
}

func TestGeoLocator(t *testing.T) {
	t.Parallel()

	t.Run("must be able to create a new GeoLocator", func(t *testing.T) {
		t.Parallel()

		geo, err := NewGeoLocator()
		if err != nil {
			t.Fatalf("NewGeoLocator() error = %v", err)
		}

		if len(geo.coordinates) == 0 {
			t.Fatalf("NewGeoLocator() returned empty coordinates map")
		}
	})

	t.Run("must be able to get the coordinate of a location", func(t *testing.T) {
		t.Parallel()

		geo, err := NewGeoLocator()
		if err != nil {
			t.Fatalf("NewGeoLocator() error = %v", err)
		}

		loc := jtypes.Location{Country: "US", City: "Los Angeles"}
		coord, err := geo.Coordinate(loc)
		if err != nil {
			t.Fatalf("GeoLocator.Coordinate() error = %v", err)
		}
		if coord.Empty() {
			t.Fatalf("GeoLocator.Coordinate() returned empty coordinate")
		}
	})
}
