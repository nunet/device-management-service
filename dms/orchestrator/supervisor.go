// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	RegisterHealthCheckTimeout = 5 * time.Second
)

var (
	HealthCheckTimeout       = 5 * time.Second
	FailureEscalationTimeout = 2 * time.Minute
)

// Supervisor encapsulates supervision logic.
type Supervisor struct {
	id       string
	ctx      context.Context
	actor    actor.Actor
	manifest jtypes.EnsembleManifest

	registeredHealthChecks map[string]struct{} // allocationID -> struct{}
	failures               map[string]int      // allocationID -> failureCount
	escalations            map[string]int      // allocationID -> escalationCount
	lock                   sync.Mutex
}

// NewSupervisor creates a new Supervisor instance.
func NewSupervisor(ctx context.Context, actor actor.Actor, id string) *Supervisor {
	return &Supervisor{
		ctx:                    ctx,
		actor:                  actor,
		id:                     id,
		failures:               make(map[string]int),
		escalations:            make(map[string]int),
		registeredHealthChecks: make(map[string]struct{}),
		manifest: jtypes.EnsembleManifest{
			ID:          id,
			Allocations: make(map[string]jtypes.AllocationManifest),
			Nodes:       make(map[string]jtypes.NodeManifest),
		},
	}
}

// Supervise runs the supervision loop, including registration and periodic healthchecks.
func (s *Supervisor) Supervise(manifestReader jtypes.ManifestReader) {
	log.Debugw("supervisor started for orchestrator",
		"labels", string(observability.LabelDeployment),
		"supervisorID", s.id,
		"allocations", s.manifest.Allocations)

	manifest := manifestReader.Read()

	wg := sync.WaitGroup{}
	// Registration Phase – register allocations that have a defined healthcheck.
	for _, allocation := range manifest.Allocations {
		if allocation.Healthcheck.Type == "" {
			continue
		}

		wg.Add(1)
		go func(allocation jtypes.AllocationManifest) {
			defer wg.Done()

			if err := s.registerHealthCheck(allocation, manifest.Orchestrator); err != nil {
				log.Errorf("register healthcheck for allocation: %s", err)
			}
		}(allocation)
	}
	wg.Wait()

	// Update the manifest
	s.lock.Lock()
	s.manifest = manifest
	s.lock.Unlock()

	// Supervision Phase – start the supervision loop
	s.startSupervision()
}

// startSupervision performs periodic health checks on registered allocations.
func (s *Supervisor) startSupervision() {
	ticker := time.NewTicker(actor.HealthCheckInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			wg := sync.WaitGroup{}
			for allocationID := range s.registeredHealthChecks {
				fmt.Println("allocationID", allocationID)

				// Parse the allocation ID to get the manifest key
				allocID, err := types.ParseAllocationID(allocationID)
				if err != nil {
					log.Warnf("failed to parse allocation ID %s: %v", allocationID, err)
					continue
				}

				manifestKey := allocID.ManifestKey()
				allocation, ok := s.getAllocation(manifestKey)
				if !ok {
					log.Warnf("allocation not found in manifest to supervise: %s", allocationID)
					continue
				}
				if allocation.Healthcheck.Type == "" {
					log.Debugf("allocation does not have a healthcheck: %s", allocationID)
					continue
				}

				wg.Add(1)
				go func(allocation jtypes.AllocationManifest) {
					defer wg.Done()
					if err := s.performHealthCheck(allocation); err != nil {
						log.Errorf("failed to perform healthcheck for allocation %s: %s", allocation.ID, err)
					}
				}(allocation)
				wg.Wait()
			}
		}
	}
}

