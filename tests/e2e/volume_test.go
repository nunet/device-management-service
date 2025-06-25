package itest

// DeployWithVolume runs the tests that deploy with volume.
func DeployWithVolumeTest(suite *TestSuite) {
	suite.Run("dms creates a volume on storage node", func() {
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
