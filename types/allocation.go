package types

func ConstructAllocationID(ensembleID, allocName string) string {
	return ensembleID + "_" + allocName
}
