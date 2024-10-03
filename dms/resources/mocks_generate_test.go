package resources

// Currently we need explicitly exclude interfaces from the mock generation
// TODO: Change it after https://github.com/uber-go/mock/pull/200 is merged

// UsageMonitor
//go:generate mockgen -destination=mock_usage_monitor_test.go -source=../../types/resource.go -package=resources -exclude_interfaces=ResourceManager,ResourceOps
// DB Repositories
//go:generate mockgen -source=../../db/repositories/generic_repository.go -destination mock_generic_repository_test.go -package=resources
//go:generate mockgen -source=../../db/repositories/generic_entity_repository.go -destination mock_generic_entity_repository_test.go -package=resources
