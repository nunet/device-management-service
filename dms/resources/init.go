package resources

import (
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"gitlab.com/nunet/device-management-service/db"
	gormRepo "gitlab.com/nunet/device-management-service/db/repositories/gorm"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

var (
	// zlog is the logger for the resources package
	zlog *otelzap.Logger
	// ManagerInstance is the ResourceManager instance
	ManagerInstance Manager
)

// TODO: This needs to be initialized in `dms` package and removed from here
// https://gitlab.com/nunet/device-management-service/-/issues/536
// it is being initialized in `dms` package now but there is usage in executor
// in executor/docker/executor.go:262:25 in function newDockerExecutionContainer
// which heavily depends on this var and any attempt to fix it will involve
// too many changes. Once that code moves to allocations, this can be removed.
func init() {
	zlog = logger.OtelZapLogger("resources")

	repos := ManagerRepos{
		FreeResources:      gormRepo.NewFreeResources(db.DB),
		OnboardedResources: gormRepo.NewOnboardedResources(db.DB),
		RequiredResources:  gormRepo.NewRequiredResources(db.DB),
		VirtualMachine:     gormRepo.NewVirtualMachine(db.DB),
		Services:           gormRepo.NewServices(db.DB),
	}
	ManagerInstance = NewResourceManager(repos)
}