func (s *Supervisor) registerHealthCheck(allocation jtypes.AllocationManifest, orchestrator actor.Handle) error {
	expiry := actor.MakeExpiry(RegisterHealthCheckTimeout)
	msg, err := actor.Message(
		orchestrator,
		allocation.Handle,
		behaviors.RegisterHealthcheckBehavior,
		behaviors.RegisterHealthcheckRequest{
			EnsembleID:  s.id,
			HealthCheck: allocation.Healthcheck,
		},
		actor.WithMessageExpiry(expiry),
	)
	if err != nil {
		return fmt.Errorf("create actor message: %w", err)
	}

	replyCh, err := s.actor.Invoke(msg)
	if err != nil {
		return fmt.Errorf("register healthcheck on allocation: %w", err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
		defer reply.Discard()

		var response behaviors.RegisterHealthcheckResponse
		if err := json.Unmarshal(reply.Message, &response); err != nil {
			return fmt.Errorf("unmarshalling supervisor reply: %w", err)
		}

		if !response.OK {
			return fmt.Errorf("error registering healthcheck: %s", response.Error)
		}

		s.lock.Lock()
		s.registeredHealthChecks[allocation.ID] = struct{}{}
		s.lock.Unlock()
		log.Infof("successfully registered healthcheck for allocation: %s", allocation.ID)
		return nil

	case <-time.After(RegisterHealthCheckTimeout):
		return fmt.Errorf("timeout waiting for supervisor reply")
	}
}

func (s *Supervisor) unregisterHealthCheck(allocationID string) {
	s.lock.Lock()
	delete(s.registeredHealthChecks, allocationID)
	s.lock.Unlock()
}

func (s *Supervisor) performHealthCheck(allocation jtypes.AllocationManifest) error {
	// Parse the allocation ID to get the manifest key for termination check
	allocID, err := types.ParseAllocationID(allocation.ID)
	if err != nil {
		log.Warnf("failed to parse allocation ID %s: %v", allocation.ID, err)
		return err
	}

	manifestKey := allocID.ManifestKey()
	if s.manifest.IsTerminatedTask(manifestKey) {
		return nil
	}
	expiry := actor.MakeExpiry(HealthCheckTimeout)
	msg, err := actor.Message(
		s.actor.Handle(),
		allocation.Handle,
		actor.HealthCheckBehavior,
		allocation.ID,
		actor.WithMessageExpiry(expiry),
	)
	if err != nil {
		return fmt.Errorf("create supervisor message: %w", err)
	}

	replyCh, err := s.actor.Invoke(msg)
	if err != nil {
		return fmt.Errorf("invoke healthcheck on allocation: %w", err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
		defer reply.Discard()

		var resp behaviors.HealthCheckResponse
		if err := json.Unmarshal(reply.Message, &resp); err != nil {
			return fmt.Errorf("unmarshalling supervisor reply: %w", err)
		}

		if !resp.OK {
			log.Errorf("error in healthcheck for allocation %s: %s", allocation.ID, resp.Error)

			s.lock.Lock()
			s.failures[allocation.ID]++
			failureCount := s.failures[allocation.ID]
			s.lock.Unlock()
			if failureCount >= 3 {
				log.Infof("escalating failure for allocation %s", allocation.ID)
				if err := s.escalateFailure(allocation); err != nil {
					log.Errorf("failed to escalate failure: %s", err)
				} else {
					log.Debug("escalated failure, resetting healthcheck failures counter")
					s.lock.Lock()
					delete(s.failures, allocation.ID)
					s.lock.Unlock()
				}
			}
		} else {
			log.Infof("successfully healthchecked allocation %s", allocation.ID)
			s.lock.Lock()
			delete(s.failures, allocation.ID)
			s.lock.Unlock()
		}
	case <-time.After(HealthCheckTimeout):
		if s.manifest.IsTerminatedTask(manifestKey) {
			return nil
		}

		log.Warnf("timeout waiting for supervisor reply for allocation %s", allocation.ID)
		s.lock.Lock()
		s.failures[allocation.ID]++
		v := s.failures[allocation.ID]
		s.lock.Unlock()
		if v >= 3 {
			if err := s.escalateFailure(allocation); err != nil {
				log.Errorf("failed to escalate failure: %s", err)
			} else {
				log.Debug("escalated failure, resetting healthcheck failures counter")
				s.lock.Lock()
				delete(s.failures, allocation.ID)
				s.lock.Unlock()
			}
		}
		return fmt.Errorf("timeout waiting for supervisor reply")
	}

	return nil
}

// escalateFailure handles escalation when an allocation repeatedly fails its healthcheck.
func (s *Supervisor) escalateFailure(allocation jtypes.AllocationManifest) error {
	// TODO we need to decide how to handle repeated failures and also correlated failures
	//      from a node.
	//      Also, we should not restart at first failure, but wait for a number of
	//      consecutive failures.
	//      See https://gitlab.com/nunet/device-management-service/-/issues/794
	log.Debugf("escalating failure for allocation %s", allocation.Handle.String())
	log.Debugw("escalating failure for allocation",
		"labels", string(observability.LabelAllocation),
		"allocationHandle", allocation.Handle.String(),
		"supervisorID", s.id)
	expiry := actor.MakeExpiry(5 * time.Second)
	msg, err := actor.Message(
		s.actor.Handle(),
		allocation.Handle,
		behaviors.AllocationRestartBehavior,
		allocation.ID,
		actor.WithMessageExpiry(expiry),
	)
	if err != nil {
		return err
	}

	replyCh, err := s.actor.Invoke(msg)
	if err != nil {
		return err
	}

	select {
	case reply := <-replyCh:
		defer reply.Discard()

		var resp behaviors.AllocationRestartResponse
		if err := json.Unmarshal(reply.Message, &resp); err != nil {
			return fmt.Errorf("unmarshalling supervisor reply: %w", err)
		}

		if !resp.OK {
			return fmt.Errorf("error restarting allocation: %s", resp.Error)
		}

		s.lock.Lock()
		defer s.lock.Unlock()
		s.escalations[allocation.ID]++
		s.failures[allocation.ID] = 0
		return nil
	case <-time.After(FailureEscalationTimeout):
		return fmt.Errorf("timeout waiting for supervisor reply")
	}
}

// Update updates the supervisor with a new ensemble manifest.
//
//  1. register new healthchecks and start healthchecking
//  2. unregister healthchecks for removed allocations
//  3. update manifest
//
// For 1. and 2., the updates will already be reflected on
// the next ticker.
func (s *Supervisor) Update(manifestReader jtypes.ManifestReader) {
	manifest := manifestReader.Read()
	log.Debug("updating supervisor")

	// 1. Registering the healthchecks for just the new allocations
	var wg sync.WaitGroup
	for _, allocation := range manifest.Allocations {
		if allocation.Healthcheck.Type == "" {
			continue
		}

		// register healthcheck only if it is not already present
		if _, ok := s.getAllocation(types.AllocationNameFromID(allocation.ID)); !ok {
			wg.Add(1)
			go func(allocation jtypes.AllocationManifest) {
				defer wg.Done()

				if err := s.registerHealthCheck(allocation, manifest.Orchestrator); err != nil {
					log.Errorf("failed to register healthcheck for allocation: %s", err)
				}
			}(allocation)
		} else {
			// 2. unregister healthcheck if it is no longer present
			s.unregisterHealthCheck(allocation.ID)
		}
	}

	wg.Wait()

	// 3. Update the manifest
	s.lock.Lock()
	s.manifest = manifest
	s.lock.Unlock()
}

func (s *Supervisor) getAllocation(name string) (jtypes.AllocationManifest, bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	a, ok := s.manifest.Allocations[name]
	return a, ok
}
