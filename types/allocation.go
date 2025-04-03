package types

import (
	"strings"
)

func ConstructAllocationID(ensembleID, allocName string) string {
	return ensembleID + "_" + allocName
}

func AllocationNameFromID(id string) string {
	return id[strings.LastIndex(id, "_")+1:]
}
