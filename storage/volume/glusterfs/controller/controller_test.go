// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const pemData = `-----BEGIN CERTIFICATE-----
MIIDBzCCAe+gAwIBAgIUUv6ChJPjUGuZgha3FVuBAwVsJ2swDQYJKoZIhvcNAQEL
BQAwEzERMA8GA1UEAwwIY2xpZW50MDEwHhcNMjUwMzIxMTE0NzAzWhcNMjUwNDIw
MTE0NzAzWjATMREwDwYDVQQDDAhjbGllbnQwMTCCASIwDQYJKoZIhvcNAQEBBQAD
ggEPADCCAQoCggEBAMrKH6rORCn+keE3xWRigo5emNR3dgy4sAUppagVRAeAlr24
GHzoE88/CThmAB/+jo9BTa5q9KYFB7XzjIFmgWKbCRK2nYYlIHBq0G1CPMVuMb3S
19TMTgksCfXlBS2SNhBTOJMbBpQPmOQO0zFWqI1G5Fsnlnmz09nN01Y6JEW86OMt
vUkjsbLrkFY9fZFmOSsGmG71UH+oFz4I0axy26uToG3ofQTHdmyK9NGnWHo9Gevk
FKCLO39IK+XHB57Q9eHgH4rqtCHdwUkwNb1if6sYl8Zpq2e8WfPmfsYvJ/QJTj3e
posDMCOh5nYsaM+a6+Wm1CyYz6osWt/NsGAPduUCAwEAAaNTMFEwHQYDVR0OBBYE
FC4Ec9NLWbt7UMvRZqxQE+jNaPVrMB8GA1UdIwQYMBaAFC4Ec9NLWbt7UMvRZqxQ
E+jNaPVrMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBACsUhg8e
nLlg6VjsPusiSFQVkvfgkaOFnHDlXy1srNcLkjAgDCfWN/UWUC16Gajo6R/86nKq
UlVkYOvjWCnbXTljTeCK/S9UJp/HjzyqyQa6RB8g5mh9BKVNUkiuqABS8X9UuxVP
Fjsc8HDShZOG9e4V12T2R8lAFZkVKt0IAye2D1wY/Zu5iCvIwjeGPstkX5b6Sshg
jhXHPS0IVfFENiF5P3HUzUa+lj5ekINjp18EjCFuG9JDuue97DgK9ibvaokbMsY9
dGRATsA89JBR8SKXX4iW3XX+UpV1TpPZQBpdU2sBV6+SWGP1VBF4DgWhq/IinpN6
1l2b8kBso2JR/Jg=
-----END CERTIFICATE-----`

func TestValidatePEM(t *testing.T) {
	err := validatePEM([]byte(pemData))
	require.NoError(t, err)
}
