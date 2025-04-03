// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package geolocation

import (
	"net"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	types "gitlab.com/nunet/device-management-service/types"
)

type Geolocation struct {
	Continent string
	Country   string
	City      string
}

// TODO: Add city check when we have GeoIP database with cities
func (g *Geolocation) Empty() bool {
	return g.Continent == "" && g.Country == ""
}

func Geolocate(ip net.IP, geoIP types.GeoIPLocator) (jobtypes.Location, error) {
	rec, err := geoIP.Country(ip)
	if err != nil {
		return jobtypes.Location{}, err
	}
	location := jobtypes.Location{
		Country:   rec.Country.IsoCode,
		Continent: rec.Continent.Code,
	}
	return location, nil
}
