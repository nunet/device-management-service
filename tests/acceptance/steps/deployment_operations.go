package steps

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/tests/acceptance/hooks"
	"gitlab.com/nunet/device-management-service/tests/acceptance/utils"
)

// DeploymentOperations registers all step definitions for deployment operations feature
func DeploymentOperations(ctx *godog.ScenarioContext) {
	ctx.Before(func(ctx context.Context, _ *godog.Scenario) (context.Context, error) {
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})
	ctx.After(func(ctx context.Context, scenario *godog.Scenario, _ error) (context.Context, error) {
		scenarioName := strings.ReplaceAll(scenario.Name, " ", "_")
		if err := hooks.SaveLogs(ctx, scenarioName); err != nil {
			return ctx, err
		}
		if err := hooks.CleanupNodes(); err != nil {
			return ctx, err
		}
		return ctx, nil
	})

	ctx.Step(`^the following nodes$`, theFollowingNodes)
	ctx.Step(`^"([^"]*)" has (\d+) (\w+)? "([^"]*)" on "([^"]*)"$`, hasDeployments)
	ctx.Step(`^"([^"]*)" has (\d+) (\w+)? (\w+) on "([^"]*)"$`, hasDeployments)
	ctx.Step(`^"([^"]*)" restarts DMS$`, restartsDMS)
	ctx.Step(`^"([^"]*)" list deployments$`, listDeployments)
	ctx.Step(`^"([^"]*)" should see the (\w+)? restored$`, shouldSeeDeploymentRestored)
	ctx.Step(`^"([^"]*)" prunes the deployment$`, prunesTheDeployment)
	ctx.Step(`^"([^"]*)" should see deployment list empty$`, shouldSeeDeploymentListEmpty)
	ctx.Step(`^"([^"]*)" has (\d+) tasks with status "([^"]*)" on "([^"]*)"$`, hasMultipleTasksWithStatus)
	ctx.Step(`^"([^"]*)" lists deployments filtered by status "([^"]*)" with limit (\d+) and offset (\d+)$`, listDeploymentsWithFiltersAndPagination)
	ctx.Step(`^"([^"]*)" should see (\d+) deployment(s)?$`, shouldSeeDeploymentCount)
	ctx.Step(`^all deployments should have status "([^"]*)"$`, allDeploymentsShouldHaveStatus)
	ctx.Step(`^the response should indicate (true|false) more results available$`, shouldHaveMoreResults)
	ctx.Step(`^the response should have total (\d+)$`, shouldHaveTotalCount)
}

func hasDeployments(ctx context.Context, spName string, count int, ensemble, status, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	switch ensemble {
	case "task":
		ensemble = dockerHelloYAML
	case "service":
		ensemble = "nginx.yaml"
	default:
		return ctx, fmt.Errorf("invalid type of ensemble: %s", ensemble)
	}

	// TODO: check if we have enough CP nodes
	switch count {
	case 1:
		deployCtx, err := hasDeployedOn(tc.Unwrap(), spName, ensemble, cpName)
		assert.NoError(t, err)
		tc = utils.NewTestCtx(deployCtx)
	default:
		return ctx, fmt.Errorf("currently multiple deployment not supported")
	}

	err := deploymentIs(tc.Unwrap(), spName, status)
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func restartsDMS(ctx context.Context, spName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	assert.NoError(t, spDmsCtx.Stop())

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.False(c, sp.IsDMSRunning(9999))
	}, 60*time.Second, 500*time.Millisecond, "failed to stop dms")

	slog.Info("dms stopped")

	time.Sleep(2 * time.Second)

	start := time.Now()
	assert.NoError(t, spDmsCtx.Run(t))

	require.EventuallyWithT(t, func(c *assert.CollectT) {
		assert.True(c, sp.IsDMSRunning(9999))
	}, 30*time.Second, 1*time.Second, "failed to start dms")

	duration := time.Since(start)
	if duration > time.Minute {
		slog.Warn("dms restart took too long", "duration", duration)
	} else {
		slog.Info("dms restarted", "duration", duration)
	}

	// test command to see if dms restarted
	//	require.Eventually(t, func() bool {
	//		_, err := spDmsCtx.PeerAddr()
	//		return assert.NoError(t, err)
	//	}, 30*time.Second, 500*time.Millisecond, "failed to test restarted dms")

	return tc.Unwrap(), nil
}

func listDeployments(ctx context.Context, spName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	deployments, err := spDmsCtx.DeploymentList()
	assert.NoError(t, err)

	tc = tc.WithDeployments(deployments)

	return tc.Unwrap(), nil
}

