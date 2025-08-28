// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package e2e

// DeployWithVolume runs the tests that deploy with volume.
func DeployWithVolumeTest(suite *TestSuite) {
	suite.Run("dms creates a volume on storage node", func() {
		suite.T().Skip("glusterfs container is running in host mode")
		// glusterfs container is running in host mode
		// we can directly use the bootstrap nodes here
		// peers := strings.Join(suite.bootstrapPeers, ",")
		// envVars := []string{
		//	"DMS_PASSPHRASE=password3",
		//	"GOLOG_LOG_LEVEL=debug",
		//	"BOOTSTRAP_PEERS=" + peers,
		// }
		// err := runBinaryInContainer(glusterContainerName, "/home/dms", []string{"run", "--data-dir", "/home/data"}, envVars, "/home/output.log")
		// require.NoError(t, err)
		// firstNodeClient := suite.nodes[0]
		// suite.T().Logf("createVolume: glusterDMSDID: %s", suite.glusterDMSDID)
		// time.Sleep(20 * time.Second)
		// _, err = firstNodeClient.client.createVolume(t, firstNodeClient.userContext, firstNodeClient.password, suite.glusterDMSDID)
		// require.NoError(t, err)
	})
}
