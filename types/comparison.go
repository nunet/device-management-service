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

type ComplexComparison map[string]Comparison

// this type is unused but still reserved in case we will need it in the future
// and still used in some tests that are left from previous versions of the package;
// TOTO: remove / update after the package is finished and refactor
