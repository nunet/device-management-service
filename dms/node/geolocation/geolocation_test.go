// ss Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package geolocation

import (
	"net"
	"testing"

	"github.com/oschwald/geoip2-golang"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

func TestGeolocate(t *testing.T) {
	tests := []struct {
		name       string
		ip         net.IP
		country    string
		continent  string
		networkErr error
		geoIPErr   error
		expectErr  bool
		expectLoc  jobtypes.Location
	}{
		{
			name:      "successful geolocation",
			ip:        net.ParseIP("1.2.3.4"),
			country:   "US",
			continent: "NA",
			expectLoc: jobtypes.Location{
				Country:   "US",
				Continent: "NA",
			},
		},
		{
			name:       "network error",
			networkErr: assert.AnError,
			expectErr:  true,
		},
		{
			name:      "geoIP error",
			ip:        net.ParseIP("1.2.3.4"),
			geoIPErr:  assert.AnError,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockNetwork := NewMockNetwork(ctrl)
			mockGeoIP := NewMockGeoIPLocator(ctrl)

			mockNetwork.EXPECT().
				HostPublicIP().
				Return(tt.ip, tt.networkErr)

			if tt.networkErr == nil {
				mockGeoIP.EXPECT().
					Country(tt.ip).
					Return(&geoip2.Country{
						Country: struct {
							Names             map[string]string `maxminddb:"names"`
							IsoCode           string            `maxminddb:"iso_code"`
							GeoNameID         uint              `maxminddb:"geoname_id"`
							IsInEuropeanUnion bool              `maxminddb:"is_in_european_union"`
						}{
							IsoCode: tt.country,
						},
						Continent: struct {
							Names     map[string]string `maxminddb:"names"`
							Code      string            `maxminddb:"code"`
							GeoNameID uint              `maxminddb:"geoname_id"`
						}{
							Code: tt.continent,
						},
					}, tt.geoIPErr)
			}

			ip, err := mockNetwork.HostPublicIP()
			if err != nil && tt.networkErr != nil {
				return
			}

			loc, err := Geolocate(ip, mockGeoIP)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectLoc, loc)
			}
		})
	}
}
