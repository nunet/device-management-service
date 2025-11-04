package steps

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
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
	ctx.After(func(ctx context.Context, _ *godog.Scenario, _ error) (context.Context, error) {
		if err := hooks.SaveLogs(ctx); err != nil {
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
}

func hasDeployments(ctx context.Context, spName string, count int, ensemble, status, cpName string) (context.Context, error) {
	t := godog.T(ctx)
	tc := utils.NewTestCtx(ctx)

	switch ensemble {
	case "task":
		ensemble = "docker_hello.yaml"
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
