package dms

// ActorRegistry represents an actor registry.
type ActorRegistry struct {
	actors map[string]*actorInfo
}

type actorInfo struct {
	addrInfo *ActorAddrInfo
	parent   *ActorAddrInfo
	childs   []*ActorAddrInfo
}

// NewActorRegistry creates an actor registry.
func NewActorRegistry() *ActorRegistry {
	return &ActorRegistry{
		actors: make(map[string]*actorInfo),
	}
}

// AddActorAddress adds an actor address to the registry.
func (r *ActorRegistry) AddActorAddress(a *ActorAddrInfo) {
	_, ok := r.actors[a.InboxAddress]
	if ok {
		return
	}

	r.actors[a.InboxAddress] = &actorInfo{
		addrInfo: a,
		parent:   nil,
		childs:   make([]*ActorAddrInfo, 0),
	}
}

// SetParentAddress sets parent address of an actor.
func (r *ActorRegistry) SetParentAddress(actorID string, parent *ActorAddrInfo) {
	actor, ok := r.actors[actorID]
	if !ok {
		return
	}

	actor.parent = parent
	r.actors[actorID] = actor
}

// AddChild adds a child to an actor.
func (r *ActorRegistry) AddChild(actorID string, child *ActorAddrInfo) {
	actor, ok := r.actors[actorID]
	if !ok {
		return
	}

	actor.childs = append(actor.childs, child)
	r.actors[actorID] = actor
}

func (r *ActorRegistry) GetActorAddress(address string) (*ActorAddrInfo, bool) {
	a, ok := r.actors[address]
	return a.addrInfo, ok
}

func (r *ActorRegistry) GetParentAddress(address string) (*ActorAddrInfo, bool) {
	a, ok := r.actors[address]
	if !ok {
		return nil, false
	}

	return a.parent, true
}
