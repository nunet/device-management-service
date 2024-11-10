// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package dms

import (
	"gorm.io/gorm"
)

// SanityCheck before being deleted performed basic consistency checks before starting the DMS
// in the following sequence:
// It checks for services that are marked running from the database and stops then removes them.
// Update their status to 'finshed with errors'.
// Recalculates free resources and update the database.
//
// Deleted now because dependencies such as the docker package have been replaced with executor/docker
func SanityCheck(_ *gorm.DB) {
	// TODO: sanity check of DMS last exit and correction of invalid states

	// resources.CalcFreeResAndUpdateDB()
}
