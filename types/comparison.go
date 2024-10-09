package types

// Comparison is a type for comparison results
type Comparison string

const (
	// Worse means left object is 'worse' than right object
	Worse Comparison = "Worse"
	// Better means left object is 'better' than right object
	Better Comparison = "Better"
	// Equal means objects on the left and right are 'equally good'
	Equal Comparison = "Equal"
	// None means comparison could not be performed
	None Comparison = "None"
)

// And returns the result of AND operation of two Comparison values
// it respects the following table of truth:
// |   AND  | Better |  Worse |  Equal |  None  |
// | ------ | ------ |--------|--------|--------|
// | Better | Better |  Worse | Better |  None  |
// | Worse  | Worse  |  Worse | Worse  |  None  |
// | Equal  | Better |  Worse | Equal  |  None  |
// | None   | None   |  None  | None   |  None  |
func (c Comparison) And(cmp Comparison) Comparison {
	if c == cmp {
		return c
	}

	switch c {
	case Equal:
		switch cmp {
		case Better:
			return Better
		case Worse:
			return Worse
		case Equal:
			return Equal
		default:
			return None
		}

	case Better:
		switch cmp {
		case Worse:
			return Worse
		case Equal:
			return Better
		default:
			return None
		}
	case Worse:
		return Worse

	default:
		return None
	}
}

// ComplexComparison is a map of string to Comparison
type ComplexComparison map[string]Comparison

// Result returns the result of AND operation of all Comparison values in the ComplexComparison
func (c *ComplexComparison) Result() Comparison {
	result := Equal
	for _, comparison := range *c {
		result = result.And(comparison)
	}
	return result
}
