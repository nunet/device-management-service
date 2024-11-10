// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package resources

// DB Repositories
//go:generate mockgen -source=../../db/repositories/generic_repository.go -destination mock_generic_repository_test.go -package=resources
//go:generate mockgen -source=../../db/repositories/generic_entity_repository.go -destination mock_generic_entity_repository_test.go -package=resources
//go:generate mockgen -destination=mock_hardware_manager_test.go -source=../../types/hardware.go -package=resources
