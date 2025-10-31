package utils

import "fmt"

type Node struct {
	Name      string
	userCtx   *Context
	dmsCtx    *Context
	Role      string
	Org       string
	Onboarded bool
	Instance  *Instance
}

func NewNode(name, role, org string, onboarded bool, i *Instance) *Node {
	return &Node{
		Name:      name,
		Role:      role,
		Org:       org,
		Onboarded: onboarded,
		Instance:  i,
	}
}

func (n *Node) User() *Context {
	return n.userCtx
}

func (n *Node) DMS() *Context {
	return n.dmsCtx
}

// InitialCaps creates user and DMS contexts with proper capabilities
func (n *Node) InitialCaps() error {
	userCtx, err := CreateContext(n.Instance, n.Name)
	if err != nil {
		return fmt.Errorf("failed to create user ctx: %w", err)
	}
	dmsCtx, err := CreateContext(n.Instance, n.Name+DefaultDMSSuffix)
	if err != nil {
		return fmt.Errorf("failed to create dms ctx: %w", err)
	}

	n.dmsCtx = dmsCtx
	n.userCtx = userCtx

	err = dmsCtx.Anchor("root", userCtx.DID)
	if err != nil {
		return fmt.Errorf("failed to anchor user as root: %w", err)
	}

	return nil
}
