package api

//go:generate mockgen -destination=mock_network_test.go -source=../network/network.go -package=api -exclude_interfaces=Messenger
