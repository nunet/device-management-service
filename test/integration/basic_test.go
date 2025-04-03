package itest

func BasicTests(suite *TestSuite) {
	// every node in the network must be able to broadcast
	for _, node := range suite.nodes {
		result := node.client.broadcast(suite.T(), node.userContext, node.password)
		suite.Equal(suite.numNodes, countDIDOccurrences(result))
	}
}
