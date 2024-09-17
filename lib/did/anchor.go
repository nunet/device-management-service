package did

type GetAnchorFunc func(did DID) (Anchor, error)

var anchorMethods map[string]GetAnchorFunc

func init() {
	anchorMethods = map[string]GetAnchorFunc{
		"key": makeKeyAnchor,
	}
}

func GetAnchorForDID(did DID) (Anchor, error) {
	makeAnchor, ok := anchorMethods[did.Method()]
	if !ok {
		return nil, ErrNoAnchorMethod
	}

	return makeAnchor(did)
}

func makeKeyAnchor(did DID) (Anchor, error) {
	pubk, err := PublicKeyFromDID(did)
	if err != nil {
		return nil, err
	}

	return NewAnchor(did, pubk), nil
}
