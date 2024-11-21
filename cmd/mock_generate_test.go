package cmd

//go:generate mockgen -destination=mock_docker_client_test.go -source=../executor/docker/client.go -package=cmd

//go:generate mockgen -destination=mock_hardware_manager_test.go -source=../types/hardware.go -package=cmd
