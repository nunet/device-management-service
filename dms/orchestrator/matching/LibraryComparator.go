package matching

import (
	"github.com/hashicorp/go-version"
	"gitlab.com/nunet/device-management-service/types"
)

func LibraryComparator(lraw, rraw interface{}, _ ...Preference) types.Comparison {
	// comparator for single Library type:
	// left represent machine capabilities;
	// right represent required capabilities;

	// validate input type
	_, lrawok := lraw.(types.Library)
	_, rrawok := rraw.(types.Library)
	if !lrawok || !rrawok {
		return types.Error
	}

	l := lraw.(types.Library)
	lVersion, err := version.NewVersion(l.Version)
	if err != nil {
		return types.Error
	}
	r := rraw.(types.Library)

	// return 'Error' if the version of the left library is not valid
	constraints, err := version.NewConstraint(r.Constraint + " " + r.Version)
	if err != nil {
		return types.Error
	}

	// return 'Error' if the names of the libraries are different
	if l.Name != r.Name {
		return types.Error
	}

	// else return 'Equal if versions of libraries are equal and the constraint is '='
	if r.Constraint == "=" && constraints.Check(lVersion) {
		return types.Equal
	}

	// else return 'Better' if versions of libraries match the constraint
	if constraints.Check(lVersion) {
		return types.Better
	}

	// else return 'Worse'
	return types.Worse
}
