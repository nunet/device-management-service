package resources

// DB Repositories
//go:generate mockgen -source=../../db/repositories/generic_repository.go -destination mock_generic_repository_test.go -package=resources
//go:generate mockgen -source=../../db/repositories/generic_entity_repository.go -destination mock_generic_entity_repository_test.go -package=resources
//go:generate mockgen -destination=mock_hardware_manager_test.go -source=../../types/hardware.go -package=resources
