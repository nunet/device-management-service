// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package client

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

// NewMessage creates a new actor message with the specified behavior and payload
func (c *Client) NewActorMessage(ctx context.Context, behavior string, payload any, msgOpts MessageOptions) (actor.Envelope, error) {
	dmsHandle, err := c.GetDMSHandle(ctx)
	if err != nil {
		return actor.Envelope{}, fmt.Errorf("get DMS handle: %w", err)
	}

	// Create user handle
	nonce := c.sctx.Nonce()
	inbox := fmt.Sprintf("user-%d", nonce)
	src := c.newUserHandle(c.sctx.ID(), c.sctx.DID(), dmsHandle, inbox)

	// Handle destination
	var dest actor.Handle

	opts := []actor.MessageOption{}
	replyTo := ""

	// Configure behavior based on message type
	switch {
	case msgOpts.Topic != "":
		opts = append(opts, actor.WithMessageTopic(msgOpts.Topic))
		replyTo = fmt.Sprintf("/public/user/%d", nonce)
	case msgOpts.Destination != "":
		// Parse destination string into a handle
		var err error
		dest, err = c.parseDestinationHandle(msgOpts.Destination)
		if err != nil {
			return actor.Envelope{}, fmt.Errorf("create destination handle: %w", err)
		}
	default:
		dest = dmsHandle
	}

	// Handle invocation flag
	if msgOpts.IsInvocation {
		replyTo = fmt.Sprintf("/private/user/%d", nonce)
	}

	// Handle expiry
	if !msgOpts.Expiry.IsZero() {
		opts = append(opts, actor.WithMessageExpiry(uint64(msgOpts.Expiry.UnixNano())))
	}

	// Handle timeout
	if msgOpts.Timeout > 0 {
		opts = append(opts, actor.WithMessageTimeout(msgOpts.Timeout))
	}

	// TODO: Do we delegate capabilities here?
	// Handle reply address
	// delegate := []ucan.Capability{}
	if replyTo != "" || msgOpts.ReplyTo != "" {
		if msgOpts.ReplyTo != "" {
			replyTo = msgOpts.ReplyTo
		}
		opts = append(opts, actor.WithMessageReplyTo(replyTo))
		// if msgOpts.Topic == "" {
		// 	delegate = append(delegate, ucan.Capability(replyTo))
		// }
	}

	// Add message signature
	opts = append(opts, actor.WithMessageSignature(c.sctx, []ucan.Capability{ucan.Capability(behavior)}, []ucan.Capability{}))

	// Create the message
	msg, err := actor.Message(src, dest, behavior, payload, opts...)
	if err != nil {
		return actor.Envelope{}, fmt.Errorf("construct message: %w", err)
	}

	return msg, nil
}

// SendMessage sends a message to a specific actor
func (c *Client) SendMessage(ctx context.Context, behavior string, payload any, msgOpts ...Option) (actor.Envelope, error) {
	// Create message
	opts := NewMessageOptions(msgOpts...)

	msg, err := c.NewActorMessage(ctx, behavior, payload, opts)
	if err != nil {
		return actor.Envelope{}, fmt.Errorf("create actor message: %w", err)
	}

	// Send message
	return c.SendMessageRaw(ctx, msg)
}

// SendMessageRaw sends a message to a specific actor
func (c *Client) SendMessageRaw(ctx context.Context, msg actor.Envelope) (actor.Envelope, error) {
	// Send message
	var response actor.Envelope
	err := c.post(ctx, ActorSendMessageEndpoint, nil, msg, &response)
	if err != nil {
		return response, fmt.Errorf("send message: %w", err)
	}

	return response, nil
}

// InvokeBehavior invokes a behavior on an actor
func (c *Client) InvokeBehavior(ctx context.Context, behavior string, payload any, msgOpts ...Option) (actor.Envelope, error) {
	// Create message
	opts := NewMessageOptions(msgOpts...)
	opts.IsInvocation = true

	// Apply timeout to context if specified
	if opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
		defer cancel()
	}

	msg, err := c.NewActorMessage(ctx, behavior, payload, opts)
	if err != nil {
		return actor.Envelope{}, fmt.Errorf("create actor message: %w", err)
	}

	// Invoke behavior
	return c.InvokeBehaviorRaw(ctx, msg)
}

// InvokeBehaviorRaw invokes a behavior on an actor
func (c *Client) InvokeBehaviorRaw(ctx context.Context, msg actor.Envelope) (actor.Envelope, error) {
	// Invoke behavior
	var response actor.Envelope
	err := c.post(ctx, ActorInvokeEndpoint, nil, msg, &response)
	if err != nil {
		return response, fmt.Errorf("invoke behavior: %w", err)
	}

	return response, nil
}

// BroadcastMessage broadcasts a message to a topic
func (c *Client) BroadcastMessage(ctx context.Context, behavior, topic string, payload any, msgOpts ...Option) ([]actor.Envelope, error) {
	opts := NewMessageOptions(msgOpts...)
	opts.Topic = topic

	// Verify that a topic is provided
	if opts.Topic == "" {
		return nil, fmt.Errorf("broadcast requires a topic")
	}

	msg, err := c.NewActorMessage(ctx, behavior, payload, opts)
	if err != nil {
		return nil, fmt.Errorf("create actor message: %w", err)
	}

	// Broadcast message
	return c.BroadcastMessageRaw(ctx, msg)
}

// BroadcastMessageRaw broadcasts a message to a topic
func (c *Client) BroadcastMessageRaw(ctx context.Context, msg actor.Envelope) ([]actor.Envelope, error) {
	// Broadcast message
	var responses []actor.Envelope
	err := c.post(ctx, ActorBroadcastEndpoint, nil, msg, &responses)
	if err != nil {
		return nil, fmt.Errorf("broadcast message: %w", err)
	}

	return responses, nil
}
