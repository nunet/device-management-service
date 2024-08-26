package types

type Comparison string

const (
	Worse  Comparison = "Worse"  // left object is 'worse' than right object
	Better Comparison = "Better" // left object is 'better' than right object
	Equal  Comparison = "Equal"  // objects on the left and right are 'equally good'
	Error  Comparison = "Error"  // error in comparison or objects incomparable
)

// TODO: Consider comments in this thread: https://gitlab.com/nunet/device-management-service/-/merge_requests/356#note_1997854443
// TODO: Consider comments in this thread: https://gitlab.com/nunet/device-management-service/-/merge_requests/356#note_1997875361

// 'left' means 'this object' and 'right' means 'the supplied other object';
// it makes sense when using the type in functions like Compare(left, right)

// this type is unused but still reserved in case we will need it in the future
// and still used in some tests that are left from previous versions of the package;
// TOTO: remove / update after the package is finished and refactor
type ComplexComparison map[string]Comparison

// And returns the result of AND operation of two Comparison values
// it respects the following table of truth:
// |   AND  | Better |  Worse |  Equal |  Error |
// | ------ | ------ |--------|--------|--------|
// | Better | Better |  Worse | Better |  Error |
// | Worse  | Worse  |  Worse | Worse  |  Error |
// | Equal  | Better |  Worse | Equal  |  Error |
// | Error  | Error  |  Error | Error  |  Error |
func (c Comparison) And(cmp Comparison) Comparison {
	if c == Error || cmp == Error {
		return Error
	}

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
		default:
			return Error
		}

	case Better:
		switch cmp {
		case Worse:
			return Worse
		case Equal:
			return Better
		default:
			return Error
		}
	case Worse:
		if cmp == Error {
			return Error
		}
		return Worse

	default:
		return Error
	}
}
