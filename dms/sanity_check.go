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
