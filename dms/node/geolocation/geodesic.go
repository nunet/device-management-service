// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package geolocation

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"

	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// currently only using a GeoNames file for cities with population > 5000
// download it here: https://download.geonames.org/export/dump/cities5000.zip
//
//go:embed cities5000.txt
var cities5000 string

const LightSpeed = 299792.458 // in km/s

type Coordinate struct {
	lat  float64
	long float64
}

func (c *Coordinate) Empty() bool {
	return c.lat == 0 && c.long == 0
}

type GeoLocator struct {
	coordinates map[string]map[string]Coordinate // country -> city -> coordinate
}

func NewGeoLocator() (*GeoLocator, error) {
	buf := bytes.NewBufferString(cities5000)
	geo := &GeoLocator{
		coordinates: make(map[string]map[string]Coordinate),
	}

	scanner := bufio.NewScanner(buf)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // increase buffer size for large lines

	for scanner.Scan() {
		fields := strings.SplitN(scanner.Text(), "\t", 20) // limit to 20 fields in each entry
		if len(fields) < 19 {
			continue
		}

		cityName := fields[1]
		countryCode := fields[8]
		coordinate, err := parseCoordinate(fields)
		if err != nil {
			log.Warnf("failure parsing coordiates for %s in %s: %s", cityName, countryCode, err)
			continue
		}

		countryMap, ok := geo.coordinates[countryCode]
		if !ok {
			countryMap = make(map[string]Coordinate)
			geo.coordinates[countryCode] = countryMap
		}
		countryMap[cityName] = coordinate
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading cities file: %w", err)
	}

	return geo, nil
}

func (geo *GeoLocator) Coordinate(loc jtypes.Location) (Coordinate, error) {
	if loc.Country == "" || loc.City == "" {
		return Coordinate{}, fmt.Errorf("no city in location")
	}

	coord, ok := geo.coordinates[loc.Country][loc.City]
	if !ok {
		return Coordinate{}, fmt.Errorf("unknown city")
	}

	return coord, nil
}

func parseCoordinate(fields []string) (Coordinate, error) {
	lat, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		return Coordinate{}, fmt.Errorf("parse latitude: %w", err)
	}
	long, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		return Coordinate{}, fmt.Errorf("parse longitude: %w", err)
	}
	return Coordinate{lat: lat, long: long}, nil
}

// using a haversine formula to calculate the shortest path
func ComputeGeodesic(p1, p2 Coordinate) float64 {
	const earthRadius = 6371 // km

	if p1.Empty() || p2.Empty() {
		return 0
	}

	lat1 := p1.lat * math.Pi / 180
	lat2 := p2.lat * math.Pi / 180
	dLat := (p2.lat - p1.lat) * math.Pi / 180
	dLon := (p2.long - p1.long) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
