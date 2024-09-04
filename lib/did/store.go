package did

import (
	"io"
)

func SaveTrustContext(_ TrustContext, _ io.Writer) (int, error) {
	// TODO follow up
	return 0, ErrTODO
}

func LoadTrustContext(_ io.Reader) (TrustContext, error) {
	// TODO follow up
	return nil, ErrTODO
}
