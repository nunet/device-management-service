package jobs

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// currently only using a GeoNames file for cities with population > 5000
// download it here: https://download.geonames.org/export/dump/cities5000.zip
//
//go:embed cities5000.txt
var cities5000 string

const lightSpeed = 299792.458 // in km/s

type Coordinate struct {
	lat  float64
	long float64
}

func (c *Coordinate) Empty() bool {
	return c.lat == 0 && c.long == 0
}

type GeoLocator struct {
	coord map[string]map[string]Coordinate // country -> city -> coordinate
}

func NewGeoLocator() (*GeoLocator, error) {
	buf := bytes.NewBufferString(cities5000)
	geo := &GeoLocator{
		coord: make(map[string]map[string]Coordinate),
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
		coord, err := parseCoordinate(fields)
		if err != nil {
			log.Warnf("error parsing coordiates for %s in %s: %s", cityName, countryCode, err)
			continue
		}

		countryMap, ok := geo.coord[countryCode]
		if !ok {
			countryMap = make(map[string]Coordinate)
			geo.coord[countryCode] = countryMap
		}
		countryMap[cityName] = coord
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading cities file: %w", err)
	}

	return geo, nil
}

func parseCoordinate(fields []string) (Coordinate, error) {
	lat, err := strconv.ParseFloat(fields[4], 64)
	if err != nil {
		return Coordinate{}, fmt.Errorf("failed to parse latitude: %w", err)
	}
	long, err := strconv.ParseFloat(fields[5], 64)
	if err != nil {
		return Coordinate{}, fmt.Errorf("failed to parse longitude: %w", err)
	}
	return Coordinate{lat: lat, long: long}, nil
}

func (geo *GeoLocator) Coordinate(loc Location) (Coordinate, error) {
	if loc.Country == "" || loc.City == "" {
		return Coordinate{}, fmt.Errorf("no city in location")
	}

	coord, ok := geo.coord[loc.Country][loc.City]
	if !ok {
		return Coordinate{}, fmt.Errorf("unknown city")
	}

	return coord, nil
}

// using a haversine formula to calculate the shortest path
func computeGeodesic(p1, p2 Coordinate) float64 {
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
