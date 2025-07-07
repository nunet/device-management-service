// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package main

import (
	"gitlab.com/nunet/device-management-service/cmd"
)

//	@title			Device Management Service
//	@version		0.4.185
//	@description	A dashboard application for computing providers.
//	@termsOfService	https://nunet.io/tos

//	@contact.name	Support
//	@contact.url	https://devexchange.nunet.io/
//	@contact.email	support@nunet.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @host		localhost:9999
//
// @Schemes	http
//
// @BasePath	/api/v1
func main() {
	cmd.Execute()
}