func shouldSeeDeploymentRestored(ctx context.Context, spName string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	ensembleID, err := tc.EnsembleID()
	assert.NoError(t, err)
	assert.NotEmpty(t, ensembleID)

	deployments, err := tc.Deployments()
	assert.NoError(t, err)
	assert.NotEmpty(t, deployments)

	rawGotStatus, ok := deployments[ensembleID]
	assert.True(t, ok)

	var gotStatus jobtypes.DeploymentStatus
	var statusFound bool
	for i := jobtypes.DeploymentStatusPreparing; i <= jobtypes.DeploymentStatusCompleted; i++ {
		if strings.EqualFold(i.String(), rawGotStatus) {
			gotStatus = i
			statusFound = true
			break
		}
	}
	assert.True(t, statusFound, "unknown status: %s", rawGotStatus)

	assert.GreaterOrEqual(t, gotStatus, jobtypes.DeploymentStatusPreparing)

	// switch jobtypes.AllocationType(allocType) {
	// case jobtypes.AllocationTypeService:
	// 	wantStatus := jobtypes.DeploymentStatusRunning
	// 	assert.Equal(t, wantStatus, gotStatus, "wanted status %s, got %s", wantStatus, gotStatus)
	// case jobtypes.AllocationTypeTask:
	// 	minStatus := jobtypes.DeploymentStatusRunning
	// 	maxStatus := jobtypes.DeploymentStatusCompleted
	// 	statusOK := gotStatus >= minStatus && gotStatus <= maxStatus
	// 	assert.True(t, statusOK, "wanted status in range [%s, %s], but got %s", minStatus, maxStatus, gotStatus)
	// }

	return nil
}

func prunesTheDeployment(ctx context.Context, spName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	err = spDmsCtx.PruneDeployments()
	assert.NoError(t, err)

	return tc.Unwrap(), nil
}

func shouldSeeDeploymentListEmpty(ctx context.Context, spName string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	nodes, err := tc.Nodes()
	assert.NoError(t, err)
	assert.NotEmpty(t, nodes)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	deployments, err := spDmsCtx.DeploymentList()
	assert.NoError(t, err)
	assert.Empty(t, deployments)

	return nil
}

func hasMultipleTasksWithStatus(ctx context.Context, spName string, count int, status, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	// Choose the appropriate YAML file based on expected status
	// - For "Running": use nginx.yaml (long-running service)
	// - For "Completed": use docker_hello.yaml (task that completes quickly)
	var ensembleFile string
	var needsPortModification bool
	switch strings.ToLower(status) {
	case "running":
		ensembleFile = "nginx.yaml"
		needsPortModification = true // nginx.yaml uses static port 50001
	case "completed":
		ensembleFile = dockerHelloYAML
		needsPortModification = false // docker_hello.yaml doesn't use ports
	default:
		// Default to docker_hello.yaml for other statuses
		ensembleFile = dockerHelloYAML
		needsPortModification = false
	}

	// Get nodes and DMS context
	nodes, err := tc.Nodes()
	require.NoError(t, err)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	cp, cpDmsCtx := utils.NodeWithDMS(nodes, cpName)
	assert.NotNil(t, cp)
	assert.NotNil(t, cpDmsCtx)

	// Get CP peer info once
	cpInfo, err := cpDmsCtx.PeerAddr()
	require.NoError(t, err)
	assert.NotNil(t, cpInfo)

	cpAddr, err := utils.MultiaddrFromCLI(cpInfo)
	require.NoError(t, err)
	assert.NotEmpty(t, cpAddr)

	err = spDmsCtx.Connect(cpAddr)
	require.NoError(t, err)

	// Find the ensemble file
	ensemblePath := fmt.Sprintf("ensembles/%s", ensembleFile)
	file := utils.FindTestdata(ensemblePath)

	ensembleIDs := make([]string, 0, count)
	basePort := 50001 // Starting port for nginx deployments

	for i := 0; i < count; i++ {
		// Upload ensemble file (creates a new copy each time)
		ensemble, err := utils.UploadFile(sp, file)
		require.NoError(t, err)
		assert.NotEmpty(t, ensemble)

		// Upload scripts if needed
		err = utils.UploadScripts(sp, ensemble)
		require.NoError(t, err)

		// Set peer
		_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node1.peer = \"%s\"", cpInfo.ID), ensemble})
		require.NoError(t, err)

		// Modify port if needed (for nginx.yaml)
		if needsPortModification {
			port := basePort + i
			_, err = sp.RunCMD([]string{"yq", "-i", fmt.Sprintf(".nodes.node1.ports[0].public = %d", port), ensemble})
			require.NoError(t, err)
		}

		// Deploy
		ensembleID, err := spDmsCtx.Deploy(ensemble)
		require.NoError(t, err)
		assert.NotEmpty(t, ensembleID)
		ensembleIDs = append(ensembleIDs, ensembleID)
	}

	// Now wait for deployments to reach expected status in parallel
	// (nodes and spDmsCtx are already set above)

	wg := sync.WaitGroup{}
	// Launch goroutines to wait for each deployment in parallel
	for _, id := range ensembleIDs {
		wg.Add(1)
		go func(deploymentID string) {
			defer wg.Done()
			require.Eventually(t, func() bool {
				ensembleStatus, err := spDmsCtx.EnsembleStatus(deploymentID)
				// Don't use require.NoError here - just return false on error
				// require.Eventually will keep retrying until timeout
				if err != nil {
					return false
				}
				return strings.EqualFold(ensembleStatus, status)
			}, 3*60*time.Second, 1*time.Second,
				"deployment %s did not reach %s status", deploymentID, status)
		}(id)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return tc.Unwrap(), nil
}

