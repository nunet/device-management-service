// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

// Currently we need explicitly exclude interfaces from the mock generation
// TODO: Change it after https://github.com/uber-go/mock/pull/200 is merged

// Resource Manager
//go:generate mockgen -destination=mock_resource_manager_test.go -source=../../types/resources.go -package=node -exclude_interfaces=UsageMonitor,ResourceOps
// Hardware Manager
//go:generate mockgen -destination=mock_hardware_manager_test.go -source=../../types/hardware.go -package=node
// Network
//go:generate mockgen -destination=mock_network_test.go -source=../../network/network.go -package=node -exclude_interfaces=Messenger
// GeoIPLocator
//go:generate mockgen -destination=mock_geoip_locator_test.go -source=../../types/types.go -package=node
// Actor
//go:generate mockgen -destination=mock_actor_test.go -source=../../actor/interface.go -package=node
// Orchestrator
//go:generate mockgen -destination=mock_orchestrator_test.go -source=../jobs/orchestrator.go -package=node
