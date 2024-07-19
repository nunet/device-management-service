package matching

import (
	"gitlab.com/nunet/device-management-service/models"
	"github.com/hashicorp/go-version"
)

func LibraryComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison {
	// comparator for single Library type:
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(models.Library)
	_, rrawok := rraw.(models.Library)
	if !lrawok || !rrawok {
		return models.Error
	}	

	l := lraw.(models.Library)
	lVersion, err := version.NewVersion(l.Version)
	if err != nil {
		return models.Error
	}
	r := rraw.(models.Library)

	// return 'Error' if the version of the left library is not valid
	constraints, err := version.NewConstraint(r.Constraint + " " + r.Version)
	if err != nil {	
		return models.Error
	}
	
	// return 'Error' if the names of the libraries are different
	if l.Name != r.Name {
		return models.Error
	}
	
	// else return 'Equal if versions of libraries are equal and the constraint is '='
	if r.Constraint == "=" && constraints.Check(lVersion) {
		return models.Equal
	}

	// else return 'Better' if versions of libraries match the constraint
	if constraints.Check(lVersion) {
		return models.Better
	}

	// else return 'Worse'
	return models.Worse
}