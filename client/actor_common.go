package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
)

// GetDMSHandle retrieves the DMS handle from the server
func (c *Client) GetDMSHandle(ctx context.Context) (actor.Handle, error) {
	if !c.dmsHandle.Empty() {
		return c.dmsHandle, nil
	}

	err := c.get(ctx, "/actor/handle", nil, &c.dmsHandle)
	if err != nil {
		return actor.Handle{}, fmt.Errorf("get source handle: %w", err)
	}

	return c.dmsHandle, nil
}

// parseDestinationHandle parses a destination string into a handle
func (c *Client) parseDestinationHandle(destStr string) (actor.Handle, error) {
	// Input validation
	if destStr == "" {
		return actor.Handle{}, fmt.Errorf("empty destination string")
	}

	// First check if it's a DID (starts with "did:")
	if strings.HasPrefix(destStr, "did:") {
		dest, err := actor.HandleFromDID(destStr)
		if err != nil {
			return actor.Handle{}, fmt.Errorf("failed to parse DID handle: %w", err)
		}
		return dest, nil
	}

	// Try to parse as JSON handle - don't return if it fails
	var jsonDest actor.Handle
	if err := json.Unmarshal([]byte(destStr), &jsonDest); err == nil {
		// Successfully parsed as JSON
		return jsonDest, nil
	}

	// Default: try to parse as a peer ID
	dest, err := actor.HandleFromPeerID(destStr)
	if err != nil {
		return actor.Handle{}, fmt.Errorf("failed to parse peer ID handle: %w", err)
	}

	return dest, nil
}

// newUserHandle creates a new user handle
func (c *Client) newUserHandle(id crypto.ID, userDID did.DID, dmsHandle actor.Handle, inbox string) actor.Handle {
	return actor.Handle{
		ID:  id,
		DID: userDID,
		Address: actor.Address{
			HostID:       dmsHandle.Address.HostID,
			InboxAddress: inbox,
		},
	}
}

// newClient creates a new client
func (c *Client) unmarshalResponse(resp actor.Envelope, v any) error {
	if resp.Message == nil {
		return nil
	}

	if err := json.Unmarshal(resp.Message, v); err != nil {
		return fmt.Errorf("unmarshal response: %w", err)
	}

	return nil
}
