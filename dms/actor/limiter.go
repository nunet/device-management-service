package actor

type BasicDispatchLimiter struct {
	// TODO we can leave this for follow up
}

func (l *BasicDispatchLimiter) Reserve(_ Envelope) error {
	// TODO we can leave this for follow up
	return ErrTODO
}

func (l *BasicDispatchLimiter) Release(_ Envelope) {
	// TODO we can leave this for follow up
}