func listDeploymentsWithFiltersAndPagination(ctx context.Context, spName, statusStr string, limit, offset int) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	// Parse status string to DeploymentStatus
	var status jobtypes.DeploymentStatus
	statusFound := false
	statusStr = strings.TrimSpace(statusStr)
	for i := jobtypes.DeploymentStatusPreparing; i <= jobtypes.DeploymentStatusCompleted; i++ {
		if strings.EqualFold(i.String(), statusStr) {
			status = i
			statusFound = true
			break
		}
	}
	require.True(t, statusFound, "unknown status: %s", statusStr)

	nodes, err := tc.Nodes()
	require.NoError(t, err)

	sp, spDmsCtx := utils.NodeWithDMS(nodes, spName)
	assert.NotNil(t, sp)
	assert.NotNil(t, spDmsCtx)

	// Use new DeploymentListWithQuery method
	response, err := spDmsCtx.DeploymentListWithQuery(utils.DeploymentListQuery{
		Limit:  limit,
		Offset: offset,
		Status: []jobtypes.DeploymentStatus{status},
	})
	assert.NoError(t, err)

	// Store response in context for later assertions
	tc = tc.WithDeploymentListResponse(response)

	return tc.Unwrap(), nil
}

func shouldSeeDeploymentCount(ctx context.Context, _ string, expectedCount int) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	response, err := tc.DeploymentListResponse()
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Equal(t, expectedCount, len(response.Deployments),
		"expected %d deployments, got %d", expectedCount, len(response.Deployments))

	return nil
}

func allDeploymentsShouldHaveStatus(ctx context.Context, expectedStatusStr string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	response, err := tc.DeploymentListResponse()
	require.NoError(t, err)
	assert.NotNil(t, response)

	expectedStatus, err := parseStatus(expectedStatusStr)
	require.NoError(t, err)

	for _, deployment := range response.Deployments {
		actualStatus, err := parseStatus(deployment.Status)
		require.NoError(t, err)
		assert.Equal(t, expectedStatus, actualStatus,
			"deployment %s has status %s, expected %s",
			deployment.OrchestratorID, deployment.Status, expectedStatusStr)
	}

	return nil
}

func shouldHaveMoreResults(ctx context.Context, hasMoreStr string) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	response, err := tc.DeploymentListResponse()
	require.NoError(t, err)
	assert.NotNil(t, response)

	expectedHasMore := hasMoreStr == "true"
	assert.Equal(t, expectedHasMore, response.HasMore,
		"expected has_more=%v, got %v", expectedHasMore, response.HasMore)

	return nil
}

func shouldHaveTotalCount(ctx context.Context, expectedTotal int) error {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	response, err := tc.DeploymentListResponse()
	require.NoError(t, err)
	assert.NotNil(t, response)

	assert.Equal(t, expectedTotal, response.Total,
		"expected total=%d, got %d", expectedTotal, response.Total)

	return nil
}

// Helper function to parse status string to DeploymentStatus
func parseStatus(statusStr string) (jobtypes.DeploymentStatus, error) {
	statusStr = strings.TrimSpace(statusStr)
	for i := jobtypes.DeploymentStatusPreparing; i <= jobtypes.DeploymentStatusCompleted; i++ {
		if strings.EqualFold(i.String(), statusStr) {
			return i, nil
		}
	}
	return 0, fmt.Errorf("unknown status: %s", statusStr)
}
