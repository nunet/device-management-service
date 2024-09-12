package node

// Currently we need explicitly exclude interfaces from the mock generation
// TODO: Change it after https://github.com/uber-go/mock/pull/200 is merged

// Resource Manager
//go:generate mockgen -destination=mock_resource_manager_test.go -source=../../types/resources.go -package=node -exclude_interfaces=UsageMonitor,SystemSpecs,ResourceOps
