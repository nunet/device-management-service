package main

import "gitlab.com/nunet/device-management-service/cmd"

//	@title			Device Management Service
//	@version		0.4.171
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
