package matching

import (
	"gitlab.com/nunet/device-management-service/models"
	"github.com/hashicorp/go-version"
)

func LibraryComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison {
	// comparator for single Library type:
	// left represent machine capabilities;
	// right represent required capabilities;

	l := lraw.(models.Library)
	lVersion, err := version.NewVersion(l.Version)
	if err != nil {
		return models.Error
	}
	r := rraw.(models.Library)

	constraints, err := version.NewConstraint(r.Constraint + " " + r.Version)
	if err != nil {	
		return models.Error
	}
	
	if l.Name != r.Name {
		return models.Error
	} 	
	if r.Constraint == "=" && constraints.Check(lVersion) {
		return models.Equal
	}
	if constraints.Check(lVersion) {
		return models.Better
	}
	return models.Worse
}